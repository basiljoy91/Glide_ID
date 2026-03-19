package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/config"
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/models"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type employeeSummary struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
}

type employeeProfileResponse struct {
	User              *models.User              `json:"user"`
	DepartmentName    *string                   `json:"department_name"`
	Manager           *employeeSummary          `json:"manager"`
	DirectReports     []employeeSummary         `json:"direct_reports"`
	EmergencyContacts []models.EmergencyContact `json:"emergency_contacts"`
	Documents         []models.EmployeeDocument `json:"documents"`
}

type bulkEditChanges struct {
	DepartmentID   *string `json:"department_id"`
	ManagerID      *string `json:"manager_id"`
	EmploymentType *string `json:"employment_type"`
	WorkLocation   *string `json:"work_location"`
	CostCenter     *string `json:"cost_center"`
	Designation    *string `json:"designation"`
	IsActive       *bool   `json:"is_active"`
}

type bulkPreviewRequest struct {
	UserIDs []string        `json:"user_ids"`
	Changes bulkEditChanges `json:"changes"`
}

type bulkEditPreviewRow struct {
	UserID      uuid.UUID      `json:"user_id"`
	EmployeeID  string         `json:"employee_id"`
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	BeforeState map[string]any `json:"before_state"`
	AfterState  map[string]any `json:"after_state"`
}

type bulkChangeBatchResponse struct {
	ID         uuid.UUID            `json:"id"`
	ChangeType string               `json:"change_type"`
	Status     string               `json:"status"`
	Summary    map[string]any       `json:"summary"`
	AppliedAt  *time.Time           `json:"applied_at"`
	RolledBack *time.Time           `json:"rolled_back_at"`
	CreatedAt  time.Time            `json:"created_at"`
	Items      []bulkEditPreviewRow `json:"items,omitempty"`
}

type enrollInviteClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Scope    string `json:"scope"`
	jwt.RegisteredClaims
}

func parseOptionalUUID(value *string) (*uuid.UUID, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalDate(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*value))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalDateTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func nullableString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func buildEnrollInviteLink(tenantID, userID string) (string, string, error) {
	now := time.Now().UTC()
	claims := enrollInviteClaims{
		UserID:   userID,
		TenantID: tenantID,
		Scope:    "face_enroll",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			Issuer:    "enterprise-attendance-api",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	cfg := config.Load()
	signed, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}
	baseURL := ""
	if len(cfg.CORSOrigins) > 0 {
		baseURL = strings.TrimRight(strings.TrimSpace(cfg.CORSOrigins[0]), "/")
	}
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	return signed, fmt.Sprintf("%s/enroll/%s", baseURL, signed), nil
}

func loadEmployeeProfile(ctx context.Context, db *pgxpool.Pool, tenantID, targetUserID string) (*employeeProfileResponse, error) {
	user := &models.User{}
	var departmentName *string
	var managerID *uuid.UUID
	var managerFirst, managerLast, managerEmail *string
	var managerRole *string
	if err := db.QueryRow(ctx, `
		SELECT
			u.id, u.tenant_id, u.employee_id, u.email, u.phone, u.first_name, u.last_name,
			u.department_id, u.designation, u.date_of_joining, u.shift_start_time, u.shift_end_time,
			u.shift_length_hours, u.role, u.is_active, u.data_privacy_consent, u.consent_date,
			u.last_login_at, u.manager_id, u.employment_type, u.work_location, u.cost_center,
			u.invite_status, u.invite_sent_at, u.offboarded_at, u.offboarding_reason, u.created_at, u.updated_at, u.deleted_at,
			d.name,
			m.id, m.first_name, m.last_name, m.email, m.role::text
		FROM users u
		LEFT JOIN departments d ON d.id = u.department_id AND d.deleted_at IS NULL
		LEFT JOIN users m ON m.id = u.manager_id AND m.deleted_at IS NULL
		WHERE u.tenant_id = $1 AND u.id = $2 AND u.deleted_at IS NULL
	`, tenantID, targetUserID).Scan(
		&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
		&user.DepartmentID, &user.Designation, &user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
		&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent, &user.ConsentDate,
		&user.LastLoginAt, &user.ManagerID, &user.EmploymentType, &user.WorkLocation, &user.CostCenter,
		&user.InviteStatus, &user.InviteSentAt, &user.OffboardedAt, &user.OffboardingReason, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&departmentName,
		&managerID, &managerFirst, &managerLast, &managerEmail, &managerRole,
	); err != nil {
		return nil, err
	}

	resp := &employeeProfileResponse{User: user, DepartmentName: departmentName}
	if managerID != nil && managerFirst != nil && managerLast != nil && managerEmail != nil && managerRole != nil {
		resp.Manager = &employeeSummary{ID: *managerID, FirstName: *managerFirst, LastName: *managerLast, Email: *managerEmail, Role: *managerRole}
	}

	contacts, err := listEmergencyContacts(ctx, db, targetUserID)
	if err != nil {
		return nil, err
	}
	resp.EmergencyContacts = contacts

	documents, err := listEmployeeDocuments(ctx, db, tenantID, targetUserID)
	if err != nil {
		return nil, err
	}
	resp.Documents = documents

	reportRows, err := db.Query(ctx, `
		SELECT id, first_name, last_name, email, role::text
		FROM users
		WHERE tenant_id = $1 AND manager_id = $2 AND deleted_at IS NULL
		ORDER BY first_name, last_name
	`, tenantID, targetUserID)
	if err != nil {
		return nil, err
	}
	defer reportRows.Close()
	resp.DirectReports = []employeeSummary{}
	for reportRows.Next() {
		var row employeeSummary
		if err := reportRows.Scan(&row.ID, &row.FirstName, &row.LastName, &row.Email, &row.Role); err != nil {
			return nil, err
		}
		resp.DirectReports = append(resp.DirectReports, row)
	}
	return resp, nil
}

func listEmergencyContacts(ctx context.Context, db *pgxpool.Pool, userID string) ([]models.EmergencyContact, error) {
	rows, err := db.Query(ctx, `
		SELECT id, user_id, name, relationship, phone, email, is_primary, created_at, updated_at
		FROM emergency_contacts
		WHERE user_id = $1
		ORDER BY is_primary DESC, created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	contacts := []models.EmergencyContact{}
	for rows.Next() {
		var item models.EmergencyContact
		if err := rows.Scan(&item.ID, &item.UserID, &item.Name, &item.Relationship, &item.Phone, &item.Email, &item.IsPrimary, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		contacts = append(contacts, item)
	}
	return contacts, nil
}

func listEmployeeDocuments(ctx context.Context, db *pgxpool.Pool, tenantID, userID string) ([]models.EmployeeDocument, error) {
	rows, err := db.Query(ctx, `
		SELECT id, tenant_id, user_id, document_type, name, file_url, expires_at, COALESCE(metadata, '{}'::jsonb), uploaded_by, created_at, updated_at
		FROM employee_documents
		WHERE tenant_id = $1 AND user_id = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	documents := []models.EmployeeDocument{}
	for rows.Next() {
		var item models.EmployeeDocument
		var rawMetadata []byte
		if err := rows.Scan(&item.ID, &item.TenantID, &item.UserID, &item.DocumentType, &item.Name, &item.FileURL, &item.ExpiresAt, &rawMetadata, &item.UploadedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(rawMetadata, &item.Metadata)
		documents = append(documents, item)
	}
	return documents, nil
}

func validateManagerHierarchy(ctx context.Context, db *pgxpool.Pool, tenantID string, targetUserID string, managerID *uuid.UUID) error {
	if managerID == nil {
		return nil
	}
	if managerID.String() == targetUserID {
		return errors.New("employee cannot report to themselves")
	}
	var exists bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users
			WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL AND is_active = true
		)
	`, *managerID, tenantID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("manager not found")
	}
	return nil
}

func GetEmployeeProfile(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		targetUserID := c.Params("id")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			switch {
			case errors.Is(err, errRoleMutationForbidden):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot view this employee profile"})
			default:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
		}
		resp, err := loadEmployeeProfile(c.Context(), userSvc.GetDB(), tenantID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load employee profile"})
		}
		return c.JSON(resp)
	}
}

func UpdateEmployeeProfile(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		actorUserID := middleware.GetUserID(c)
		targetUserID := c.Params("id")

		var body struct {
			Email          string  `json:"email"`
			Phone          *string `json:"phone"`
			FirstName      string  `json:"first_name"`
			LastName       string  `json:"last_name"`
			DepartmentID   *string `json:"department_id"`
			Designation    *string `json:"designation"`
			DateOfJoining  *string `json:"date_of_joining"`
			IsActive       *bool   `json:"is_active"`
			ManagerID      *string `json:"manager_id"`
			EmploymentType *string `json:"employment_type"`
			WorkLocation   *string `json:"work_location"`
			CostCenter     *string `json:"cost_center"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		currentUser, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil)
		if err != nil {
			switch {
			case errors.Is(err, errRoleMutationForbidden):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot modify this employee profile"})
			default:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
		}
		managerID, err := parseOptionalUUID(body.ManagerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid manager_id"})
		}
		if err := validateManagerHierarchy(c.Context(), userSvc.GetDB(), tenantID, targetUserID, managerID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		departmentID, err := parseOptionalUUID(body.DepartmentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid department_id"})
		}
		dateOfJoining := currentUser.DateOfJoining
		if body.DateOfJoining != nil && strings.TrimSpace(*body.DateOfJoining) != "" {
			parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*body.DateOfJoining))
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date_of_joining"})
			}
			dateOfJoining = parsed
		}
		isActive := currentUser.IsActive
		if body.IsActive != nil {
			isActive = *body.IsActive
		}
		if strings.TrimSpace(body.Email) == "" || strings.TrimSpace(body.FirstName) == "" || strings.TrimSpace(body.LastName) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email, first_name and last_name are required"})
		}

		_, err = userSvc.GetDB().Exec(c.Context(), `
			UPDATE users
			SET email = $1,
				phone = $2,
				first_name = $3,
				last_name = $4,
				department_id = $5,
				designation = $6,
				date_of_joining = $7,
				is_active = $8,
				manager_id = $9,
				employment_type = NULLIF($10, ''),
				work_location = $11,
				cost_center = $12,
				updated_at = NOW()
			WHERE tenant_id = $13 AND id = $14 AND deleted_at IS NULL
		`, strings.TrimSpace(body.Email), nullableString(body.Phone), strings.TrimSpace(body.FirstName), strings.TrimSpace(body.LastName), departmentID, nullableString(body.Designation), dateOfJoining, isActive, managerID, strings.TrimSpace(stringValueOrEmpty(body.EmploymentType)), nullableString(body.WorkLocation), nullableString(body.CostCenter), tenantID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update employee profile"})
		}

		updated, err := loadEmployeeProfile(c.Context(), userSvc.GetDB(), tenantID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Profile updated but failed to reload employee profile"})
		}
		if auditSvc != nil {
			tenantUUID := uuid.MustParse(tenantID)
			actorUUID := uuid.MustParse(actorUserID)
			targetUUID := uuid.MustParse(targetUserID)
			_ = auditSvc.LogAction(c.Context(), &models.AuditLog{TenantID: &tenantUUID, UserID: &actorUUID, TargetUserID: &targetUUID, Action: "user_updated", ResourceType: stringPtr("user"), ResourceID: &targetUUID, IPAddress: stringPtr(c.IP()), UserAgent: stringPtr(c.Get("User-Agent"))})
		}
		return c.JSON(updated)
	}
}

func ListEmployeeEmergencyContacts(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		targetUserID := c.Params("id")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot view these contacts"})
		}
		contacts, err := listEmergencyContacts(c.Context(), userSvc.GetDB(), targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load emergency contacts"})
		}
		return c.JSON(contacts)
	}
}

func CreateEmployeeEmergencyContact(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		targetUserID := c.Params("id")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot modify these contacts"})
		}
		var body struct {
			Name         string  `json:"name"`
			Relationship *string `json:"relationship"`
			Phone        string  `json:"phone"`
			Email        *string `json:"email"`
			IsPrimary    bool    `json:"is_primary"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Phone) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and phone are required"})
		}
		tx, err := userSvc.GetDB().Begin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start emergency contact update"})
		}
		defer tx.Rollback(c.Context())
		if body.IsPrimary {
			if _, err := tx.Exec(c.Context(), `UPDATE emergency_contacts SET is_primary = false, updated_at = NOW() WHERE user_id = $1`, targetUserID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update primary contact"})
			}
		}
		contactID := uuid.New()
		if _, err := tx.Exec(c.Context(), `
			INSERT INTO emergency_contacts (id, user_id, name, relationship, phone, email, is_primary)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, contactID, targetUserID, strings.TrimSpace(body.Name), nullableString(body.Relationship), strings.TrimSpace(body.Phone), nullableString(body.Email), body.IsPrimary); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create emergency contact"})
		}
		if err := tx.Commit(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save emergency contact"})
		}
		contacts, err := listEmergencyContacts(c.Context(), userSvc.GetDB(), targetUserID)
		if err != nil {
			return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": contactID})
		}
		return c.Status(fiber.StatusCreated).JSON(contacts)
	}
}

func UpdateEmployeeEmergencyContact(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		targetUserID := c.Params("id")
		contactID := c.Params("contactId")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot modify these contacts"})
		}
		var body struct {
			Name         string  `json:"name"`
			Relationship *string `json:"relationship"`
			Phone        string  `json:"phone"`
			Email        *string `json:"email"`
			IsPrimary    bool    `json:"is_primary"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		tx, err := userSvc.GetDB().Begin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start emergency contact update"})
		}
		defer tx.Rollback(c.Context())
		if body.IsPrimary {
			if _, err := tx.Exec(c.Context(), `UPDATE emergency_contacts SET is_primary = false, updated_at = NOW() WHERE user_id = $1`, targetUserID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update primary contact"})
			}
		}
		tag, err := tx.Exec(c.Context(), `
			UPDATE emergency_contacts
			SET name = $1, relationship = $2, phone = $3, email = $4, is_primary = $5, updated_at = NOW()
			WHERE id = $6 AND user_id = $7
		`, strings.TrimSpace(body.Name), nullableString(body.Relationship), strings.TrimSpace(body.Phone), nullableString(body.Email), body.IsPrimary, contactID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update emergency contact"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Emergency contact not found"})
		}
		if err := tx.Commit(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save emergency contact"})
		}
		contacts, err := listEmergencyContacts(c.Context(), userSvc.GetDB(), targetUserID)
		if err != nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{"success": true})
		}
		return c.JSON(contacts)
	}
}

func DeleteEmployeeEmergencyContact(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		targetUserID := c.Params("id")
		contactID := c.Params("contactId")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot modify these contacts"})
		}
		tag, err := userSvc.GetDB().Exec(c.Context(), `DELETE FROM emergency_contacts WHERE id = $1 AND user_id = $2`, contactID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete emergency contact"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Emergency contact not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func ListEmployeeDocuments(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		targetUserID := c.Params("id")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot view these documents"})
		}
		documents, err := listEmployeeDocuments(c.Context(), userSvc.GetDB(), tenantID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load employee documents"})
		}
		return c.JSON(documents)
	}
}

func CreateEmployeeDocument(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		actorUserID := middleware.GetUserID(c)
		targetUserID := c.Params("id")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot modify these documents"})
		}
		var body struct {
			DocumentType string         `json:"document_type"`
			Name         string         `json:"name"`
			FileURL      string         `json:"file_url"`
			ExpiresAt    *string        `json:"expires_at"`
			Metadata     map[string]any `json:"metadata"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if strings.TrimSpace(body.DocumentType) == "" || strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.FileURL) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "document_type, name and file_url are required"})
		}
		expiresAt, err := parseOptionalDateTime(body.ExpiresAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid expires_at"})
		}
		metadata, _ := json.Marshal(body.Metadata)
		_, err = userSvc.GetDB().Exec(c.Context(), `
			INSERT INTO employee_documents (id, tenant_id, user_id, document_type, name, file_url, expires_at, metadata, uploaded_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, uuid.New(), tenantID, targetUserID, strings.TrimSpace(body.DocumentType), strings.TrimSpace(body.Name), strings.TrimSpace(body.FileURL), expiresAt, metadata, actorUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save employee document"})
		}
		documents, err := listEmployeeDocuments(c.Context(), userSvc.GetDB(), tenantID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true})
		}
		return c.Status(fiber.StatusCreated).JSON(documents)
	}
}

func DeleteEmployeeDocument(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		targetUserID := c.Params("id")
		documentID := c.Params("documentId")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot modify these documents"})
		}
		tag, err := userSvc.GetDB().Exec(c.Context(), `
			UPDATE employee_documents
			SET deleted_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2 AND user_id = $3 AND deleted_at IS NULL
		`, documentID, tenantID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete employee document"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Employee document not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func ResendEmployeeInvite(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		actorUserID := middleware.GetUserID(c)
		targetUserID := c.Params("id")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot send an invite to this employee"})
		}
		token, inviteURL, err := buildEnrollInviteLink(tenantID, targetUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate invite"})
		}
		_, err = userSvc.GetDB().Exec(c.Context(), `
			UPDATE users
			SET invite_status = 'sent', invite_sent_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		`, targetUserID, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invite generated but failed to update invite status"})
		}
		if auditSvc != nil {
			tenantUUID := uuid.MustParse(tenantID)
			actorUUID := uuid.MustParse(actorUserID)
			targetUUID := uuid.MustParse(targetUserID)
			_ = auditSvc.LogAction(c.Context(), &models.AuditLog{TenantID: &tenantUUID, UserID: &actorUUID, TargetUserID: &targetUUID, Action: "user_updated", ResourceType: stringPtr("user"), ResourceID: &targetUUID, IPAddress: stringPtr(c.IP()), UserAgent: stringPtr(c.Get("User-Agent")), Details: map[string]any{"invite_status": "sent"}})
		}
		return c.JSON(fiber.Map{"token": token, "invite_url": inviteURL, "status": "sent"})
	}
}

func OffboardEmployee(userSvc *services.UserService, adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		actorUserID := middleware.GetUserID(c)
		targetUserID := c.Params("id")
		if _, err := authorizeUserMutation(c.Context(), userSvc, tenantID, actorRole, targetUserID, nil); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot offboard this employee"})
		}
		var body struct {
			Reason string `json:"reason"`
		}
		_ = c.BodyParser(&body)
		tx, err := userSvc.GetDB().Begin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start offboarding"})
		}
		defer tx.Rollback(c.Context())
		if _, err := tx.Exec(c.Context(), `
			UPDATE users
			SET is_active = false,
				offboarded_at = NOW(),
				offboarding_reason = $1,
				invite_status = 'revoked',
				manager_id = NULL,
				updated_at = NOW()
			WHERE tenant_id = $2 AND id = $3 AND deleted_at IS NULL
		`, strings.TrimSpace(body.Reason), tenantID, targetUserID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to offboard employee"})
		}
		if _, err := tx.Exec(c.Context(), `UPDATE users SET manager_id = NULL, updated_at = NOW() WHERE tenant_id = $1 AND manager_id = $2 AND deleted_at IS NULL`, tenantID, targetUserID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear direct report hierarchy"})
		}
		if err := tx.Commit(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize offboarding"})
		}
		if adminSvc != nil {
			_, _ = adminSvc.RevokeUserSessions(c.Context(), tenantID, targetUserID, actorUserID)
		}
		if auditSvc != nil {
			tenantUUID := uuid.MustParse(tenantID)
			actorUUID := uuid.MustParse(actorUserID)
			targetUUID := uuid.MustParse(targetUserID)
			_ = auditSvc.LogAction(c.Context(), &models.AuditLog{TenantID: &tenantUUID, UserID: &actorUUID, TargetUserID: &targetUUID, Action: "user_deactivated", ResourceType: stringPtr("user"), ResourceID: &targetUUID, IPAddress: stringPtr(c.IP()), UserAgent: stringPtr(c.Get("User-Agent")), Details: map[string]any{"reason": strings.TrimSpace(body.Reason)}})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func stringValueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func loadBulkTargetUsers(ctx context.Context, db *pgxpool.Pool, tenantID string, userIDs []string) ([]*models.User, error) {
	ids := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("invalid user id: %s", id)
		}
		ids = append(ids, parsed)
	}
	rows, err := db.Query(ctx, `
		SELECT id, tenant_id, employee_id, email, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, manager_id, employment_type, work_location, cost_center,
			invite_status, invite_sent_at, offboarded_at, offboarding_reason, created_at, updated_at, deleted_at
		FROM users
		WHERE tenant_id = $1 AND id = ANY($2::uuid[]) AND deleted_at IS NULL
	`, tenantID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.Phone, &user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation, &user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime, &user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent, &user.ConsentDate, &user.LastLoginAt, &user.ManagerID, &user.EmploymentType, &user.WorkLocation, &user.CostCenter, &user.InviteStatus, &user.InviteSentAt, &user.OffboardedAt, &user.OffboardingReason, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func buildBulkPreviewForUser(user *models.User, changes bulkEditChanges) (map[string]any, map[string]any, error) {
	if user.Role == "dept_manager" {
		return nil, nil, errors.New("bulk edit does not support department managers; use the profile page for manager-linked employees")
	}
	before := map[string]any{
		"department_id":   uuidToString(user.DepartmentID),
		"manager_id":      uuidToString(user.ManagerID),
		"employment_type": stringOrNil(user.EmploymentType),
		"work_location":   stringOrNil(user.WorkLocation),
		"cost_center":     stringOrNil(user.CostCenter),
		"designation":     stringOrNil(user.Designation),
		"is_active":       user.IsActive,
	}
	after := map[string]any{}
	for k, v := range before {
		after[k] = v
	}
	if changes.DepartmentID != nil {
		after["department_id"] = strings.TrimSpace(stringValueOrEmpty(changes.DepartmentID))
	}
	if changes.ManagerID != nil {
		after["manager_id"] = strings.TrimSpace(stringValueOrEmpty(changes.ManagerID))
	}
	if changes.EmploymentType != nil {
		after["employment_type"] = strings.TrimSpace(stringValueOrEmpty(changes.EmploymentType))
	}
	if changes.WorkLocation != nil {
		after["work_location"] = strings.TrimSpace(stringValueOrEmpty(changes.WorkLocation))
	}
	if changes.CostCenter != nil {
		after["cost_center"] = strings.TrimSpace(stringValueOrEmpty(changes.CostCenter))
	}
	if changes.Designation != nil {
		after["designation"] = strings.TrimSpace(stringValueOrEmpty(changes.Designation))
	}
	if changes.IsActive != nil {
		after["is_active"] = *changes.IsActive
	}
	return before, after, nil
}

func uuidToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func stringOrNil(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func PreviewBulkEmployeeEdit(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		actorUserID := middleware.GetUserID(c)
		var body bulkPreviewRequest
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if len(body.UserIDs) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_ids is required"})
		}
		if len(body.UserIDs) > 500 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bulk edit limit is 500 users"})
		}
		if err := authorizeBulkUserMutation(c.Context(), userSvc, tenantID, actorRole, body.UserIDs); err != nil {
			if errors.Is(err, errRoleMutationForbidden) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "You cannot bulk-edit one or more selected employees"})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		users, err := loadBulkTargetUsers(c.Context(), userSvc.GetDB(), tenantID, body.UserIDs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load selected employees"})
		}
		batchID := uuid.New()
		tx, err := userSvc.GetDB().Begin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start bulk edit preview"})
		}
		defer tx.Rollback(c.Context())
		summary := map[string]any{"user_count": len(users)}
		summaryJSON, _ := json.Marshal(summary)
		if _, err := tx.Exec(c.Context(), `INSERT INTO bulk_change_batches (id, tenant_id, created_by, change_type, status, summary) VALUES ($1, $2, $3, 'employee_bulk_edit', 'previewed', $4)`, batchID, tenantID, actorUserID, summaryJSON); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create bulk edit batch"})
		}
		items := []bulkEditPreviewRow{}
		for _, user := range users {
			before, after, err := buildBulkPreviewForUser(user, body.Changes)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			}
			if raw := fmt.Sprint(after["manager_id"]); raw != "" {
				managerID, err := uuid.Parse(raw)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid manager_id in changes"})
				}
				if err := validateManagerHierarchy(c.Context(), userSvc.GetDB(), tenantID, user.ID.String(), &managerID); err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
				}
			}
			beforeJSON, _ := json.Marshal(before)
			afterJSON, _ := json.Marshal(after)
			if _, err := tx.Exec(c.Context(), `INSERT INTO bulk_change_batch_items (id, batch_id, user_id, before_state, after_state) VALUES ($1, $2, $3, $4, $5)`, uuid.New(), batchID, user.ID, beforeJSON, afterJSON); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to store bulk edit preview"})
			}
			items = append(items, bulkEditPreviewRow{UserID: user.ID, EmployeeID: user.EmployeeID, FirstName: user.FirstName, LastName: user.LastName, BeforeState: before, AfterState: after})
		}
		if err := tx.Commit(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save bulk edit preview"})
		}
		return c.JSON(bulkChangeBatchResponse{ID: batchID, ChangeType: "employee_bulk_edit", Status: "previewed", Summary: summary, Items: items, CreatedAt: time.Now().UTC()})
	}
}

func ListBulkEmployeeEditBatches(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		rows, err := db.Query(c.Context(), `
			SELECT id, change_type, status, summary, applied_at, rolled_back_at, created_at
			FROM bulk_change_batches
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT 20
		`, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load bulk edit batches"})
		}
		defer rows.Close()
		batches := []bulkChangeBatchResponse{}
		for rows.Next() {
			var item bulkChangeBatchResponse
			var rawSummary []byte
			if err := rows.Scan(&item.ID, &item.ChangeType, &item.Status, &rawSummary, &item.AppliedAt, &item.RolledBack, &item.CreatedAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read bulk edit batches"})
			}
			_ = json.Unmarshal(rawSummary, &item.Summary)
			batches = append(batches, item)
		}
		return c.JSON(batches)
	}
}

func applyBulkChangeState(ctx context.Context, tx pgx.Tx, tenantID string, userID uuid.UUID, state map[string]any) error {
	var departmentID *uuid.UUID
	if raw := strings.TrimSpace(fmt.Sprint(state["department_id"])); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return err
		}
		departmentID = &parsed
	}
	var managerID *uuid.UUID
	if raw := strings.TrimSpace(fmt.Sprint(state["manager_id"])); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return err
		}
		managerID = &parsed
	}
	_, err := tx.Exec(ctx, `
		UPDATE users
		SET department_id = $1,
			manager_id = $2,
			employment_type = NULLIF($3, ''),
			work_location = NULLIF($4, ''),
			cost_center = NULLIF($5, ''),
			designation = NULLIF($6, ''),
			is_active = $7,
			updated_at = NOW()
		WHERE tenant_id = $8 AND id = $9 AND deleted_at IS NULL
	`, departmentID, managerID, strings.TrimSpace(fmt.Sprint(state["employment_type"])), strings.TrimSpace(fmt.Sprint(state["work_location"])), strings.TrimSpace(fmt.Sprint(state["cost_center"])), strings.TrimSpace(fmt.Sprint(state["designation"])), state["is_active"], tenantID, userID)
	return err
}

func mutateBulkEditBatchStatus(c *fiber.Ctx, db *pgxpool.Pool, fromStatus string, apply bool) error {
	tenantID := middleware.GetTenantID(c)
	batchID := c.Params("id")
	tx, err := db.Begin(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start bulk edit update")
	}
	defer tx.Rollback(c.Context())
	if err := mutateBulkEditBatchStatusTx(c.Context(), tx, tenantID, batchID, fromStatus, apply); err != nil {
		switch {
		case errors.Is(err, errBulkEditBatchNotFound):
			return fiber.NewError(fiber.StatusNotFound, "Bulk edit batch not found")
		case errors.Is(err, errBulkEditBatchInvalidFlow):
			if apply {
				return fiber.NewError(fiber.StatusBadRequest, "Bulk edit batch is not ready to apply")
			}
			return fiber.NewError(fiber.StatusBadRequest, "Bulk edit batch cannot be rolled back")
		default:
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	if err := tx.Commit(c.Context()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to finalize bulk edit update")
	}
	return nil
}

func ApplyBulkEmployeeEdit(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := mutateBulkEditBatchStatus(c, db, "previewed", true); err != nil {
			return err
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func RollbackBulkEmployeeEdit(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := mutateBulkEditBatchStatus(c, db, "applied", false); err != nil {
			return err
		}
		return c.JSON(fiber.Map{"success": true})
	}
}
