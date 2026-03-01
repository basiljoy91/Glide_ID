package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttendanceService struct {
	db              *pgxpool.Pool
	mqttClient      MQTTClient
	aiServiceURL    string
	aiServiceAPIKey string
}

// GetDB returns the database connection (for middleware)
func (s *AttendanceService) GetDB() *pgxpool.Pool {
	return s.db
}

type MQTTClient interface {
	Publish(topic string, payload []byte) error
}

func NewAttendanceService(db *pgxpool.Pool, mqttClient MQTTClient, aiServiceURL, aiServiceAPIKey string) *AttendanceService {
	return &AttendanceService{
		db:              db,
		mqttClient:      mqttClient,
		aiServiceURL:    aiServiceURL,
		aiServiceAPIKey: aiServiceAPIKey,
	}
}

// CheckInRequest represents a check-in request from kiosk
type CheckInRequest struct {
	ImageBase64      string  `json:"image_base64"`
	KioskCode       string  `json:"kiosk_code"`
	LocalTime       *string `json:"local_time"`
	MonotonicOffsetMs *int64 `json:"monotonic_offset_ms"`
	VerificationMethod string `json:"verification_method"` // "biometric", "pin"
	PinCode         *string `json:"pin_code"`
	IPAddress       *string `json:"ip_address"`
}

// CheckInResponse represents the response
type CheckInResponse struct {
	Success         bool      `json:"success"`
	UserID          *string   `json:"user_id"`
	UserName        *string   `json:"user_name"`
	Confidence      *float64  `json:"confidence"`
	PunchTime       time.Time `json:"punch_time"`
	Status          string    `json:"status"` // "check_in" or "check_out"
	DoorOpened      bool      `json:"door_opened"`
	Message         string    `json:"message"`
}

// ProcessCheckIn processes a check-in/check-out request
func (s *AttendanceService) ProcessCheckIn(ctx context.Context, tenantID string, req CheckInRequest) (*CheckInResponse, error) {
	// Set tenant context for RLS
	if err := s.db.QueryRow(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID).Scan(); err != nil && err.Error() != "no rows in result set" {
		// Ignore error for SET LOCAL
	}

	// Get kiosk info
	var kioskID uuid.UUID
	var mqttTopic *string
	err := s.db.QueryRow(ctx, `
		SELECT id, mqtt_topic
		FROM kiosks
		WHERE code = $1 AND tenant_id = $2 AND status = 'active'
	`, req.KioskCode, tenantID).Scan(&kioskID, &mqttTopic)
	if err != nil {
		return nil, fmt.Errorf("kiosk not found: %w", err)
	}

	// Calculate true punch time using monotonic clock offset
	punchTime := time.Now()
	if req.MonotonicOffsetMs != nil && req.LocalTime != nil {
		// Reconcile offline time
		localTime, err := time.Parse(time.RFC3339, *req.LocalTime)
		if err == nil {
			offsetDuration := time.Duration(*req.MonotonicOffsetMs) * time.Millisecond
			punchTime = localTime.Add(offsetDuration)
		}
	}

	// If PIN verification, do lightweight face detection for buddy punching prevention
	if req.VerificationMethod == "pin" && req.PinCode != nil {
		return s.processPinCheckIn(ctx, tenantID, kioskID, req, punchTime, mqttTopic)
	}

	// Biometric verification via AI service
	return s.processBiometricCheckIn(ctx, tenantID, kioskID, req, punchTime, mqttTopic)
}

func (s *AttendanceService) processBiometricCheckIn(
	ctx context.Context,
	tenantID string,
	kioskID uuid.UUID,
	req CheckInRequest,
	punchTime time.Time,
	mqttTopic *string,
) (*CheckInResponse, error) {
	// Call AI service for 1:N comparison
	aiReq := map[string]interface{}{
		"image_base64": req.ImageBase64,
		"tenant_id":    tenantID,
		"threshold":    0.85,
	}

	jsonData, _ := json.Marshal(aiReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.aiServiceURL+"/api/v1/compare/multiple", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create AI request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", s.aiServiceAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("AI service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &CheckInResponse{
			Success: false,
			Message: "Face recognition failed",
		}, nil
	}

	var aiResp struct {
		Matches []struct {
			UserID      string  `json:"user_id"`
			Confidence  float64 `json:"confidence"`
			UserDetails struct {
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"user_details"`
		} `json:"matches"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, fmt.Errorf("failed to decode AI response: %w", err)
	}

	if len(aiResp.Matches) == 0 {
		return &CheckInResponse{
			Success: false,
			Message: "No matching face found",
		}, nil
	}

	// Get best match
	match := aiResp.Matches[0]
	userID := match.UserID
	confidence := match.Confidence

	// Determine check-in vs check-out based on last attendance
	status := s.determineAttendanceStatus(ctx, userID, tenantID)

	// Create attendance log
	attendanceLog := models.AttendanceLog{
		ID:                 uuid.New(),
		TenantID:           uuid.MustParse(tenantID),
		UserID:             uuid.MustParse(userID),
		KioskID:            &kioskID,
		Status:             status,
		PunchTime:          punchTime,
		FaceMatchConfidence: &confidence,
		VerificationMethod: "biometric",
		IPAddress:          req.IPAddress,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Insert attendance log
	_, err = s.db.Exec(ctx, `
		INSERT INTO attendance_logs (
			id, tenant_id, user_id, kiosk_id, status, punch_time,
			face_match_confidence, verification_method, ip_address, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, attendanceLog.ID, attendanceLog.TenantID, attendanceLog.UserID, attendanceLog.KioskID,
		attendanceLog.Status, attendanceLog.PunchTime, attendanceLog.FaceMatchConfidence,
		attendanceLog.VerificationMethod, attendanceLog.IPAddress, attendanceLog.CreatedAt, attendanceLog.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert attendance log: %w", err)
	}

	// Trigger door relay via MQTT
	doorOpened := false
	if s.mqttClient != nil && mqttTopic != nil {
		doorPayload := map[string]interface{}{
			"action":   "open",
			"user_id":  userID,
			"kiosk_id": kioskID.String(),
			"timestamp": time.Now().Unix(),
		}
		doorJSON, _ := json.Marshal(doorPayload)
		if err := s.mqttClient.Publish(*mqttTopic, doorJSON); err == nil {
			doorOpened = true
		}
	}

	userName := fmt.Sprintf("%s %s", match.UserDetails.FirstName, match.UserDetails.LastName)

	return &CheckInResponse{
		Success:    true,
		UserID:     &userID,
		UserName:   &userName,
		Confidence: &confidence,
		PunchTime:  punchTime,
		Status:     status,
		DoorOpened: doorOpened,
		Message:    fmt.Sprintf("Successfully checked %s", status),
	}, nil
}

func (s *AttendanceService) processPinCheckIn(
	ctx context.Context,
	tenantID string,
	kioskID uuid.UUID,
	req CheckInRequest,
	punchTime time.Time,
	mqttTopic *string,
) (*CheckInResponse, error) {
	// Verify PIN and get user
	var userID uuid.UUID
	var firstName, lastName string
	err := s.db.QueryRow(ctx, `
		SELECT id, first_name, last_name
		FROM users
		WHERE tenant_id = $1 AND employee_id = $2 AND is_active = true AND deleted_at IS NULL
	`, tenantID, req.PinCode).Scan(&userID, &firstName, &lastName)

	if err != nil {
		return &CheckInResponse{
			Success: false,
			Message: "Invalid PIN",
		}, nil
	}

	// Lightweight face detection for buddy punching prevention
	// Call AI service for liveness detection only
	aiReq := map[string]interface{}{
		"image_base64": req.ImageBase64,
		"liveness_type": "passive",
	}

	jsonData, _ := json.Marshal(aiReq)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", s.aiServiceURL+"/api/v1/liveness", bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", s.aiServiceAPIKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(httpReq)
	anomalyDetected := false
	if err == nil && resp.StatusCode == http.StatusOK {
		var livenessResp struct {
			IsLive bool `json:"is_live"`
		}
		if json.NewDecoder(resp.Body).Decode(&livenessResp) == nil && !livenessResp.IsLive {
			anomalyDetected = true
		}
		resp.Body.Close()
	}

	// Determine status
	status := s.determineAttendanceStatus(ctx, userID.String(), tenantID)

	// Create attendance log
	attendanceLog := models.AttendanceLog{
		ID:                 uuid.New(),
		TenantID:           uuid.MustParse(tenantID),
		UserID:             userID,
		KioskID:            &kioskID,
		Status:             status,
		PunchTime:          punchTime,
		VerificationMethod: "pin",
		PinUsed:            true,
		AnomalyDetected:    anomalyDetected,
		IPAddress:          req.IPAddress,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if anomalyDetected {
		reason := "Liveness detection failed - possible buddy punching"
		attendanceLog.AnomalyReason = &reason
	}

	// Insert attendance log
	_, err = s.db.Exec(ctx, `
		INSERT INTO attendance_logs (
			id, tenant_id, user_id, kiosk_id, status, punch_time,
			verification_method, pin_used, anomaly_detected, anomaly_reason, ip_address, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, attendanceLog.ID, attendanceLog.TenantID, attendanceLog.UserID, attendanceLog.KioskID,
		attendanceLog.Status, attendanceLog.PunchTime, attendanceLog.VerificationMethod,
		attendanceLog.PinUsed, attendanceLog.AnomalyDetected, attendanceLog.AnomalyReason,
		attendanceLog.IPAddress, attendanceLog.CreatedAt, attendanceLog.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert attendance log: %w", err)
	}

	// Trigger door relay
	doorOpened := false
	if s.mqttClient != nil && mqttTopic != nil && !anomalyDetected {
		doorPayload := map[string]interface{}{
			"action":   "open",
			"user_id":  userID.String(),
			"kiosk_id": kioskID.String(),
			"timestamp": time.Now().Unix(),
		}
		doorJSON, _ := json.Marshal(doorPayload)
		if err := s.mqttClient.Publish(*mqttTopic, doorJSON); err == nil {
			doorOpened = true
		}
	}

	userIDStr := userID.String()
	userName := fmt.Sprintf("%s %s", firstName, lastName)
	message := fmt.Sprintf("Successfully checked %s", status)
	if anomalyDetected {
		message += " (Anomaly detected - flagged for HR review)"
	}

	return &CheckInResponse{
		Success:    true,
		UserID:     &userIDStr,
		UserName:   &userName,
		PunchTime:  punchTime,
		Status:     status,
		DoorOpened: doorOpened,
		Message:    message,
	}, nil
}

// determineAttendanceStatus determines if this should be check-in or check-out
func (s *AttendanceService) determineAttendanceStatus(ctx context.Context, userID, tenantID string) string {
	var lastStatus string
	err := s.db.QueryRow(ctx, `
		SELECT status
		FROM attendance_logs
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY punch_time DESC
		LIMIT 1
	`, userID, tenantID).Scan(&lastStatus)

	// If no previous record or last was check_out, this is check_in
	if err != nil || lastStatus == "check_out" {
		return "check_in"
	}

	// Otherwise, toggle
	if lastStatus == "check_in" {
		return "check_out"
	}

	return "check_in"
}

