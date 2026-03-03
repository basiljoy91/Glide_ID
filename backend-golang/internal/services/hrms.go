package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HRMSService struct {
	db *pgxpool.Pool
}

func NewHRMSService(db *pgxpool.Pool) *HRMSService {
	return &HRMSService{db: db}
}

type HRMSIntegrationPublic struct {
	ID         uuid.UUID              `json:"id"`
	Provider   string                 `json:"provider"`
	WebhookURL *string                `json:"webhook_url"`
	Config     map[string]interface{} `json:"config"`
	IsActive   bool                   `json:"is_active"`
	LastSyncAt *string                `json:"last_sync_at,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

func toPublicIntegration(m *models.HRMSIntegration) HRMSIntegrationPublic {
	var lastSync *string
	if m.LastSyncAt != nil {
		s := m.LastSyncAt.UTC().Format(time.RFC3339)
		lastSync = &s
	}
	return HRMSIntegrationPublic{
		ID:         m.ID,
		Provider:   m.Provider,
		WebhookURL: m.WebhookURL,
		Config:     m.Config,
		IsActive:   m.IsActive,
		LastSyncAt: lastSync,
		CreatedAt:  m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  m.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ListIntegrations returns tenant-scoped HRMS integrations (public fields only).
func (s *HRMSService) ListIntegrations(ctx context.Context, tenantID string) ([]HRMSIntegrationPublic, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, provider, webhook_url, config, is_active, last_sync_at, created_at, updated_at
		FROM hrms_integrations
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]HRMSIntegrationPublic, 0)
	for rows.Next() {
		var m models.HRMSIntegration
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Provider, &m.WebhookURL, &m.Config, &m.IsActive, &m.LastSyncAt, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, toPublicIntegration(&m))
	}
	return out, nil
}

type UpsertIntegrationInput struct {
	Provider  string                 `json:"provider"`
	APIKey    string                 `json:"api_key"`
	APISecret *string                `json:"api_secret"`
	Config    map[string]interface{} `json:"config"`
	IsActive  *bool                  `json:"is_active"`
}

// UpsertIntegration inserts or updates an integration for (tenant_id, provider).
func (s *HRMSService) UpsertIntegration(ctx context.Context, tenantID string, in UpsertIntegrationInput) (*HRMSIntegrationPublic, error) {
	if in.Provider == "" || in.APIKey == "" {
		return nil, fmt.Errorf("provider and api_key are required")
	}
	if in.Config == nil {
		in.Config = map[string]interface{}{}
	}

	// Preserve existing secret if not provided.
	if in.APISecret == nil {
		var existing *string
		err := s.db.QueryRow(ctx, `
			SELECT api_secret
			FROM hrms_integrations
			WHERE tenant_id = $1 AND provider = $2
		`, tenantID, in.Provider).Scan(&existing)
		if err != nil && err != pgx.ErrNoRows {
			return nil, err
		}
		in.APISecret = existing
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	var m models.HRMSIntegration
	err := s.db.QueryRow(ctx, `
		INSERT INTO hrms_integrations (tenant_id, provider, api_key, api_secret, config, is_active, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW())
		ON CONFLICT (tenant_id, provider)
		DO UPDATE SET
			api_key = EXCLUDED.api_key,
			api_secret = EXCLUDED.api_secret,
			config = EXCLUDED.config,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING id, tenant_id, provider, webhook_url, config, is_active, last_sync_at, created_at, updated_at
	`, tenantID, in.Provider, in.APIKey, in.APISecret, in.Config, isActive).Scan(
		&m.ID, &m.TenantID, &m.Provider, &m.WebhookURL, &m.Config, &m.IsActive, &m.LastSyncAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	pub := toPublicIntegration(&m)
	return &pub, nil
}

// ProcessWebhook processes an incoming HRMS webhook
func (s *HRMSService) ProcessWebhook(ctx context.Context, tenantID, provider string, payload map[string]interface{}, signature string) error {
	// Verify webhook signature
	integration := &models.HRMSIntegration{}
	err := s.db.QueryRow(ctx, `
		SELECT id, api_secret
		FROM hrms_integrations
		WHERE tenant_id = $1 AND provider = $2 AND is_active = true
	`, tenantID, provider).Scan(&integration.ID, &integration.APISecret)

	if err != nil {
		return fmt.Errorf("HRMS integration not found: %w", err)
	}

	// Verify HMAC signature if secret exists
	if integration.APISecret != nil && *integration.APISecret != "" {
		if !s.verifyWebhookSignature(payload, signature, *integration.APISecret) {
			return fmt.Errorf("invalid webhook signature")
		}
	}

	// Process webhook based on provider
	switch provider {
	case "workday":
		return s.processWorkdayWebhook(ctx, tenantID, payload)
	case "sap":
		return s.processSAPWebhook(ctx, tenantID, payload)
	default:
		return s.processGenericWebhook(ctx, tenantID, payload)
	}
}

func (s *HRMSService) verifyWebhookSignature(payload map[string]interface{}, signature, secret string) bool {
	// Create message from payload
	message := fmt.Sprintf("%v", payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (s *HRMSService) processWorkdayWebhook(ctx context.Context, tenantID string, payload map[string]interface{}) error {
	// Implement Workday-specific webhook processing
	// This would create/update users based on Workday data
	return nil
}

func (s *HRMSService) processSAPWebhook(ctx context.Context, tenantID string, payload map[string]interface{}) error {
	// Implement SAP-specific webhook processing
	return nil
}

func (s *HRMSService) processGenericWebhook(ctx context.Context, tenantID string, payload map[string]interface{}) error {
	// Generic webhook processing
	return nil
}

// ExportTimesheet exports attendance data for payroll
func (s *HRMSService) ExportTimesheet(ctx context.Context, tenantID string, startDate, endDate string) (map[string]interface{}, error) {
	// Query attendance logs for date range
	rows, err := s.db.Query(ctx, `
		SELECT user_id, punch_time, status
		FROM attendance_logs
		WHERE tenant_id = $1 AND punch_time >= $2 AND punch_time <= $3
		ORDER BY user_id, punch_time
	`, tenantID, startDate, endDate)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Format data for payroll export
	timesheetData := make(map[string]interface{})
	timesheetData["start_date"] = startDate
	timesheetData["end_date"] = endDate
	timesheetData["records"] = []map[string]interface{}{}

	for rows.Next() {
		var userID uuid.UUID
		var punchTime string
		var status string
		if err := rows.Scan(&userID, &punchTime, &status); err != nil {
			continue
		}
		timesheetData["records"] = append(timesheetData["records"].([]map[string]interface{}), map[string]interface{}{
			"user_id":    userID.String(),
			"punch_time": punchTime,
			"status":     status,
		})
	}

	return timesheetData, nil
}

