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
	Provider   string                 `json:"provider"`
	APIKey     string                 `json:"api_key"`
	APISecret  *string                `json:"api_secret"`
	WebhookURL *string                `json:"webhook_url"`
	Config     map[string]interface{} `json:"config"`
	IsActive   *bool                  `json:"is_active"`
}

type UpdateIntegrationInput struct {
	Provider   *string                `json:"provider"`
	APIKey     *string                `json:"api_key"`
	APISecret  *string                `json:"api_secret"`
	WebhookURL *string                `json:"webhook_url"`
	Config     map[string]interface{} `json:"config"`
	IsActive   *bool                  `json:"is_active"`
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
	// Preserve existing webhook URL if not provided.
	if in.WebhookURL == nil {
		var existing *string
		err := s.db.QueryRow(ctx, `
			SELECT webhook_url
			FROM hrms_integrations
			WHERE tenant_id = $1 AND provider = $2
		`, tenantID, in.Provider).Scan(&existing)
		if err != nil && err != pgx.ErrNoRows {
			return nil, err
		}
		in.WebhookURL = existing
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	var m models.HRMSIntegration
	err := s.db.QueryRow(ctx, `
		INSERT INTO hrms_integrations (tenant_id, provider, webhook_url, api_key, api_secret, config, is_active, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		ON CONFLICT (tenant_id, provider)
		DO UPDATE SET
			webhook_url = EXCLUDED.webhook_url,
			api_key = EXCLUDED.api_key,
			api_secret = EXCLUDED.api_secret,
			config = EXCLUDED.config,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING id, tenant_id, provider, webhook_url, config, is_active, last_sync_at, created_at, updated_at
	`, tenantID, in.Provider, in.WebhookURL, in.APIKey, in.APISecret, in.Config, isActive).Scan(
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

// UpdateIntegrationByID updates an integration row by id for a tenant.
func (s *HRMSService) UpdateIntegrationByID(ctx context.Context, tenantID, integrationID string, in UpdateIntegrationInput) (*HRMSIntegrationPublic, error) {
	var current struct {
		Provider  string
		APIKey    string
		APISecret *string
		Webhook   *string
		IsActive  bool
		Config    map[string]interface{}
	}

	err := s.db.QueryRow(ctx, `
		SELECT provider, api_key, api_secret, webhook_url, is_active, config
		FROM hrms_integrations
		WHERE id = $1 AND tenant_id = $2
	`, integrationID, tenantID).Scan(&current.Provider, &current.APIKey, &current.APISecret, &current.Webhook, &current.IsActive, &current.Config)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("integration not found")
		}
		return nil, err
	}

	provider := current.Provider
	if in.Provider != nil && *in.Provider != "" {
		provider = *in.Provider
	}
	apiKey := current.APIKey
	if in.APIKey != nil && *in.APIKey != "" {
		apiKey = *in.APIKey
	}
	apiSecret := current.APISecret
	if in.APISecret != nil {
		if *in.APISecret == "" {
			apiSecret = nil
		} else {
			apiSecret = in.APISecret
		}
	}
	webhookURL := current.Webhook
	if in.WebhookURL != nil {
		if *in.WebhookURL == "" {
			webhookURL = nil
		} else {
			webhookURL = in.WebhookURL
		}
	}
	config := current.Config
	if in.Config != nil {
		config = in.Config
	}
	isActive := current.IsActive
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	var m models.HRMSIntegration
	err = s.db.QueryRow(ctx, `
		UPDATE hrms_integrations
		SET
			provider = $1,
			api_key = $2,
			api_secret = $3,
			webhook_url = $4,
			config = $5,
			is_active = $6,
			updated_at = NOW()
		WHERE id = $7 AND tenant_id = $8
		RETURNING id, tenant_id, provider, webhook_url, config, is_active, last_sync_at, created_at, updated_at
	`, provider, apiKey, apiSecret, webhookURL, config, isActive, integrationID, tenantID).Scan(
		&m.ID, &m.TenantID, &m.Provider, &m.WebhookURL, &m.Config, &m.IsActive, &m.LastSyncAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	out := toPublicIntegration(&m)
	return &out, nil
}

// ToggleIntegration sets active state for integration.
func (s *HRMSService) ToggleIntegration(ctx context.Context, tenantID, integrationID string, isActive bool) (*HRMSIntegrationPublic, error) {
	var m models.HRMSIntegration
	err := s.db.QueryRow(ctx, `
		UPDATE hrms_integrations
		SET is_active = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
		RETURNING id, tenant_id, provider, webhook_url, config, is_active, last_sync_at, created_at, updated_at
	`, isActive, integrationID, tenantID).Scan(
		&m.ID, &m.TenantID, &m.Provider, &m.WebhookURL, &m.Config, &m.IsActive, &m.LastSyncAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("integration not found")
		}
		return nil, err
	}
	out := toPublicIntegration(&m)
	return &out, nil
}

// TestIntegration performs configuration validation and updates last_sync_at when checks pass.
func (s *HRMSService) TestIntegration(ctx context.Context, tenantID, integrationID string) (map[string]interface{}, error) {
	var provider string
	var isActive bool
	var cfg map[string]interface{}

	err := s.db.QueryRow(ctx, `
		SELECT provider, is_active, config
		FROM hrms_integrations
		WHERE id = $1 AND tenant_id = $2
	`, integrationID, tenantID).Scan(&provider, &isActive, &cfg)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("integration not found")
		}
		return nil, err
	}

	checks := []map[string]interface{}{}
	addCheck := func(name string, ok bool, detail string) {
		checks = append(checks, map[string]interface{}{
			"name":   name,
			"ok":     ok,
			"detail": detail,
		})
	}

	addCheck("integration_active", isActive, "Integration should be active to receive webhooks")

	switch provider {
	case "workday":
		_, hasTenant := cfg["tenant"]
		_, hasEndpoint := cfg["endpoint"]
		addCheck("workday_config", hasTenant || hasEndpoint, "Expected config.tenant or config.endpoint")
	case "sap":
		_, hasEndpoint := cfg["endpoint"]
		_, hasCompany := cfg["company_code"]
		addCheck("sap_config", hasEndpoint || hasCompany, "Expected config.endpoint or config.company_code")
	case "bamboohr":
		_, hasSubdomain := cfg["subdomain"]
		_, hasEndpoint := cfg["endpoint"]
		addCheck("bamboohr_config", hasSubdomain || hasEndpoint, "Expected config.subdomain or config.endpoint")
	default:
		_, hasEndpoint := cfg["endpoint"]
		addCheck("custom_config", hasEndpoint, "Expected config.endpoint")
	}

	passed := true
	for _, c := range checks {
		if ok, _ := c["ok"].(bool); !ok {
			passed = false
			break
		}
	}

	if passed {
		_, _ = s.db.Exec(ctx, `
			UPDATE hrms_integrations
			SET last_sync_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2
		`, integrationID, tenantID)
	}

	return map[string]interface{}{
		"ok":      passed,
		"checks":  checks,
		"message": map[bool]string{true: "Integration test passed", false: "Integration test failed"}[passed],
	}, nil
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

type HRMSSyncSchedulePublic struct {
	ID            uuid.UUID `json:"id"`
	IntegrationID uuid.UUID `json:"integration_id"`
	Frequency     string    `json:"frequency"`
	DayOfWeek     *int      `json:"day_of_week,omitempty"`
	TimeOfDay     string    `json:"time_of_day"`
	Timezone      string    `json:"timezone"`
	IsActive      bool      `json:"is_active"`
	LastRunAt     *string   `json:"last_run_at,omitempty"`
	NextRunAt     *string   `json:"next_run_at,omitempty"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
}

type UpsertHRMSSyncScheduleInput struct {
	Frequency string `json:"frequency"`
	DayOfWeek *int   `json:"day_of_week"`
	TimeOfDay string `json:"time_of_day"`
	Timezone  string `json:"timezone"`
	IsActive  *bool  `json:"is_active"`
}

func (s *HRMSService) GetSyncSchedule(ctx context.Context, tenantID, integrationID string) (*HRMSSyncSchedulePublic, error) {
	var row struct {
		ID            uuid.UUID
		IntegrationID uuid.UUID
		Frequency     string
		DayOfWeek     *int
		TimeOfDay     time.Time
		Timezone      string
		IsActive      bool
		LastRunAt     *time.Time
		CreatedAt     time.Time
		UpdatedAt     time.Time
	}
	err := s.db.QueryRow(ctx, `
		SELECT id, integration_id, frequency, day_of_week, time_of_day, timezone, is_active, last_run_at, created_at, updated_at
		FROM hrms_sync_schedules
		WHERE tenant_id = $1 AND integration_id = $2
	`, tenantID, integrationID).Scan(
		&row.ID, &row.IntegrationID, &row.Frequency, &row.DayOfWeek, &row.TimeOfDay, &row.Timezone,
		&row.IsActive, &row.LastRunAt, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return toSyncSchedulePublic(row), nil
}

func (s *HRMSService) UpsertSyncSchedule(ctx context.Context, tenantID, integrationID string, in UpsertHRMSSyncScheduleInput) (*HRMSSyncSchedulePublic, error) {
	if in.Frequency == "" {
		return nil, fmt.Errorf("frequency is required")
	}
	if in.TimeOfDay == "" {
		in.TimeOfDay = "00:00"
	}
	if in.Timezone == "" {
		in.Timezone = "UTC"
	}
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	var row struct {
		ID            uuid.UUID
		IntegrationID uuid.UUID
		Frequency     string
		DayOfWeek     *int
		TimeOfDay     time.Time
		Timezone      string
		IsActive      bool
		LastRunAt     *time.Time
		CreatedAt     time.Time
		UpdatedAt     time.Time
	}
	err := s.db.QueryRow(ctx, `
		INSERT INTO hrms_sync_schedules (tenant_id, integration_id, frequency, day_of_week, time_of_day, timezone, is_active, updated_at)
		VALUES ($1,$2,$3,$4,$5::time,$6,$7,NOW())
		ON CONFLICT (integration_id)
		DO UPDATE SET
			frequency = EXCLUDED.frequency,
			day_of_week = EXCLUDED.day_of_week,
			time_of_day = EXCLUDED.time_of_day,
			timezone = EXCLUDED.timezone,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
		RETURNING id, integration_id, frequency, day_of_week, time_of_day, timezone, is_active, last_run_at, created_at, updated_at
	`, tenantID, integrationID, in.Frequency, in.DayOfWeek, in.TimeOfDay, in.Timezone, isActive).Scan(
		&row.ID, &row.IntegrationID, &row.Frequency, &row.DayOfWeek, &row.TimeOfDay, &row.Timezone,
		&row.IsActive, &row.LastRunAt, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return toSyncSchedulePublic(row), nil
}

func (s *HRMSService) DeleteSyncSchedule(ctx context.Context, tenantID, integrationID string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM hrms_sync_schedules
		WHERE tenant_id = $1 AND integration_id = $2
	`, tenantID, integrationID)
	return err
}

type HRMSSyncLogPublic struct {
	ID            uuid.UUID `json:"id"`
	IntegrationID uuid.UUID `json:"integration_id"`
	Status        string    `json:"status"`
	Message       *string   `json:"message,omitempty"`
	StartedAt     string    `json:"started_at"`
	CompletedAt   *string   `json:"completed_at,omitempty"`
}

func (s *HRMSService) ListSyncLogs(ctx context.Context, tenantID, integrationID string, limit, offset int) ([]HRMSSyncLogPublic, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, integration_id, status, message, started_at, completed_at
		FROM hrms_sync_logs
		WHERE tenant_id = $1 AND integration_id = $2
		ORDER BY started_at DESC
		LIMIT $3 OFFSET $4
	`, tenantID, integrationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []HRMSSyncLogPublic{}
	for rows.Next() {
		var r HRMSSyncLogPublic
		var started time.Time
		var completed *time.Time
		if err := rows.Scan(&r.ID, &r.IntegrationID, &r.Status, &r.Message, &started, &completed); err != nil {
			return nil, err
		}
		r.StartedAt = started.UTC().Format(time.RFC3339)
		if completed != nil {
			t := completed.UTC().Format(time.RFC3339)
			r.CompletedAt = &t
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *HRMSService) RunSync(ctx context.Context, tenantID, integrationID, message string) (*HRMSSyncLogPublic, error) {
	var logRow HRMSSyncLogPublic
	var started time.Time
	var completed time.Time
	err := s.db.QueryRow(ctx, `
		INSERT INTO hrms_sync_logs (tenant_id, integration_id, status, message, started_at, completed_at)
		VALUES ($1,$2,'success',$3,NOW(),NOW())
		RETURNING id, integration_id, status, message, started_at, completed_at
	`, tenantID, integrationID, message).Scan(
		&logRow.ID, &logRow.IntegrationID, &logRow.Status, &logRow.Message, &started, &completed,
	)
	if err != nil {
		return nil, err
	}
	logRow.StartedAt = started.UTC().Format(time.RFC3339)
	t := completed.UTC().Format(time.RFC3339)
	logRow.CompletedAt = &t

	_, _ = s.db.Exec(ctx, `
		UPDATE hrms_integrations
		SET last_sync_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`, integrationID, tenantID)

	_, _ = s.db.Exec(ctx, `
		UPDATE hrms_sync_schedules
		SET last_run_at = NOW(), updated_at = NOW()
		WHERE integration_id = $1 AND tenant_id = $2
	`, integrationID, tenantID)

	return &logRow, nil
}

func toSyncSchedulePublic(row struct {
	ID            uuid.UUID
	IntegrationID uuid.UUID
	Frequency     string
	DayOfWeek     *int
	TimeOfDay     time.Time
	Timezone      string
	IsActive      bool
	LastRunAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}) *HRMSSyncSchedulePublic {
	var lastRun *string
	if row.LastRunAt != nil {
		s := row.LastRunAt.UTC().Format(time.RFC3339)
		lastRun = &s
	}
	nextRun := computeNextRun(row.Frequency, row.DayOfWeek, row.TimeOfDay, row.Timezone)
	return &HRMSSyncSchedulePublic{
		ID:            row.ID,
		IntegrationID: row.IntegrationID,
		Frequency:     row.Frequency,
		DayOfWeek:     row.DayOfWeek,
		TimeOfDay:     row.TimeOfDay.Format("15:04"),
		Timezone:      row.Timezone,
		IsActive:      row.IsActive,
		LastRunAt:     lastRun,
		NextRunAt:     nextRun,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func computeNextRun(freq string, dayOfWeek *int, timeOfDay time.Time, tz string) *string {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), timeOfDay.Hour(), timeOfDay.Minute(), 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	switch freq {
	case "hourly":
		next = now.Truncate(time.Hour).Add(time.Hour)
	case "weekly":
		if dayOfWeek != nil {
			// move to next desired weekday
			delta := (*dayOfWeek - int(next.Weekday()) + 7) % 7
			if delta == 0 && !next.After(now) {
				delta = 7
			}
			next = next.AddDate(0, 0, delta)
		}
	case "monthly":
		if !next.After(now) {
			next = next.AddDate(0, 1, 0)
		}
	}
	s := next.UTC().Format(time.RFC3339)
	return &s
}
