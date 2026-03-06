package services

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttendanceService struct {
	db                 *pgxpool.Pool
	mqttClient         MQTTClient
	aiServiceURL       string
	aiServiceAPIKey    string
	offlinePrivateKey  *rsa.PrivateKey
	faceMatchThreshold float64
	vectorizeTimeout   time.Duration
	livenessTimeout    time.Duration
	compareTimeout     time.Duration
	pinTimeout         time.Duration
}

// GetDB returns the database connection (for middleware)
func (s *AttendanceService) GetDB() *pgxpool.Pool {
	return s.db
}

type MQTTClient interface {
	Publish(topic string, payload []byte) error
}

func NewAttendanceService(
	db *pgxpool.Pool,
	mqttClient MQTTClient,
	aiServiceURL, aiServiceAPIKey string,
	offlinePrivateKeyPEM string,
	faceMatchThreshold float64,
	vectorizeTimeout time.Duration,
	livenessTimeout time.Duration,
	compareTimeout time.Duration,
	pinTimeout time.Duration,
) *AttendanceService {
	var pk *rsa.PrivateKey
	if pemRaw := strings.TrimSpace(offlinePrivateKeyPEM); pemRaw != "" {
		if parsed, err := parseRSAPrivateKeyPEM([]byte(pemRaw)); err == nil {
			pk = parsed
		}
	}
	if pk == nil {
		// Fallback for local/dev: load private key from file candidates.
		if pemFromFile := loadOfflinePrivateKeyFromFileCandidates(); pemFromFile != "" {
			if parsed, err := parseRSAPrivateKeyPEM([]byte(pemFromFile)); err == nil {
				pk = parsed
			}
		}
	}
	return &AttendanceService{
		db:                 db,
		mqttClient:         mqttClient,
		aiServiceURL:       aiServiceURL,
		aiServiceAPIKey:    aiServiceAPIKey,
		offlinePrivateKey:  pk,
		faceMatchThreshold: faceMatchThreshold,
		vectorizeTimeout:   normalizeAITimeout(vectorizeTimeout, 90*time.Second),
		livenessTimeout:    normalizeAITimeout(livenessTimeout, 15*time.Second),
		compareTimeout:     normalizeAITimeout(compareTimeout, 20*time.Second),
		pinTimeout:         normalizeAITimeout(pinTimeout, 8*time.Second),
	}
}

func normalizeAITimeout(v time.Duration, fallback time.Duration) time.Duration {
	if v <= 0 {
		return fallback
	}
	if v < time.Second {
		return time.Second
	}
	return v
}

func (s *AttendanceService) clampFaceThreshold() float64 {
	t := s.faceMatchThreshold
	if t <= 0 {
		t = 0.62
	}
	if t < 0.45 {
		return 0.45
	}
	if t > 0.92 {
		return 0.92
	}
	return t
}

func loadOfflinePrivateKeyFromFileCandidates() string {
	candidates := []string{}

	if envPath := strings.TrimSpace(os.Getenv("OFFLINE_PRIVATE_KEY_PATH")); envPath != "" {
		candidates = append(candidates, envPath)
	}

	candidates = append(candidates,
		"../keys/kiosk_offline_private.pem",
		"./keys/kiosk_offline_private.pem",
	)

	if exePath, err := os.Executable(); err == nil && exePath != "" {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "keys", "kiosk_offline_private.pem"),
			filepath.Join(exeDir, "..", "keys", "kiosk_offline_private.pem"),
		)
	}

	for _, p := range candidates {
		clean := filepath.Clean(p)
		b, err := os.ReadFile(clean)
		if err != nil || len(b) == 0 {
			continue
		}
		return string(b)
	}
	return ""
}

type offlineEnvelope struct {
	Alg string `json:"alg"`
	EK  string `json:"ek"` // RSA-OAEP encrypted AES key (base64)
	IV  string `json:"iv"` // AES-GCM IV (base64)
	CT  string `json:"ct"` // AES-GCM ciphertext (base64)
}

func parseRSAPrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM")
	}
	// Try PKCS8 first
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := k.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, errors.New("not RSA private key")
	}
	// Fallback PKCS1
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	return nil, errors.New("failed to parse RSA private key")
}

func (s *AttendanceService) decryptOfflinePayload(encryptedPayload string) ([]byte, error) {
	if s.offlinePrivateKey == nil {
		return nil, errors.New("offline decryption not configured")
	}

	var env offlineEnvelope
	if err := json.Unmarshal([]byte(encryptedPayload), &env); err != nil {
		return nil, fmt.Errorf("invalid envelope: %w", err)
	}

	ek, err := base64.StdEncoding.DecodeString(env.EK)
	if err != nil {
		return nil, fmt.Errorf("invalid ek: %w", err)
	}
	iv, err := base64.StdEncoding.DecodeString(env.IV)
	if err != nil {
		return nil, fmt.Errorf("invalid iv: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(env.CT)
	if err != nil {
		return nil, fmt.Errorf("invalid ct: %w", err)
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, s.offlinePrivateKey, ek, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt key failed: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	plain, err := gcm.Open(nil, iv, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload failed: %w", err)
	}
	return plain, nil
}

func (s *AttendanceService) IsOfflineDecryptionConfigured() bool {
	return s.offlinePrivateKey != nil
}

type OfflineSyncDecrypted struct {
	Type            string `json:"type"`
	ImageData       string `json:"imageData"`
	Timestamp       int64  `json:"timestamp"`
	MonotonicOffset int64  `json:"monotonicOffset"`
	HasConsented    *bool  `json:"has_consented"`
}

// ProcessOfflineSync decrypts an offline payload and processes it as a check-in.
func (s *AttendanceService) ProcessOfflineSync(ctx context.Context, tenantID, kioskCode string, encryptedPayload string) (*CheckInResponse, error) {
	plain, err := s.decryptOfflinePayload(encryptedPayload)
	if err != nil {
		return nil, err
	}
	var d OfflineSyncDecrypted
	if err := json.Unmarshal(plain, &d); err != nil {
		return nil, fmt.Errorf("invalid decrypted payload: %w", err)
	}
	// Convert to ProcessCheckIn request
	localTime := time.UnixMilli(d.Timestamp).UTC().Format(time.RFC3339)
	mon := d.MonotonicOffset
	req := CheckInRequest{
		ImageBase64:        "",
		KioskCode:          kioskCode,
		LocalTime:          &localTime,
		MonotonicOffsetMs:  &mon,
		VerificationMethod: "biometric",
		HasConsented:       d.HasConsented,
	}
	// Extract base64 from data URL if present
	if idx := strings.Index(d.ImageData, ","); idx != -1 {
		req.ImageBase64 = d.ImageData[idx+1:]
	} else {
		req.ImageBase64 = d.ImageData
	}
	return s.ProcessCheckIn(ctx, tenantID, req)
}

// VectorizeAndStore calls the AI service to vectorize and persist a face vector for a user.
func (s *AttendanceService) VectorizeAndStore(ctx context.Context, tenantID, userID, imageBase64 string) error {
	if imageBase64 == "" {
		return fmt.Errorf("image_base64 is required")
	}
	payload := map[string]interface{}{
		"user_id":         userID,
		"tenant_id":       tenantID,
		"image_base64":    imageBase64,
		"update_existing": false,
	}
	jsonData, _ := json.Marshal(payload)
	client := &http.Client{Timeout: s.vectorizeTimeout}
	var lastErr error

	for attempt := 1; attempt <= 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", s.aiServiceURL+"/api/v1/vectorize", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create AI request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if s.aiServiceAPIKey != "" {
			req.Header.Set("X-API-Key", s.aiServiceAPIKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < 2 && isRetryableAIError(err) {
				time.Sleep(900 * time.Millisecond)
				continue
			}
			return fmt.Errorf("AI service request failed: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		lastErr = fmt.Errorf("AI service responded with %d: %s", resp.StatusCode, msg)

		if attempt < 2 && (resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests) {
			time.Sleep(900 * time.Millisecond)
			continue
		}
		return lastErr
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("AI service request failed")
}

func isRetryableAIError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "timeout") || strings.Contains(errText, "deadline exceeded")
}

type aiLivenessResponse struct {
	IsLive        bool                   `json:"is_live"`
	LivenessScore float64                `json:"liveness_score"`
	Details       map[string]interface{} `json:"details"`
}

type compareCandidate struct {
	UserID      string  `json:"user_id"`
	Confidence  float64 `json:"confidence"`
	Votes       int     `json:"votes"`
	UserDetails struct {
		EmployeeID string `json:"employee_id"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
	} `json:"user_details"`
}

type compareMultipleAIResponse struct {
	TotalCandidates int                `json:"total_candidates"`
	Matches         []compareCandidate `json:"matches"`
}

func (s *AttendanceService) callLiveness(
	ctx context.Context,
	imageBase64 string,
	livenessType string,
	challengeType string,
	framesBase64 []string,
) (*aiLivenessResponse, error) {
	payload := map[string]interface{}{
		"image_base64":   imageBase64,
		"liveness_type":  livenessType,
		"challenge_type": challengeType,
	}
	if len(framesBase64) > 0 {
		payload["frames_base64"] = framesBase64
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", s.aiServiceURL+"/api/v1/liveness", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create liveness request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.aiServiceAPIKey)

	client := &http.Client{Timeout: s.livenessTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("liveness service request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("liveness service responded with %d: %s", resp.StatusCode, msg)
	}

	var out aiLivenessResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode liveness response: %w", err)
	}
	return &out, nil
}

func parseLivenessAntiSpoofMetrics(details map[string]interface{}) (bool, float64, float64) {
	toFloat := func(v interface{}) (float64, bool) {
		switch t := v.(type) {
		case float64:
			return t, true
		case float32:
			return float64(t), true
		case int:
			return float64(t), true
		case int64:
			return float64(t), true
		default:
			return 0, false
		}
	}
	toBool := func(v interface{}) (bool, bool) {
		b, ok := v.(bool)
		return b, ok
	}

	supported := false
	vote := 0.5
	score := 0.5
	if v, ok := details["anti_spoof_supported"]; ok {
		if b, ok := toBool(v); ok {
			supported = b
		}
	}
	if v, ok := details["anti_spoof_vote"]; ok {
		if f, ok := toFloat(v); ok {
			vote = f
		}
	}
	if v, ok := details["anti_spoof_score"]; ok {
		if f, ok := toFloat(v); ok {
			score = f
		}
	}
	return supported, vote, score
}

func hardSpoofFromLivenessDetails(details map[string]interface{}) bool {
	supported, vote, score := parseLivenessAntiSpoofMetrics(details)
	return supported && vote <= 0.0 && score < 0.16
}

func isAmbiguousMatch(matches []compareCandidate, threshold float64) bool {
	if len(matches) < 2 {
		return false
	}
	top1 := matches[0]
	top2 := matches[1]
	gap := top1.Confidence - top2.Confidence

	requiredGap := 0.036
	switch {
	case top1.Confidence >= threshold+0.16:
		requiredGap = 0.012
	case top1.Confidence >= threshold+0.12:
		requiredGap = 0.018
	case top1.Confidence >= threshold+0.08:
		requiredGap = 0.024
	case top1.Confidence >= threshold+0.05:
		requiredGap = 0.030
	}
	if gap >= requiredGap {
		return false
	}

	// If multi-frame consensus is stronger for top-1, allow even with a tight score gap.
	voteGap := top1.Votes - top2.Votes
	if top1.Votes >= 2 && voteGap >= 1 && top1.Confidence >= threshold+0.03 {
		return false
	}
	return true
}

// CheckInRequest represents a check-in request from kiosk
type CheckInRequest struct {
	ImageBase64        string   `json:"image_base64"`
	KioskCode          string   `json:"kiosk_code"`
	LocalTime          *string  `json:"local_time"`
	MonotonicOffsetMs  *int64   `json:"monotonic_offset_ms"`
	VerificationMethod string   `json:"verification_method"` // "biometric", "pin"
	PinCode            *string  `json:"pin_code"`
	IPAddress          *string  `json:"ip_address"`
	HasConsented       *bool    `json:"has_consented"`
	LivenessType       *string  `json:"liveness_type"`  // "active" | "passive"
	ChallengeType      *string  `json:"challenge_type"` // optional challenge
	FramesBase64       []string `json:"frames_base64"`  // optional active challenge frames
}

// CheckInResponse represents the response
type CheckInResponse struct {
	Success    bool      `json:"success"`
	UserID     *string   `json:"user_id"`
	UserName   *string   `json:"user_name"`
	EmployeeID *string   `json:"employee_id,omitempty"`
	Confidence *float64  `json:"confidence"`
	PunchTime  time.Time `json:"punch_time"`
	Status     string    `json:"status"` // "check_in" or "check_out"
	DoorOpened bool      `json:"door_opened"`
	Message    string    `json:"message"`
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
	// Enforce AI liveness checks before face matching to block spoof attacks.
	livenessType := "active"
	if req.LivenessType != nil && *req.LivenessType != "" {
		livenessType = *req.LivenessType
	}
	challengeType := "any"
	if req.ChallengeType != nil && *req.ChallengeType != "" {
		challengeType = *req.ChallengeType
	}
	livenessBody, err := s.callLiveness(ctx, req.ImageBase64, livenessType, challengeType, req.FramesBase64)
	if err != nil {
		return nil, fmt.Errorf("liveness service request failed: %w", err)
	}

	if !livenessBody.IsLive && livenessType == "active" && !hardSpoofFromLivenessDetails(livenessBody.Details) {
		passiveBody, perr := s.callLiveness(ctx, req.ImageBase64, "passive", "any", nil)
		if perr == nil && passiveBody != nil {
			livenessBody = passiveBody
			livenessType = "passive"
		}
	}

	if !livenessBody.IsLive {
		hardSpoof := hardSpoofFromLivenessDetails(livenessBody.Details)
		message := "Live verification could not be confirmed. Keep your face centered and retry."
		if hardSpoof || livenessBody.LivenessScore < 0.20 {
			message = "Spoof risk detected. Please retry live verification."
		}
		if reasonRaw, ok := livenessBody.Details["reason"]; ok {
			if reason, ok := reasonRaw.(string); ok {
				reasonLower := strings.ToLower(reason)
				if strings.Contains(reasonLower, "no face") {
					message = "No face detected. Step closer and keep your face centered."
				}
				if strings.Contains(reasonLower, "trajectory") || strings.Contains(reasonLower, "frame") {
					message = "Please hold steady for a moment and retry live verification."
				}
				if strings.Contains(reasonLower, "movement") {
					message = "Please move your head slightly and retry live verification."
				}
				if strings.Contains(reasonLower, "passive guard") || strings.Contains(reasonLower, "below threshold") {
					message = "Live verification could not be confirmed. Keep your face centered and retry."
				}
			}
		}
		return &CheckInResponse{
			Success: false,
			Message: message,
		}, nil
	}

	// Call AI service for 1:N comparison.
	threshold := s.clampFaceThreshold()
	aiResp, err := s.compareMultiple(ctx, req.ImageBase64, req.FramesBase64, tenantID, threshold)
	if err != nil {
		return nil, fmt.Errorf("face comparison failed: %w", err)
	}
	if len(aiResp.Matches) == 0 && threshold > 0.50 {
		// Fallback query with relaxed threshold for noisy webcam captures.
		relaxed := threshold - 0.10
		if relaxed < 0.50 {
			relaxed = 0.50
		}
		fallbackResp, ferr := s.compareMultiple(ctx, req.ImageBase64, req.FramesBase64, tenantID, relaxed)
		if ferr == nil && len(fallbackResp.Matches) > 0 {
			aiResp = fallbackResp
		}
	}

	if len(aiResp.Matches) == 0 {
		if aiResp.TotalCandidates == 0 {
			return &CheckInResponse{
				Success: false,
				Message: "No ArcFace enrollments found for this organization. Please re-enroll employees.",
			}, nil
		}
		return &CheckInResponse{
			Success: false,
			Message: "No matching face found",
		}, nil
	}

	if isAmbiguousMatch(aiResp.Matches, threshold) {
		return &CheckInResponse{
			Success: false,
			Message: "Ambiguous face match detected. Please retry or use PIN fallback.",
		}, nil
	}

	// Get best match
	match := aiResp.Matches[0]
	userID := match.UserID
	confidence := match.Confidence

	// Enterprise safeguard: block accidental duplicate punches within a short window.
	var lastPunch time.Time
	var lastStatus string
	if err := s.db.QueryRow(ctx, `
		SELECT punch_time, status
		FROM attendance_logs
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY punch_time DESC
		LIMIT 1
	`, userID, tenantID).Scan(&lastPunch, &lastStatus); err == nil {
		if time.Since(lastPunch) < 20*time.Second {
			return &CheckInResponse{
				Success: false,
				Message: fmt.Sprintf("Recent %s already recorded. Please wait a few seconds.", lastStatus),
			}, nil
		}
	}

	// Determine check-in vs check-out based on last attendance
	status := s.determineAttendanceStatus(ctx, userID, tenantID)

	// Create attendance log
	attendanceLog := models.AttendanceLog{
		ID:                  uuid.New(),
		TenantID:            uuid.MustParse(tenantID),
		UserID:              uuid.MustParse(userID),
		KioskID:             &kioskID,
		Status:              status,
		PunchTime:           punchTime,
		FaceMatchConfidence: &confidence,
		VerificationMethod:  "biometric",
		IPAddress:           req.IPAddress,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
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

	// If kiosk flow indicated explicit consent, persist on user record
	if req.HasConsented != nil && *req.HasConsented {
		_, _ = s.db.Exec(ctx, `
			UPDATE users
			SET data_privacy_consent = true,
				consent_date = COALESCE(consent_date, NOW()),
				updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2 AND data_privacy_consent = false
		`, userID, tenantID)
	}

	// Trigger door relay via MQTT
	doorOpened := false
	if s.mqttClient != nil && mqttTopic != nil {
		doorPayload := map[string]interface{}{
			"action":    "open",
			"user_id":   userID,
			"kiosk_id":  kioskID.String(),
			"timestamp": time.Now().Unix(),
		}
		doorJSON, _ := json.Marshal(doorPayload)
		if err := s.mqttClient.Publish(*mqttTopic, doorJSON); err == nil {
			doorOpened = true
		}
	}

	userName := fmt.Sprintf("%s %s", match.UserDetails.FirstName, match.UserDetails.LastName)
	employeeID := match.UserDetails.EmployeeID

	return &CheckInResponse{
		Success:    true,
		UserID:     &userID,
		UserName:   &userName,
		EmployeeID: &employeeID,
		Confidence: &confidence,
		PunchTime:  punchTime,
		Status:     status,
		DoorOpened: doorOpened,
		Message:    fmt.Sprintf("Successfully checked %s", status),
	}, nil
}

func (s *AttendanceService) compareMultiple(ctx context.Context, imageBase64 string, framesBase64 []string, tenantID string, threshold float64) (*compareMultipleAIResponse, error) {
	aiReq := map[string]interface{}{
		"image_base64":  imageBase64,
		"frames_base64": framesBase64,
		"tenant_id":     tenantID,
		"threshold":     threshold,
	}

	jsonData, _ := json.Marshal(aiReq)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.aiServiceURL+"/api/v1/compare/multiple", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create AI request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", s.aiServiceAPIKey)

	client := &http.Client{Timeout: s.compareTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("AI service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("AI service compare/multiple error: %s", msg)
	}

	var aiResp compareMultipleAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, fmt.Errorf("failed to decode AI response: %w", err)
	}
	return &aiResp, nil
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
		"image_base64":  req.ImageBase64,
		"liveness_type": "passive",
	}

	jsonData, _ := json.Marshal(aiReq)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", s.aiServiceURL+"/api/v1/liveness", bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", s.aiServiceAPIKey)

	client := &http.Client{Timeout: s.pinTimeout}
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
			"action":    "open",
			"user_id":   userID.String(),
			"kiosk_id":  kioskID.String(),
			"timestamp": time.Now().Unix(),
		}
		doorJSON, _ := json.Marshal(doorPayload)
		if err := s.mqttClient.Publish(*mqttTopic, doorJSON); err == nil {
			doorOpened = true
		}
	}

	userIDStr := userID.String()
	userName := fmt.Sprintf("%s %s", firstName, lastName)
	employeeID := ""
	if req.PinCode != nil {
		employeeID = *req.PinCode
	}
	message := fmt.Sprintf("Successfully checked %s", status)
	if anomalyDetected {
		message += " (Anomaly detected - flagged for HR review)"
	}

	return &CheckInResponse{
		Success:    true,
		UserID:     &userIDStr,
		UserName:   &userName,
		EmployeeID: &employeeID,
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
