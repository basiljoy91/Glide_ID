package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HRMSService struct {
	db *pgxpool.Pool
}

func NewHRMSService(db *pgxpool.Pool) *HRMSService {
	return &HRMSService{db: db}
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

