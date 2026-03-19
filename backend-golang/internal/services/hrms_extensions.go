package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HRMSWebhookEventPublic struct {
	ID          uuid.UUID `json:"id"`
	Status      string    `json:"status"`
	RetryCount  int       `json:"retry_count"`
	Error       *string   `json:"error_message,omitempty"`
	NextRetryAt *string   `json:"next_retry_at,omitempty"`
	CreatedAt   string    `json:"created_at"`
	ProcessedAt *string   `json:"processed_at,omitempty"`
}

type HRMSSyncConflictPublic struct {
	ID               uuid.UUID `json:"id"`
	ExternalRecordID string    `json:"external_record_id"`
	FieldName        string    `json:"field_name"`
	LocalValue       any       `json:"local_value"`
	ExternalValue    any       `json:"external_value"`
	Status           string    `json:"status"`
	CreatedAt        string    `json:"created_at"`
	ResolvedAt       *string   `json:"resolved_at,omitempty"`
}

func (s *HRMSService) GetDB() *pgxpool.Pool {
	return s.db
}

func generateRandomHex(length int) (string, error) {
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func readMappedValue(payload map[string]interface{}, path string) interface{} {
	parts := strings.Split(strings.TrimSpace(path), ".")
	var current interface{} = payload
	for _, part := range parts {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = obj[part]
	}
	return current
}

func normalizeFieldMappings(config map[string]interface{}, override []map[string]string) []map[string]string {
	if len(override) > 0 {
		return override
	}
	if config == nil {
		return nil
	}
	raw, ok := config["field_mapping"]
	if !ok {
		return nil
	}
	rows, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]map[string]string, 0, len(rows))
	for _, item := range rows {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, map[string]string{
			"source": strings.TrimSpace(fmt.Sprint(obj["source"])),
			"target": strings.TrimSpace(fmt.Sprint(obj["target"])),
		})
	}
	return result
}

func (s *HRMSService) mapFields(payload map[string]interface{}, config map[string]interface{}, override []map[string]string) (map[string]interface{}, []string) {
	mappings := normalizeFieldMappings(config, override)
	mapped := map[string]interface{}{}
	missing := []string{}
	if len(mappings) == 0 {
		return mapped, missing
	}
	for _, row := range mappings {
		source := strings.TrimSpace(row["source"])
		target := strings.TrimSpace(row["target"])
		if source == "" || target == "" {
			continue
		}
		value := readMappedValue(payload, source)
		if value == nil || fmt.Sprint(value) == "" {
			missing = append(missing, source)
			continue
		}
		mapped[target] = value
	}
	return mapped, missing
}

func (s *HRMSService) listFailedWebhookEvents(ctx context.Context, tenantID, integrationID string) ([]HRMSWebhookEventPublic, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, status, retry_count, error_message, next_retry_at, created_at, processed_at
		FROM hrms_webhook_events
		WHERE tenant_id = $1 AND integration_id = $2
		ORDER BY created_at DESC
		LIMIT 25
	`, tenantID, integrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []HRMSWebhookEventPublic{}
	for rows.Next() {
		var item HRMSWebhookEventPublic
		var nextRetryAt *time.Time
		var createdAt time.Time
		var processedAt *time.Time
		if err := rows.Scan(&item.ID, &item.Status, &item.RetryCount, &item.Error, &nextRetryAt, &createdAt, &processedAt); err != nil {
			return nil, err
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if nextRetryAt != nil {
			value := nextRetryAt.UTC().Format(time.RFC3339)
			item.NextRetryAt = &value
		}
		if processedAt != nil {
			value := processedAt.UTC().Format(time.RFC3339)
			item.ProcessedAt = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *HRMSService) listSyncConflicts(ctx context.Context, tenantID, integrationID string) ([]HRMSSyncConflictPublic, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, external_record_id, field_name, local_value, external_value, status, created_at, resolved_at
		FROM hrms_sync_conflicts
		WHERE tenant_id = $1 AND integration_id = $2
		ORDER BY created_at DESC
		LIMIT 50
	`, tenantID, integrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HRMSSyncConflictPublic{}
	for rows.Next() {
		var item HRMSSyncConflictPublic
		var localRaw []byte
		var externalRaw []byte
		var createdAt time.Time
		var resolvedAt *time.Time
		if err := rows.Scan(&item.ID, &item.ExternalRecordID, &item.FieldName, &localRaw, &externalRaw, &item.Status, &createdAt, &resolvedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(localRaw, &item.LocalValue)
		_ = json.Unmarshal(externalRaw, &item.ExternalValue)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if resolvedAt != nil {
			value := resolvedAt.UTC().Format(time.RFC3339)
			item.ResolvedAt = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *HRMSService) retryWebhookEvent(ctx context.Context, tenantID, integrationID, eventID string) error {
	var provider string
	var payload map[string]interface{}
	var signature string
	err := s.db.QueryRow(ctx, `
		SELECT provider, payload, COALESCE(signature, '')
		FROM hrms_webhook_events
		WHERE tenant_id = $1 AND integration_id = $2 AND id = $3
	`, tenantID, integrationID, eventID).Scan(&provider, &payload, &signature)
	if err != nil {
		return err
	}

	retryCount := 1
	if err := s.db.QueryRow(ctx, `
		UPDATE hrms_webhook_events
		SET retry_count = retry_count + 1, next_retry_at = NULL, updated_at = NOW()
		WHERE tenant_id = $1 AND integration_id = $2 AND id = $3
		RETURNING retry_count
	`, tenantID, integrationID, eventID).Scan(&retryCount); err != nil {
		return err
	}

	if err := s.processWebhookCore(ctx, tenantID, provider, payload, signature); err != nil {
		nextRetry := time.Now().UTC().Add(15 * time.Minute)
		_, _ = s.db.Exec(ctx, `
			UPDATE hrms_webhook_events
			SET status = 'failed', error_message = $1, next_retry_at = $2, updated_at = NOW()
			WHERE tenant_id = $3 AND integration_id = $4 AND id = $5
		`, err.Error(), nextRetry, tenantID, integrationID, eventID)
		return err
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(ctx, `
		UPDATE hrms_webhook_events
		SET status = 'processed', error_message = NULL, processed_at = $1, next_retry_at = NULL, updated_at = NOW()
		WHERE tenant_id = $2 AND integration_id = $3 AND id = $4
	`, now, tenantID, integrationID, eventID)
	return err
}

func (s *HRMSService) rotateIntegrationCredentials(ctx context.Context, tenantID, integrationID string) (map[string]string, error) {
	apiKey, err := generateRandomHex(16)
	if err != nil {
		return nil, err
	}
	apiSecret, err := generateRandomHex(24)
	if err != nil {
		return nil, err
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE hrms_integrations
		SET api_key = $1, api_secret = $2, updated_at = NOW()
		WHERE tenant_id = $3 AND id = $4
	`, apiKey, apiSecret, tenantID, integrationID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return map[string]string{
		"api_key":    apiKey,
		"api_secret": apiSecret,
	}, nil
}

func (s *HRMSService) resolveSyncConflict(ctx context.Context, tenantID, integrationID, conflictID, resolution, actorUserID string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE hrms_sync_conflicts
		SET status = $1, resolved_at = NOW(), resolved_by = $2, updated_at = NOW()
		WHERE tenant_id = $3 AND integration_id = $4 AND id = $5
	`, resolution, actorUserID, tenantID, integrationID, conflictID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *HRMSService) ListWebhookEvents(ctx context.Context, tenantID, integrationID string) ([]HRMSWebhookEventPublic, error) {
	return s.listFailedWebhookEvents(ctx, tenantID, integrationID)
}

func (s *HRMSService) RetryWebhookEvent(ctx context.Context, tenantID, integrationID, eventID string) error {
	return s.retryWebhookEvent(ctx, tenantID, integrationID, eventID)
}

func (s *HRMSService) RotateCredentials(ctx context.Context, tenantID, integrationID string) (map[string]string, error) {
	return s.rotateIntegrationCredentials(ctx, tenantID, integrationID)
}

func (s *HRMSService) ListSyncConflicts(ctx context.Context, tenantID, integrationID string) ([]HRMSSyncConflictPublic, error) {
	return s.listSyncConflicts(ctx, tenantID, integrationID)
}

func (s *HRMSService) ResolveSyncConflict(ctx context.Context, tenantID, integrationID, conflictID, resolution, actorUserID string) error {
	return s.resolveSyncConflict(ctx, tenantID, integrationID, conflictID, resolution, actorUserID)
}

func (s *HRMSService) TestFieldMapping(ctx context.Context, tenantID, integrationID string, sample map[string]interface{}, overrides []map[string]string) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := s.db.QueryRow(ctx, `SELECT config FROM hrms_integrations WHERE tenant_id = $1 AND id = $2`, tenantID, integrationID).Scan(&config); err != nil {
		return nil, err
	}
	mapped, missing := s.mapFields(sample, config, overrides)
	return map[string]interface{}{
		"mapped":         mapped,
		"missing_fields": missing,
	}, nil
}

func (s *HRMSService) DryRunDirectorySync(ctx context.Context, tenantID, integrationID string, records []map[string]interface{}, overrides []map[string]string) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := s.db.QueryRow(ctx, `SELECT config FROM hrms_integrations WHERE tenant_id = $1 AND id = $2`, tenantID, integrationID).Scan(&config); err != nil {
		return nil, err
	}

	validCount := 0
	invalidCount := 0
	previews := []map[string]interface{}{}
	conflicts := []map[string]interface{}{}

	for _, record := range records {
		mapped, missing := s.mapFields(record, config, overrides)
		preview := map[string]interface{}{
			"mapped":         mapped,
			"missing_fields": missing,
		}
		if len(missing) > 0 || fmt.Sprint(mapped["employee_id"]) == "" {
			invalidCount++
			previews = append(previews, preview)
			continue
		}

		validCount++
		previews = append(previews, preview)
		employeeID := fmt.Sprint(mapped["employee_id"])
		var userID uuid.UUID
		var firstName, lastName *string
		err := s.db.QueryRow(ctx, `
			SELECT id, first_name, last_name
			FROM users
			WHERE tenant_id = $1 AND employee_id = $2 AND deleted_at IS NULL
			LIMIT 1
		`, tenantID, employeeID).Scan(&userID, &firstName, &lastName)
		if err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return nil, err
		}

		for _, fieldName := range []string{"first_name", "last_name", "email", "designation"} {
			var localValue interface{}
			switch fieldName {
			case "first_name":
				if firstName != nil {
					localValue = *firstName
				}
			case "last_name":
				if lastName != nil {
					localValue = *lastName
				}
			default:
				localValue = nil
			}
			externalValue := mapped[fieldName]
			if fmt.Sprint(localValue) == "" || fmt.Sprint(externalValue) == "" || fmt.Sprint(localValue) == fmt.Sprint(externalValue) {
				continue
			}
			conflicts = append(conflicts, map[string]interface{}{
				"external_record_id": employeeID,
				"field_name":         fieldName,
				"local_value":        localValue,
				"external_value":     externalValue,
			})
			localRaw, _ := json.Marshal(localValue)
			externalRaw, _ := json.Marshal(externalValue)
			_, _ = s.db.Exec(ctx, `
				INSERT INTO hrms_sync_conflicts (tenant_id, integration_id, external_record_id, field_name, local_value, external_value, status)
				VALUES ($1, $2, $3, $4, $5, $6, 'open')
			`, tenantID, integrationID, employeeID, fieldName, localRaw, externalRaw)
		}
	}

	if len(conflicts) > 0 {
		actionURL := "/admin/org/integrations"
		_ = InsertNotification(ctx, s.db, tenantID, nil, "integration.sync_conflict", "Directory sync conflicts detected", fmt.Sprintf("%d field conflicts were detected during dry-run", len(conflicts)), "warning", &actionURL)
	}

	return map[string]interface{}{
		"valid_count":   validCount,
		"invalid_count": invalidCount,
		"preview":       previews,
		"conflicts":     conflicts,
	}, nil
}
