package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	PermissionSettingsManage     = "settings.manage"
	PermissionSecurityManage     = "security.manage"
	PermissionSessionsManage     = "sessions.manage"
	PermissionRolesManage        = "roles.manage"
	PermissionAuditView          = "audit.view"
	PermissionUsersManage        = "users.manage"
	PermissionDepartmentsManage  = "departments.manage"
	PermissionKiosksManage       = "kiosks.manage"
	PermissionIntegrationsManage = "integrations.manage"
	PermissionReportsView        = "reports.view"
	PermissionReviewsManage      = "reviews.manage"
	PermissionAttendanceView     = "attendance.view"
	PermissionAttendanceExport   = "attendance.export"
)

var (
	allPermissionCatalog = []string{
		PermissionSettingsManage,
		PermissionSecurityManage,
		PermissionSessionsManage,
		PermissionRolesManage,
		PermissionAuditView,
		PermissionUsersManage,
		PermissionDepartmentsManage,
		PermissionKiosksManage,
		PermissionIntegrationsManage,
		PermissionReportsView,
		PermissionReviewsManage,
		PermissionAttendanceView,
		PermissionAttendanceExport,
	}
	defaultRolePermissions = map[string][]string{
		"super_admin":  allPermissionCatalog,
		"org_admin":    allPermissionCatalog,
		"hr":           {PermissionUsersManage, PermissionDepartmentsManage, PermissionIntegrationsManage, PermissionAuditView, PermissionReportsView, PermissionReviewsManage, PermissionAttendanceView, PermissionAttendanceExport},
		"dept_manager": {PermissionReportsView, PermissionReviewsManage, PermissionAttendanceView},
		"employee":     {},
	}
	ErrSessionNotFound          = errors.New("session not found")
	ErrCustomRoleNotFound       = errors.New("custom role not found")
	ErrInvalidTrustedIPRange    = errors.New("invalid trusted IP range")
	ErrInvalidPasswordPolicy    = errors.New("password does not satisfy policy")
	ErrCustomRoleUserIneligible = errors.New("user is not eligible for a custom role")
)

type AdminService struct {
	db *pgxpool.Pool
}

func NewAdminService(db *pgxpool.Pool) *AdminService {
	return &AdminService{db: db}
}

func (s *AdminService) GetDB() *pgxpool.Pool {
	return s.db
}

type tenantSettingsDocument struct {
	CompanyProfile   models.CompanyProfileSettings   `json:"company_profile"`
	Operational      models.OperationalSettings      `json:"operational"`
	AttendancePolicy models.AttendancePolicySettings `json:"attendance_policy"`
	KioskDefaults    models.KioskDefaultSettings     `json:"kiosk_defaults"`
	DataRetention    models.DataRetentionSettings    `json:"data_retention"`
	Security         models.SecuritySettings         `json:"security"`
	ShiftTemplates   []models.ShiftTemplate          `json:"shift_templates"`
}

func defaultTenantSettings() tenantSettingsDocument {
	return tenantSettingsDocument{
		CompanyProfile: models.CompanyProfileSettings{BrandColor: "#111827"},
		Operational: models.OperationalSettings{
			Timezone: "UTC",
			WorkWeek: []string{"monday", "tuesday", "wednesday", "thursday", "friday"},
		},
		AttendancePolicy: models.AttendancePolicySettings{
			LateGraceMinutes:                 10,
			EarlyDepartureGraceMinutes:       10,
			BreakGraceMinutes:                5,
			AutoCheckoutHours:                16,
			RegularizationRequiresApproval:   true,
			AllowManualAttendanceAdjustments: false,
		},
		KioskDefaults: models.KioskDefaultSettings{
			HeartbeatGraceMinutes:  15,
			OfflineSyncWindowHours: 24,
			RequirePinFallback:     true,
		},
		DataRetention: models.DataRetentionSettings{
			AttendanceLogDays:     365,
			AuditLogDays:          730,
			InactiveUserPurgeDays: 365,
		},
		Security: models.SecuritySettings{
			EnforceMFA:         false,
			RequireMFAForRoles: []string{"org_admin", "hr"},
			PasswordPolicy: models.PasswordPolicySettings{
				MinLength:        12,
				RequireUppercase: true,
				RequireLowercase: true,
				RequireNumber:    true,
				RequireSymbol:    true,
				ExpireDays:       90,
			},
			SessionTimeoutMinutes: 60 * 8,
		},
		ShiftTemplates: []models.ShiftTemplate{},
	}
}

func isMissingTenantSettingsColumnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, `column "settings" does not exist`) ||
		strings.Contains(msg, `column "sso_provider" does not exist`) ||
		strings.Contains(msg, `column "sso_config" does not exist`)
}

func isRecoverableTenantSettingsError(err error) bool {
	if err == nil {
		return false
	}
	if isMissingTenantSettingsColumnError(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid character") ||
		strings.Contains(msg, "cannot unmarshal") ||
		strings.Contains(msg, "json")
}

func DefaultPasswordPolicy() models.PasswordPolicySettings {
	return defaultTenantSettings().Security.PasswordPolicy
}

func normalizeTenantSettings(doc tenantSettingsDocument) tenantSettingsDocument {
	defaults := defaultTenantSettings()
	if strings.TrimSpace(doc.CompanyProfile.BrandColor) == "" {
		doc.CompanyProfile.BrandColor = defaults.CompanyProfile.BrandColor
	}
	if strings.TrimSpace(doc.Operational.Timezone) == "" {
		doc.Operational.Timezone = defaults.Operational.Timezone
	}
	if len(doc.Operational.WorkWeek) == 0 {
		doc.Operational.WorkWeek = append([]string{}, defaults.Operational.WorkWeek...)
	}
	if doc.AttendancePolicy.LateGraceMinutes <= 0 {
		doc.AttendancePolicy.LateGraceMinutes = defaults.AttendancePolicy.LateGraceMinutes
	}
	if doc.AttendancePolicy.EarlyDepartureGraceMinutes <= 0 {
		doc.AttendancePolicy.EarlyDepartureGraceMinutes = defaults.AttendancePolicy.EarlyDepartureGraceMinutes
	}
	if doc.AttendancePolicy.BreakGraceMinutes < 0 {
		doc.AttendancePolicy.BreakGraceMinutes = defaults.AttendancePolicy.BreakGraceMinutes
	}
	if doc.AttendancePolicy.AutoCheckoutHours <= 0 {
		doc.AttendancePolicy.AutoCheckoutHours = defaults.AttendancePolicy.AutoCheckoutHours
	}
	if doc.KioskDefaults.HeartbeatGraceMinutes <= 0 {
		doc.KioskDefaults.HeartbeatGraceMinutes = defaults.KioskDefaults.HeartbeatGraceMinutes
	}
	if doc.KioskDefaults.OfflineSyncWindowHours <= 0 {
		doc.KioskDefaults.OfflineSyncWindowHours = defaults.KioskDefaults.OfflineSyncWindowHours
	}
	if doc.DataRetention.AttendanceLogDays <= 0 {
		doc.DataRetention.AttendanceLogDays = defaults.DataRetention.AttendanceLogDays
	}
	if doc.DataRetention.AuditLogDays <= 0 {
		doc.DataRetention.AuditLogDays = defaults.DataRetention.AuditLogDays
	}
	if doc.DataRetention.InactiveUserPurgeDays <= 0 {
		doc.DataRetention.InactiveUserPurgeDays = defaults.DataRetention.InactiveUserPurgeDays
	}
	if len(doc.Security.RequireMFAForRoles) == 0 {
		doc.Security.RequireMFAForRoles = append([]string{}, defaults.Security.RequireMFAForRoles...)
	}
	if doc.Security.PasswordPolicy.MinLength <= 0 {
		doc.Security.PasswordPolicy.MinLength = defaults.Security.PasswordPolicy.MinLength
	}
	if doc.Security.SessionTimeoutMinutes <= 0 {
		doc.Security.SessionTimeoutMinutes = defaults.Security.SessionTimeoutMinutes
	}
	if doc.ShiftTemplates == nil {
		doc.ShiftTemplates = []models.ShiftTemplate{}
	}
	doc.Operational.WorkWeek = normalizeLowerList(doc.Operational.WorkWeek)
	doc.Security.RequireMFAForRoles = normalizeLowerList(doc.Security.RequireMFAForRoles)
	doc.Security.TrustedIPRanges = normalizeStringList(doc.Security.TrustedIPRanges)
	for idx := range doc.ShiftTemplates {
		if strings.TrimSpace(doc.ShiftTemplates[idx].ID) == "" {
			doc.ShiftTemplates[idx].ID = uuid.NewString()
		}
		doc.ShiftTemplates[idx].Days = normalizeLowerList(doc.ShiftTemplates[idx].Days)
	}
	return doc
}

func normalizeStringList(values []string) []string {
	set := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := set[trimmed]; exists {
			continue
		}
		set[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func normalizeLowerList(values []string) []string {
	lowered := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		lowered = append(lowered, trimmed)
	}
	return normalizeStringList(lowered)
}

func (s *AdminService) loadTenantSettings(ctx context.Context, tenantID string) (tenantSettingsDocument, *string, map[string]any, error) {
	defaults := defaultTenantSettings()
	var raw []byte
	var ssoProvider *string
	var ssoConfigBytes []byte
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(settings, '{}'::jsonb), sso_provider, COALESCE(sso_config, '{}'::jsonb)
		FROM tenants
		WHERE id = $1 AND deleted_at IS NULL
	`, tenantID).Scan(&raw, &ssoProvider, &ssoConfigBytes); err != nil {
		var exists bool
		if legacyErr := s.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM tenants
				WHERE id = $1 AND deleted_at IS NULL
			)
		`, tenantID).Scan(&exists); legacyErr == nil && exists {
			return defaults, nil, map[string]any{}, nil
		}
		return defaults, nil, nil, err
	}
	doc := defaults
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &doc); err != nil {
			doc = defaults
		}
	}
	var ssoConfig map[string]any
	if len(ssoConfigBytes) > 0 {
		if err := json.Unmarshal(ssoConfigBytes, &ssoConfig); err != nil {
			ssoConfig = map[string]any{}
		}
	}
	doc = normalizeTenantSettings(doc)
	return doc, ssoProvider, ssoConfig, nil
}

func (s *AdminService) saveTenantSettings(ctx context.Context, tenantID string, doc tenantSettingsDocument, ssoProvider *string, ssoConfig map[string]any) error {
	doc = normalizeTenantSettings(doc)
	settingsJSON, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(ssoConfig)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		UPDATE tenants
		SET settings = $2::jsonb,
			sso_provider = $3,
			sso_config = $4::jsonb,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, tenantID, string(settingsJSON), nullableTrimmedString(ssoProvider), string(configJSON))
	return err
}

func nullableTrimmedString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *AdminService) GetOrganizationSettings(ctx context.Context, tenantID string) (*models.OrganizationSettings, error) {
	doc, _, _, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &models.OrganizationSettings{
		CompanyProfile:   doc.CompanyProfile,
		Operational:      doc.Operational,
		AttendancePolicy: doc.AttendancePolicy,
		KioskDefaults:    doc.KioskDefaults,
		DataRetention:    doc.DataRetention,
		ShiftTemplates:   doc.ShiftTemplates,
	}, nil
}

func (s *AdminService) UpdateOrganizationSettings(ctx context.Context, tenantID string, payload models.OrganizationSettings) (*models.OrganizationSettings, error) {
	doc, ssoProvider, ssoConfig, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	doc.CompanyProfile = payload.CompanyProfile
	doc.Operational = payload.Operational
	doc.AttendancePolicy = payload.AttendancePolicy
	doc.KioskDefaults = payload.KioskDefaults
	doc.DataRetention = payload.DataRetention
	if payload.ShiftTemplates != nil {
		doc.ShiftTemplates = payload.ShiftTemplates
	}
	if err := s.saveTenantSettings(ctx, tenantID, doc, ssoProvider, ssoConfig); err != nil {
		return nil, err
	}
	return s.GetOrganizationSettings(ctx, tenantID)
}

func (s *AdminService) GetSecuritySettings(ctx context.Context, tenantID string) (*models.SecuritySettings, error) {
	doc, _, _, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	security := doc.Security
	return &security, nil
}

func (s *AdminService) UpdateSecuritySettings(ctx context.Context, tenantID string, payload models.SecuritySettings) (*models.SecuritySettings, error) {
	for _, cidr := range payload.TrustedIPRanges {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			if ip := net.ParseIP(cidr); ip == nil {
				return nil, ErrInvalidTrustedIPRange
			}
		}
	}
	doc, ssoProvider, ssoConfig, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	doc.Security = payload
	if err := s.saveTenantSettings(ctx, tenantID, doc, ssoProvider, ssoConfig); err != nil {
		return nil, err
	}
	return s.GetSecuritySettings(ctx, tenantID)
}

func (s *AdminService) GetSSOConfiguration(ctx context.Context, tenantID string) (*models.SSOConfiguration, error) {
	doc, provider, config, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	cfg := &models.SSOConfiguration{
		Enabled:  doc.Security.SSOEnabled,
		Provider: strings.TrimSpace(valueOrEmpty(provider)),
		Config:   config,
	}
	if cfg.Config == nil {
		cfg.Config = map[string]any{}
	}
	return cfg, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *AdminService) UpdateSSOConfiguration(ctx context.Context, tenantID string, payload models.SSOConfiguration) (*models.SSOConfiguration, error) {
	doc, _, _, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	doc.Security.SSOEnabled = payload.Enabled
	provider := strings.TrimSpace(payload.Provider)
	if payload.Config == nil {
		payload.Config = map[string]any{}
	}
	if err := s.saveTenantSettings(ctx, tenantID, doc, &provider, payload.Config); err != nil {
		return nil, err
	}
	return s.GetSSOConfiguration(ctx, tenantID)
}

func (s *AdminService) ListShiftTemplates(ctx context.Context, tenantID string) ([]models.ShiftTemplate, error) {
	doc, _, _, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return doc.ShiftTemplates, nil
}

func (s *AdminService) UpsertShiftTemplate(ctx context.Context, tenantID string, template models.ShiftTemplate) (*models.ShiftTemplate, error) {
	doc, ssoProvider, ssoConfig, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(template.ID) == "" {
		template.ID = uuid.NewString()
	}
	template.Name = strings.TrimSpace(template.Name)
	if template.Name == "" {
		return nil, errors.New("shift template name is required")
	}
	template.Days = normalizeLowerList(template.Days)
	replaced := false
	for idx := range doc.ShiftTemplates {
		if doc.ShiftTemplates[idx].ID == template.ID {
			doc.ShiftTemplates[idx] = template
			replaced = true
			break
		}
	}
	if !replaced {
		doc.ShiftTemplates = append(doc.ShiftTemplates, template)
	}
	if template.IsDefault {
		for idx := range doc.ShiftTemplates {
			if doc.ShiftTemplates[idx].ID != template.ID {
				doc.ShiftTemplates[idx].IsDefault = false
			}
		}
	}
	if err := s.saveTenantSettings(ctx, tenantID, doc, ssoProvider, ssoConfig); err != nil {
		return nil, err
	}
	return &template, nil
}

func (s *AdminService) DeleteShiftTemplate(ctx context.Context, tenantID, templateID string) error {
	doc, ssoProvider, ssoConfig, err := s.loadTenantSettings(ctx, tenantID)
	if err != nil {
		return err
	}
	filtered := make([]models.ShiftTemplate, 0, len(doc.ShiftTemplates))
	for _, item := range doc.ShiftTemplates {
		if item.ID == templateID {
			continue
		}
		filtered = append(filtered, item)
	}
	doc.ShiftTemplates = filtered
	return s.saveTenantSettings(ctx, tenantID, doc, ssoProvider, ssoConfig)
}

func (s *AdminService) GetPermissionsCatalog() []string {
	catalog := make([]string, len(allPermissionCatalog))
	copy(catalog, allPermissionCatalog)
	return catalog
}

func defaultPermissionsForRole(role string) []string {
	values := defaultRolePermissions[role]
	perms := make([]string, len(values))
	copy(perms, values)
	return perms
}

func permissionsFromRows(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
	set := map[string]struct{}{}
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		set[permission] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for permission := range set {
		result = append(result, permission)
	}
	sort.Strings(result)
	return result, nil
}

func (s *AdminService) GetEffectivePermissions(ctx context.Context, tenantID, userID, baseRole string) ([]string, *models.CustomRoleSummary, error) {
	if baseRole == "super_admin" {
		return defaultPermissionsForRole(baseRole), nil, nil
	}
	var customRole models.CustomRoleSummary
	var hasAssignment bool
	row := s.db.QueryRow(ctx, `
		SELECT cr.id, cr.name
		FROM custom_role_assignments cra
		JOIN custom_roles cr ON cr.id = cra.custom_role_id
		WHERE cra.user_id = $1 AND cr.tenant_id = $2 AND cr.deleted_at IS NULL AND cr.is_active = true
	`, userID, tenantID)
	if err := row.Scan(&customRole.ID, &customRole.Name); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, err
		}
	} else {
		hasAssignment = true
	}
	if !hasAssignment {
		return defaultPermissionsForRole(baseRole), nil, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT permission
		FROM custom_role_permissions
		WHERE custom_role_id = $1
	`, customRole.ID)
	if err != nil {
		return nil, nil, err
	}
	perms, err := permissionsFromRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return perms, &customRole, nil
}

func (s *AdminService) HasPermission(ctx context.Context, tenantID, userID, baseRole, permission string) (bool, error) {
	perms, _, err := s.GetEffectivePermissions(ctx, tenantID, userID, baseRole)
	if err != nil {
		return false, err
	}
	for _, perm := range perms {
		if perm == permission {
			return true, nil
		}
	}
	return false, nil
}

func (s *AdminService) ListCustomRoles(ctx context.Context, tenantID string) ([]models.CustomRole, error) {
	rows, err := s.db.Query(ctx, `
		SELECT cr.id, cr.tenant_id, cr.name, cr.description, cr.is_active, cr.created_at, cr.updated_at,
			COALESCE((SELECT COUNT(*) FROM custom_role_assignments cra WHERE cra.custom_role_id = cr.id), 0) AS assigned_users
		FROM custom_roles cr
		WHERE cr.tenant_id = $1 AND cr.deleted_at IS NULL
		ORDER BY cr.name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []models.CustomRole{}
	for rows.Next() {
		var role models.CustomRole
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Name, &role.Description, &role.IsActive, &role.CreatedAt, &role.UpdatedAt, &role.AssignedUsers); err != nil {
			return nil, err
		}
		permRows, err := s.db.Query(ctx, `SELECT permission FROM custom_role_permissions WHERE custom_role_id = $1 ORDER BY permission ASC`, role.ID)
		if err != nil {
			return nil, err
		}
		role.Permissions, err = permissionsFromRows(permRows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (s *AdminService) CreateCustomRole(ctx context.Context, tenantID, actorUserID string, role models.CustomRoleUpsert) (*models.CustomRole, error) {
	id := uuid.New()
	tenantUUID := uuid.MustParse(tenantID)
	actorUUID := uuid.MustParse(actorUserID)
	name := strings.TrimSpace(role.Name)
	if name == "" {
		return nil, errors.New("role name is required")
	}
	permissions := normalizeStringList(role.Permissions)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO custom_roles (id, tenant_id, name, description, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, tenantUUID, name, emptyToNil(role.Description), role.IsActive, actorUUID); err != nil {
		return nil, err
	}
	for _, permission := range permissions {
		if _, err := tx.Exec(ctx, `INSERT INTO custom_role_permissions (custom_role_id, permission) VALUES ($1, $2)`, id, permission); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getCustomRole(ctx, tenantID, id.String())
}

func emptyToNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *AdminService) getCustomRole(ctx context.Context, tenantID, roleID string) (*models.CustomRole, error) {
	row := s.db.QueryRow(ctx, `
		SELECT cr.id, cr.tenant_id, cr.name, cr.description, cr.is_active, cr.created_at, cr.updated_at,
			COALESCE((SELECT COUNT(*) FROM custom_role_assignments cra WHERE cra.custom_role_id = cr.id), 0) AS assigned_users
		FROM custom_roles cr
		WHERE cr.id = $1 AND cr.tenant_id = $2 AND cr.deleted_at IS NULL
	`, roleID, tenantID)
	var role models.CustomRole
	if err := row.Scan(&role.ID, &role.TenantID, &role.Name, &role.Description, &role.IsActive, &role.CreatedAt, &role.UpdatedAt, &role.AssignedUsers); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomRoleNotFound
		}
		return nil, err
	}
	permRows, err := s.db.Query(ctx, `SELECT permission FROM custom_role_permissions WHERE custom_role_id = $1 ORDER BY permission ASC`, role.ID)
	if err != nil {
		return nil, err
	}
	role.Permissions, err = permissionsFromRows(permRows)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *AdminService) UpdateCustomRole(ctx context.Context, tenantID, roleID string, payload models.CustomRoleUpsert) (*models.CustomRole, error) {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return nil, errors.New("role name is required")
	}
	permissions := normalizeStringList(payload.Permissions)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	cmd, err := tx.Exec(ctx, `
		UPDATE custom_roles
		SET name = $3, description = $4, is_active = $5, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, roleID, tenantID, name, emptyToNil(payload.Description), payload.IsActive)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, ErrCustomRoleNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM custom_role_permissions WHERE custom_role_id = $1`, roleID); err != nil {
		return nil, err
	}
	for _, permission := range permissions {
		if _, err := tx.Exec(ctx, `INSERT INTO custom_role_permissions (custom_role_id, permission) VALUES ($1, $2)`, roleID, permission); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.getCustomRole(ctx, tenantID, roleID)
}

func (s *AdminService) DeleteCustomRole(ctx context.Context, tenantID, roleID string) error {
	cmd, err := s.db.Exec(ctx, `
		UPDATE custom_roles
		SET deleted_at = NOW(), updated_at = NOW(), is_active = false
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, roleID, tenantID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrCustomRoleNotFound
	}
	return nil
}

func (s *AdminService) ListRoleAssignments(ctx context.Context, tenantID string) ([]models.CustomRoleAssignment, error) {
	rows, err := s.db.Query(ctx, `
		SELECT cra.user_id, u.first_name, u.last_name, u.email, u.role, cra.custom_role_id, cr.name, cra.assigned_at
		FROM custom_role_assignments cra
		JOIN users u ON u.id = cra.user_id
		JOIN custom_roles cr ON cr.id = cra.custom_role_id
		WHERE cr.tenant_id = $1 AND u.deleted_at IS NULL AND cr.deleted_at IS NULL
		ORDER BY u.first_name ASC, u.last_name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assignments := []models.CustomRoleAssignment{}
	for rows.Next() {
		var assignment models.CustomRoleAssignment
		if err := rows.Scan(&assignment.UserID, &assignment.FirstName, &assignment.LastName, &assignment.Email, &assignment.BaseRole, &assignment.CustomRoleID, &assignment.CustomRoleName, &assignment.AssignedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	return assignments, nil
}

func (s *AdminService) AssignCustomRole(ctx context.Context, tenantID, targetUserID string, customRoleID *string, actorUserID string) error {
	var baseRole string
	var isActive bool
	if err := s.db.QueryRow(ctx, `
		SELECT role::text, is_active
		FROM users
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, targetUserID, tenantID).Scan(&baseRole, &isActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if !isActive || (baseRole != "org_admin" && baseRole != "hr" && baseRole != "dept_manager") {
		return ErrCustomRoleUserIneligible
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM custom_role_assignments WHERE user_id = $1`, targetUserID); err != nil {
		return err
	}
	if customRoleID != nil && strings.TrimSpace(*customRoleID) != "" {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM custom_roles WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL AND is_active = true)`, *customRoleID, tenantID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrCustomRoleNotFound
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO custom_role_assignments (user_id, custom_role_id, assigned_by)
			VALUES ($1, $2, $3)
		`, targetUserID, *customRoleID, actorUserID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *AdminService) ListSessions(ctx context.Context, tenantID string) ([]models.AuthSession, error) {
	rows, err := s.db.Query(ctx, `
		SELECT s.id, s.tenant_id, s.user_id, u.first_name, u.last_name, u.email, u.role::text,
			s.ip_address::text, s.user_agent, s.last_seen_at, s.expires_at, s.revoked_at, s.created_at
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.tenant_id = $1
		ORDER BY s.last_seen_at DESC, s.created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []models.AuthSession
	for rows.Next() {
		var session models.AuthSession
		if err := rows.Scan(
			&session.ID, &session.TenantID, &session.UserID, &session.FirstName, &session.LastName, &session.Email, &session.Role,
			&session.IPAddress, &session.UserAgent, &session.LastSeenAt, &session.ExpiresAt, &session.RevokedAt, &session.CreatedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (s *AdminService) RevokeSession(ctx context.Context, tenantID, sessionID, actorUserID string) error {
	cmd, err := s.db.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = NOW(), revoked_by = $3
		WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL
	`, sessionID, tenantID, actorUserID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *AdminService) RevokeUserSessions(ctx context.Context, tenantID, targetUserID, actorUserID string) (int64, error) {
	cmd, err := s.db.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = NOW(), revoked_by = $3
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, tenantID, targetUserID, actorUserID)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}

func (s *AdminService) IsIPAllowed(ctx context.Context, tenantID, ipAddress string) (bool, []string, error) {
	security, err := s.GetSecuritySettings(ctx, tenantID)
	if err != nil {
		return true, nil, nil
	}
	trusted := security.TrustedIPRanges
	if len(trusted) == 0 {
		return true, nil, nil
	}
	ip := net.ParseIP(strings.TrimSpace(ipAddress))
	if ip == nil {
		return false, trusted, nil
	}
	for _, entry := range trusted {
		if parsed := net.ParseIP(entry); parsed != nil {
			if parsed.Equal(ip) {
				return true, nil, nil
			}
			continue
		}
		_, network, err := net.ParseCIDR(entry)
		if err == nil && network.Contains(ip) {
			return true, nil, nil
		}
	}
	return false, trusted, nil
}

func hashChallengeCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func (s *AdminService) ValidatePasswordPolicy(ctx context.Context, tenantID, password string) error {
	security, err := s.GetSecuritySettings(ctx, tenantID)
	if err != nil {
		return err
	}
	return ValidatePasswordWithPolicy(security.PasswordPolicy, password)
}

func ValidatePasswordWithPolicy(policy models.PasswordPolicySettings, password string) error {
	password = strings.TrimSpace(password)
	if len(password) < policy.MinLength {
		return fmt.Errorf("%w: minimum length is %d", ErrInvalidPasswordPolicy, policy.MinLength)
	}
	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range password {
		switch {
		case 'A' <= r && r <= 'Z':
			hasUpper = true
		case 'a' <= r && r <= 'z':
			hasLower = true
		case '0' <= r && r <= '9':
			hasDigit = true
		default:
			hasSymbol = true
		}
	}
	if policy.RequireUppercase && !hasUpper {
		return fmt.Errorf("%w: at least one uppercase letter is required", ErrInvalidPasswordPolicy)
	}
	if policy.RequireLowercase && !hasLower {
		return fmt.Errorf("%w: at least one lowercase letter is required", ErrInvalidPasswordPolicy)
	}
	if policy.RequireNumber && !hasDigit {
		return fmt.Errorf("%w: at least one number is required", ErrInvalidPasswordPolicy)
	}
	if policy.RequireSymbol && !hasSymbol {
		return fmt.Errorf("%w: at least one symbol is required", ErrInvalidPasswordPolicy)
	}
	return nil
}

func (s *AdminService) MFARequiredForRole(ctx context.Context, tenantID, role string) (bool, error) {
	security, err := s.GetSecuritySettings(ctx, tenantID)
	if err != nil {
		return false, nil
	}
	if !security.EnforceMFA {
		return false, nil
	}
	role = strings.ToLower(strings.TrimSpace(role))
	for _, candidate := range security.RequireMFAForRoles {
		if candidate == role {
			return true, nil
		}
	}
	return false, nil
}

func (s *AdminService) SessionTimeout(ctx context.Context, tenantID string) (time.Duration, error) {
	security, err := s.GetSecuritySettings(ctx, tenantID)
	if err != nil {
		return time.Duration(defaultTenantSettings().Security.SessionTimeoutMinutes) * time.Minute, nil
	}
	minutes := security.SessionTimeoutMinutes
	if minutes <= 0 {
		minutes = defaultTenantSettings().Security.SessionTimeoutMinutes
	}
	return time.Duration(minutes) * time.Minute, nil
}
