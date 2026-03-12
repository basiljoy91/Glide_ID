package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user/employee
type User struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	EmployeeID         string     `json:"employee_id"`
	Email              string     `json:"email"`
	PasswordHash       *string    `json:"-"` // for password auth only; never expose
	Phone              *string    `json:"phone"`
	FirstName          string     `json:"first_name"`
	LastName           string     `json:"last_name"`
	DepartmentID       *uuid.UUID `json:"department_id"`
	Designation        *string    `json:"designation"`
	DateOfJoining      time.Time  `json:"date_of_joining"`
	ShiftStartTime     *string    `json:"shift_start_time"`
	ShiftEndTime       *string    `json:"shift_end_time"`
	ShiftLengthHours   *float64   `json:"shift_length_hours"`
	Role               string     `json:"role"`
	IsActive           bool       `json:"is_active"`
	DataPrivacyConsent bool       `json:"data_privacy_consent"`
	ConsentDate        *time.Time `json:"consent_date"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	LastCheckInAt      *time.Time `json:"last_check_in_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at"`
}

// Tenant represents an organization/tenant
type Tenant struct {
	ID                    uuid.UUID  `json:"id"`
	Name                  string     `json:"name"`
	Slug                  string     `json:"slug"`
	SubscriptionTier      string     `json:"subscription_tier"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at"`
	MaxUsers              int        `json:"max_users"`
	MaxKiosks             int        `json:"max_kiosks"`
	KioskCode             string     `json:"kiosk_code"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// Kiosk represents a kiosk device
type Kiosk struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Name            string     `json:"name"`
	Code            string     `json:"code"`
	HMACSecret      string     `json:"-"` // Never expose in JSON
	Status          string     `json:"status"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at"`
	MQTTTopic       *string    `json:"mqtt_topic"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AttendanceLog represents an attendance record
type AttendanceLog struct {
	ID                  uuid.UUID  `json:"id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	UserID              uuid.UUID  `json:"user_id"`
	KioskID             *uuid.UUID `json:"kiosk_id"`
	Status              string     `json:"status"` // check_in, check_out, break_start, break_end
	PunchTime           time.Time  `json:"punch_time"`
	LocalTime           *time.Time `json:"local_time"`
	MonotonicOffsetMs   *int64     `json:"monotonic_offset_ms"`
	FaceMatchConfidence *float64   `json:"face_match_confidence"`
	LivenessType        *string    `json:"liveness_type"`
	LivenessScore       *float64   `json:"liveness_score"`
	VerificationMethod  string     `json:"verification_method"`
	PinUsed             bool       `json:"pin_used"`
	AnomalyDetected     bool       `json:"anomaly_detected"`
	AnomalyReason       *string    `json:"anomaly_reason"`
	IPAddress           *string    `json:"ip_address"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// Department represents a department
type Department struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	Code        *string    `json:"code"`
	Description *string    `json:"description"`
	ManagerID   *uuid.UUID `json:"manager_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AuditLog represents an audit record
type AuditLog struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     *uuid.UUID             `json:"tenant_id"`
	UserID       *uuid.UUID             `json:"user_id"`
	TargetUserID *uuid.UUID             `json:"target_user_id"`
	Action       string                 `json:"action"`
	ResourceType *string                `json:"resource_type"`
	ResourceID   *uuid.UUID             `json:"resource_id"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    *string                `json:"ip_address"`
	UserAgent    *string                `json:"user_agent"`
	CreatedAt    time.Time              `json:"created_at"`
}

// HRMSIntegration represents an HRMS integration
type HRMSIntegration struct {
	ID         uuid.UUID              `json:"id"`
	TenantID   uuid.UUID              `json:"tenant_id"`
	Provider   string                 `json:"provider"`
	WebhookURL *string                `json:"webhook_url"`
	APIKey     string                 `json:"-"` // Never expose
	APISecret  *string                `json:"-"` // Never expose
	Config     map[string]interface{} `json:"config"`
	IsActive   bool                   `json:"is_active"`
	LastSyncAt *time.Time             `json:"last_sync_at"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// PayrollExport represents a payroll export
type PayrollExport struct {
	ID             uuid.UUID              `json:"id"`
	TenantID       uuid.UUID              `json:"tenant_id"`
	ExportType     string                 `json:"export_type"`
	DateRangeStart time.Time              `json:"date_range_start"`
	DateRangeEnd   time.Time              `json:"date_range_end"`
	RequestedBy    uuid.UUID              `json:"requested_by"`
	Status         string                 `json:"status"`
	FileURL        *string                `json:"file_url"`
	Payload        map[string]interface{} `json:"payload"`
	ErrorMessage   *string                `json:"error_message"`
	CreatedAt      time.Time              `json:"created_at"`
	CompletedAt    *time.Time             `json:"completed_at"`
}
