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
	ManagerID          *uuid.UUID `json:"manager_id"`
	EmploymentType     *string    `json:"employment_type"`
	WorkLocation       *string    `json:"work_location"`
	CostCenter         *string    `json:"cost_center"`
	InviteStatus       string     `json:"invite_status"`
	InviteSentAt       *time.Time `json:"invite_sent_at"`
	OffboardedAt       *time.Time `json:"offboarded_at"`
	OffboardingReason  *string    `json:"offboarding_reason"`
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
	Location        *string    `json:"location"`
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

type CompanyProfileSettings struct {
	DisplayName  string `json:"display_name"`
	LegalName    string `json:"legal_name"`
	BrandColor   string `json:"brand_color"`
	LogoURL      string `json:"logo_url"`
	SupportEmail string `json:"support_email"`
	SupportPhone string `json:"support_phone"`
}

type Holiday struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

type OperationalSettings struct {
	Timezone        string    `json:"timezone"`
	WorkWeek        []string  `json:"work_week"`
	HolidayCalendar []Holiday `json:"holiday_calendar"`
}

type AttendancePolicySettings struct {
	LateGraceMinutes                 int     `json:"late_grace_minutes"`
	EarlyDepartureGraceMinutes       int     `json:"early_departure_grace_minutes"`
	BreakGraceMinutes                int     `json:"break_grace_minutes"`
	AutoCheckoutHours                float64 `json:"auto_checkout_hours"`
	RegularizationRequiresApproval   bool    `json:"regularization_requires_approval"`
	AllowManualAttendanceAdjustments bool    `json:"allow_manual_attendance_adjustments"`
}

type KioskDefaultSettings struct {
	HeartbeatGraceMinutes  int    `json:"heartbeat_grace_minutes"`
	OfflineSyncWindowHours int    `json:"offline_sync_window_hours"`
	RequirePinFallback     bool   `json:"require_pin_fallback"`
	DefaultLocation        string `json:"default_location"`
}

type DataRetentionSettings struct {
	AttendanceLogDays     int `json:"attendance_log_days"`
	AuditLogDays          int `json:"audit_log_days"`
	InactiveUserPurgeDays int `json:"inactive_user_purge_days"`
}

type PasswordPolicySettings struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumber    bool `json:"require_number"`
	RequireSymbol    bool `json:"require_symbol"`
	ExpireDays       int  `json:"expire_days"`
}

type SecuritySettings struct {
	EnforceMFA            bool                   `json:"enforce_mfa"`
	RequireMFAForRoles    []string               `json:"require_mfa_for_roles"`
	PasswordPolicy        PasswordPolicySettings `json:"password_policy"`
	TrustedIPRanges       []string               `json:"trusted_ip_ranges"`
	SessionTimeoutMinutes int                    `json:"session_timeout_minutes"`
	SSOEnabled            bool                   `json:"sso_enabled"`
}

type ShiftTemplate struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
	Days         []string `json:"days"`
	GraceMinutes int      `json:"grace_minutes"`
	Notes        string   `json:"notes"`
	IsDefault    bool     `json:"is_default"`
}

type OrganizationSettings struct {
	CompanyProfile   CompanyProfileSettings   `json:"company_profile"`
	Operational      OperationalSettings      `json:"operational"`
	AttendancePolicy AttendancePolicySettings `json:"attendance_policy"`
	KioskDefaults    KioskDefaultSettings     `json:"kiosk_defaults"`
	DataRetention    DataRetentionSettings    `json:"data_retention"`
	ShiftTemplates   []ShiftTemplate          `json:"shift_templates"`
}

type SSOConfiguration struct {
	Enabled  bool           `json:"enabled"`
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
}

type CustomRole struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenant_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	IsActive      bool      `json:"is_active"`
	Permissions   []string  `json:"permissions"`
	AssignedUsers int       `json:"assigned_users"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CustomRoleUpsert struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	IsActive    bool     `json:"is_active"`
	Permissions []string `json:"permissions"`
}

type CustomRoleSummary struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type CustomRoleAssignment struct {
	UserID         uuid.UUID `json:"user_id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Email          string    `json:"email"`
	BaseRole       string    `json:"base_role"`
	CustomRoleID   uuid.UUID `json:"custom_role_id"`
	CustomRoleName string    `json:"custom_role_name"`
	AssignedAt     time.Time `json:"assigned_at"`
}

type AuthSession struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	UserID     uuid.UUID  `json:"user_id"`
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	IPAddress  *string    `json:"ip_address"`
	UserAgent  *string    `json:"user_agent"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type EmergencyContact struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	Relationship *string   `json:"relationship"`
	Phone        string    `json:"phone"`
	Email        *string   `json:"email"`
	IsPrimary    bool      `json:"is_primary"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type EmployeeDocument struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	UserID       uuid.UUID      `json:"user_id"`
	DocumentType string         `json:"document_type"`
	Name         string         `json:"name"`
	FileURL      string         `json:"file_url"`
	ExpiresAt    *time.Time     `json:"expires_at"`
	Metadata     map[string]any `json:"metadata"`
	UploadedBy   *uuid.UUID     `json:"uploaded_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type LeaveRequest struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	UserID      uuid.UUID  `json:"user_id"`
	LeaveType   string     `json:"leave_type"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     time.Time  `json:"end_date"`
	DayCount    float64    `json:"day_count"`
	Reason      *string    `json:"reason"`
	Status      string     `json:"status"`
	SubmittedAt time.Time  `json:"submitted_at"`
	ReviewedBy  *uuid.UUID `json:"reviewed_by"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	ReviewNote  *string    `json:"review_note"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AttendanceRegularizationRequest struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	UserID             uuid.UUID  `json:"user_id"`
	AttendanceLogID    *uuid.UUID `json:"attendance_log_id"`
	RequestDate        time.Time  `json:"request_date"`
	RequestedStatus    string     `json:"requested_status"`
	RequestedPunchTime time.Time  `json:"requested_punch_time"`
	Reason             string     `json:"reason"`
	Status             string     `json:"status"`
	SubmittedAt        time.Time  `json:"submitted_at"`
	ReviewedBy         *uuid.UUID `json:"reviewed_by"`
	ReviewedAt         *time.Time `json:"reviewed_at"`
	ReviewNote         *string    `json:"review_note"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type OvertimeRequest struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	UserID           uuid.UUID  `json:"user_id"`
	WorkDate         time.Time  `json:"work_date"`
	RequestedMinutes int        `json:"requested_minutes"`
	ApprovedMinutes  int        `json:"approved_minutes"`
	Reason           *string    `json:"reason"`
	Status           string     `json:"status"`
	SubmittedAt      time.Time  `json:"submitted_at"`
	ReviewedBy       *uuid.UUID `json:"reviewed_by"`
	ReviewedAt       *time.Time `json:"reviewed_at"`
	ReviewNote       *string    `json:"review_note"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ShiftAssignment struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	UserID    uuid.UUID  `json:"user_id"`
	ShiftName string     `json:"shift_name"`
	StartDate time.Time  `json:"start_date"`
	EndDate   time.Time  `json:"end_date"`
	StartTime string     `json:"start_time"`
	EndTime   string     `json:"end_time"`
	WorkDays  []string   `json:"work_days"`
	IsRota    bool       `json:"is_rota"`
	Notes     *string    `json:"notes"`
	CreatedBy *uuid.UUID `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type AttendanceExceptionAssignment struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	AttendanceLogID uuid.UUID  `json:"attendance_log_id"`
	AssignedTo      uuid.UUID  `json:"assigned_to"`
	AssignedBy      *uuid.UUID `json:"assigned_by"`
	SLADueAt        *time.Time `json:"sla_due_at"`
	Status          string     `json:"status"`
	Note            *string    `json:"note"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
