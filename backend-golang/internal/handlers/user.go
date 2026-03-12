package handlers

import (
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"log"
	"strings"
	"time"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/models"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateUser creates a new user
func CreateUser(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)

		var body struct {
			EmployeeID    string  `json:"employee_id"`
			Email         string  `json:"email"`
			Phone         *string `json:"phone"`
			FirstName     string  `json:"first_name"`
			LastName      string  `json:"last_name"`
			DepartmentID  *string `json:"department_id"`
			Designation   *string `json:"designation"`
			DateOfJoining *string `json:"date_of_joining"`
			Role          string  `json:"role"`
			IsActive      *bool   `json:"is_active"`
			Password      *string `json:"password"`
			AuthMethod    *string `json:"auth_method"` // "password" | "sso"
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if body.EmployeeID == "" || body.Email == "" || body.FirstName == "" || body.LastName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "employee_id, email, first_name and last_name are required",
			})
		}

		// Default role to employee
		if body.Role == "" {
			body.Role = "employee"
		}

		// Enforce RBAC on role assignment
		if actorRole != "org_admin" {
			// Non-org_admins cannot create org_admins
			if body.Role == "org_admin" {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Only Org Admin can create Org Admin users",
				})
			}
		}

		// High-privilege roles should normally use password login unless SSO-only
		needsPassword := body.Role == "org_admin" || body.Role == "hr" || body.Role == "dept_manager"
		authMethod := "password"
		if body.AuthMethod != nil && *body.AuthMethod != "" {
			authMethod = *body.AuthMethod
		}
		if body.Role == "employee" {
			// Employees should use kiosk biometrics/PIN; avoid local password storage by default.
			authMethod = "sso"
		}
		if needsPassword && authMethod == "password" && (body.Password == nil || *body.Password == "") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Password is required for admin/HR roles when auth_method is password",
			})
		}

		var passwordHash *string
		if body.Password != nil && *body.Password != "" {
			h, err := bcrypt.GenerateFromPassword([]byte(*body.Password), bcrypt.DefaultCost)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to hash password",
				})
			}
			s := string(h)
			passwordHash = &s
		}

		// Map into models.User
		now := time.Now()
		user := models.User{
			EmployeeID:  body.EmployeeID,
			Email:       body.Email,
			Phone:       body.Phone,
			FirstName:   body.FirstName,
			LastName:    body.LastName,
			Designation: body.Designation,
			Role:        body.Role,
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if body.IsActive != nil {
			user.IsActive = *body.IsActive
		}
		if body.DepartmentID != nil && *body.DepartmentID != "" {
			if depID, err := uuid.Parse(*body.DepartmentID); err == nil {
				user.DepartmentID = &depID
			}
		}
		if body.DateOfJoining != nil && *body.DateOfJoining != "" {
			if dt, err := time.Parse(time.RFC3339, *body.DateOfJoining); err == nil {
				user.DateOfJoining = dt
			}
		} else {
			user.DateOfJoining = now
		}
		user.PasswordHash = passwordHash

		if err := userSvc.CreateUser(c.Context(), tenantID, &user); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Log audit
		tenantUUID := uuid.MustParse(tenantID)
		userUUID := uuid.MustParse(userID)
		if err := auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &userUUID,
			TargetUserID: &user.ID,
			Action:       "user_created",
			ResourceType: stringPtr("user"),
			ResourceID:   &user.ID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		}); err != nil {
			log.Printf("audit log failed for user_created: tenant=%s target=%s err=%v", tenantID, user.ID.String(), err)
		}

		return c.Status(fiber.StatusCreated).JSON(user)
	}
}

// GetUser gets a user by ID
func GetUser(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := c.Params("id")

		user, err := userSvc.GetUser(c.Context(), tenantID, userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		return c.JSON(user)
	}
}

type listUsersResponse struct {
	Data []*models.User `json:"data"`
	Meta struct {
		Total int `json:"total"`
		Page  int `json:"page"`
		Limit int `json:"limit"`
	} `json:"meta"`
}

// ListUsers lists all users
func ListUsers(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		limit := c.QueryInt("limit", 50)
		if limit <= 0 {
			limit = 50
		}
		if limit > 200 {
			limit = 200
		}
		page := c.QueryInt("page", 1)
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * limit
		query := c.Query("q", "")
		role := c.Query("role", "")
		status := c.Query("status", "")
		sortBy := c.Query("sort_by", "created_at")
		sortDir := c.Query("sort_dir", "desc")

		users, total, err := userSvc.ListUsers(c.Context(), tenantID, limit, offset, query, role, status, sortBy, sortDir)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		var resp listUsersResponse
		resp.Data = users
		resp.Meta.Total = total
		resp.Meta.Page = page
		resp.Meta.Limit = limit
		return c.JSON(resp)
	}
}

// UpdateUser updates a user
func UpdateUser(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		targetUserID := c.Params("id")

		var body models.User
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		updated, err := userSvc.UpdateUserBasic(c.Context(), tenantID, targetUserID, &body)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		tenantUUID := uuid.MustParse(tenantID)
		actorUUID := uuid.MustParse(actorUserID)
		targetUUID := uuid.MustParse(targetUserID)
		if err := auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &actorUUID,
			TargetUserID: &targetUUID,
			Action:       "user_updated",
			ResourceType: stringPtr("user"),
			ResourceID:   &targetUUID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		}); err != nil {
			log.Printf("audit log failed for user_updated: tenant=%s target=%s err=%v", tenantID, targetUserID, err)
		}

		return c.JSON(updated)
	}
}

// DeleteUser deletes a user
func DeleteUser(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := middleware.GetUserID(c)
		targetUserID := c.Params("id")

		// Soft delete user
		if err := userSvc.SoftDeleteUser(c.Context(), tenantID, targetUserID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Log audit
		tenantUUID := uuid.MustParse(tenantID)
		userUUID := uuid.MustParse(userID)
		targetUUID := uuid.MustParse(targetUserID)
		if err := auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &userUUID,
			TargetUserID: &targetUUID,
			Action:       "user_deleted",
			ResourceType: stringPtr("user"),
			ResourceID:   &targetUUID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		}); err != nil {
			log.Printf("audit log failed for user_deleted: tenant=%s target=%s err=%v", tenantID, targetUserID, err)
		}

		return c.JSON(fiber.Map{"message": "User deleted"})
	}
}

// ExportUsersCSV exports users with filters to CSV.
func ExportUsersCSV(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		query := c.Query("q", "")
		role := c.Query("role", "")
		status := c.Query("status", "")
		sortBy := c.Query("sort_by", "created_at")
		sortDir := c.Query("sort_dir", "desc")

		users, _, err := userSvc.ListUsers(c.Context(), tenantID, 5000, 0, query, role, status, sortBy, sortDir)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", "attachment; filename=users.csv")
		writer := csv.NewWriter(c.Response().BodyWriter())
		_ = writer.Write([]string{
			"employee_id",
			"first_name",
			"last_name",
			"email",
			"role",
			"status",
			"department_id",
			"designation",
			"phone",
			"last_login_at",
			"last_check_in_at",
		})
		for _, u := range users {
			statusVal := "inactive"
			if u.IsActive {
				statusVal = "active"
			}
			lastLogin := ""
			if u.LastLoginAt != nil {
				lastLogin = u.LastLoginAt.Format(time.RFC3339)
			}
			lastCheckIn := ""
			if u.LastCheckInAt != nil {
				lastCheckIn = u.LastCheckInAt.Format(time.RFC3339)
			}
			dept := ""
			if u.DepartmentID != nil {
				dept = u.DepartmentID.String()
			}
			_ = writer.Write([]string{
				u.EmployeeID,
				u.FirstName,
				u.LastName,
				u.Email,
				u.Role,
				statusVal,
				dept,
				stringOrEmpty(u.Designation),
				stringOrEmpty(u.Phone),
				lastLogin,
				lastCheckIn,
			})
		}
		writer.Flush()
		return nil
	}
}

// ResetUserPassword generates a temporary password for a user.
func ResetUserPassword(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		targetUserID := c.Params("id")

		tempPassword, err := generateTempPassword(12)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate temporary password",
			})
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to hash password",
			})
		}

		if err := userSvc.SetUserPasswordHash(c.Context(), tenantID, targetUserID, string(hash)); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		tenantUUID := uuid.MustParse(tenantID)
		actorUUID := uuid.MustParse(actorUserID)
		targetUUID := uuid.MustParse(targetUserID)
		if err := auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &actorUUID,
			TargetUserID: &targetUUID,
			Action:       "user_password_reset",
			ResourceType: stringPtr("user"),
			ResourceID:   &targetUUID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		}); err != nil {
			log.Printf("audit log failed for user_password_reset: tenant=%s target=%s err=%v", tenantID, targetUserID, err)
		}

		return c.JSON(fiber.Map{
			"temporary_password": tempPassword,
		})
	}
}

func generateTempPassword(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$"
	if length <= 0 {
		length = 12
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i := range b {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out), nil
}

func stringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func stringPtr(s string) *string {
	return &s
}

// BulkImportUsers creates users in batch.
func BulkImportUsers(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)

		var body struct {
			Rows []struct {
				EmployeeID    string  `json:"employee_id"`
				Email         string  `json:"email"`
				Phone         *string `json:"phone"`
				FirstName     string  `json:"first_name"`
				LastName      string  `json:"last_name"`
				DepartmentID  *string `json:"department_id"`
				Designation   *string `json:"designation"`
				DateOfJoining *string `json:"date_of_joining"`
				Role          string  `json:"role"`
				IsActive      *bool   `json:"is_active"`
				Password      *string `json:"password"`
				AuthMethod    *string `json:"auth_method"`
			} `json:"rows"`
		}
		if err := c.BodyParser(&body); err != nil || len(body.Rows) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "rows array is required",
			})
		}
		if len(body.Rows) > 1000 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "bulk import limit is 1000 rows per request",
			})
		}

		type rowResult struct {
			Row        int     `json:"row"`
			EmployeeID *string `json:"employee_id,omitempty"`
			UserID     *string `json:"user_id,omitempty"`
			Error      *string `json:"error,omitempty"`
		}
		results := make([]rowResult, 0, len(body.Rows))
		success := 0
		failed := 0
		now := time.Now()

		for i, r := range body.Rows {
			rowNum := i + 1
			if strings.TrimSpace(r.EmployeeID) == "" || strings.TrimSpace(r.Email) == "" || strings.TrimSpace(r.FirstName) == "" || strings.TrimSpace(r.LastName) == "" {
				errMsg := "employee_id, email, first_name and last_name are required"
				results = append(results, rowResult{Row: rowNum, EmployeeID: &r.EmployeeID, Error: &errMsg})
				failed++
				continue
			}

			role := r.Role
			if role == "" {
				role = "employee"
			}
			if actorRole != "org_admin" && role == "org_admin" {
				errMsg := "only Org Admin can create Org Admin users"
				results = append(results, rowResult{Row: rowNum, EmployeeID: &r.EmployeeID, Error: &errMsg})
				failed++
				continue
			}

			authMethod := "password"
			if r.AuthMethod != nil && *r.AuthMethod != "" {
				authMethod = *r.AuthMethod
			}
			if role == "employee" {
				authMethod = "sso"
			}
			needsPassword := role == "org_admin" || role == "hr" || role == "dept_manager"
			if needsPassword && authMethod == "password" && (r.Password == nil || strings.TrimSpace(*r.Password) == "") {
				errMsg := "password is required for admin/HR/manager users when auth_method is password"
				results = append(results, rowResult{Row: rowNum, EmployeeID: &r.EmployeeID, Error: &errMsg})
				failed++
				continue
			}

			var passwordHash *string
			if r.Password != nil && strings.TrimSpace(*r.Password) != "" {
				h, err := bcrypt.GenerateFromPassword([]byte(*r.Password), bcrypt.DefaultCost)
				if err != nil {
					errMsg := "failed to hash password"
					results = append(results, rowResult{Row: rowNum, EmployeeID: &r.EmployeeID, Error: &errMsg})
					failed++
					continue
				}
				s := string(h)
				passwordHash = &s
			}

			u := models.User{
				EmployeeID:   strings.TrimSpace(r.EmployeeID),
				Email:        strings.TrimSpace(r.Email),
				Phone:        r.Phone,
				FirstName:    strings.TrimSpace(r.FirstName),
				LastName:     strings.TrimSpace(r.LastName),
				Designation:  r.Designation,
				Role:         role,
				IsActive:     true,
				PasswordHash: passwordHash,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if r.IsActive != nil {
				u.IsActive = *r.IsActive
			}
			if r.DepartmentID != nil && *r.DepartmentID != "" {
				depID, err := uuid.Parse(*r.DepartmentID)
				if err == nil {
					u.DepartmentID = &depID
				}
			}
			u.DateOfJoining = now
			if r.DateOfJoining != nil && strings.TrimSpace(*r.DateOfJoining) != "" {
				if dt, err := parseJoiningDate(*r.DateOfJoining); err == nil {
					u.DateOfJoining = dt
				}
			}

			if err := userSvc.CreateUser(c.Context(), tenantID, &u); err != nil {
				errText := err.Error()
				results = append(results, rowResult{Row: rowNum, EmployeeID: &r.EmployeeID, Error: &errText})
				failed++
				continue
			}

			tenantUUID := uuid.MustParse(tenantID)
			actorUUID := uuid.MustParse(actorUserID)
			_ = auditSvc.LogAction(c.Context(), &models.AuditLog{
				TenantID:     &tenantUUID,
				UserID:       &actorUUID,
				TargetUserID: &u.ID,
				Action:       "user_created",
				ResourceType: stringPtr("user"),
				ResourceID:   &u.ID,
				IPAddress:    stringPtr(c.IP()),
				UserAgent:    stringPtr(c.Get("User-Agent")),
				CreatedAt:    now,
			})

			id := u.ID.String()
			results = append(results, rowResult{
				Row:        rowNum,
				EmployeeID: &u.EmployeeID,
				UserID:     &id,
			})
			success++
		}

		return c.JSON(fiber.Map{
			"success_count": success,
			"failed_count":  failed,
			"results":       results,
		})
	}
}

// BulkUserAction applies activate/deactivate/delete to multiple users.
func BulkUserAction(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)

		var body struct {
			UserIDs []string `json:"user_ids"`
			Action  string   `json:"action"` // activate | deactivate | delete
		}
		if err := c.BodyParser(&body); err != nil || len(body.UserIDs) == 0 || body.Action == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_ids and action are required",
			})
		}

		action := strings.ToLower(strings.TrimSpace(body.Action))
		var affected int64
		var err error
		switch action {
		case "activate":
			affected, err = userSvc.BulkSetActive(c.Context(), tenantID, body.UserIDs, true)
		case "deactivate":
			affected, err = userSvc.BulkSetActive(c.Context(), tenantID, body.UserIDs, false)
		case "delete":
			affected, err = userSvc.BulkSoftDelete(c.Context(), tenantID, body.UserIDs)
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "action must be activate, deactivate, or delete",
			})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		tenantUUID := uuid.MustParse(tenantID)
		actorUUID := uuid.MustParse(actorUserID)
		_ = auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &actorUUID,
			Action:       "user_updated",
			ResourceType: stringPtr("user"),
			Details: map[string]interface{}{
				"bulk_action": action,
				"count":       affected,
			},
			IPAddress: stringPtr(c.IP()),
			UserAgent: stringPtr(c.Get("User-Agent")),
			CreatedAt: time.Now(),
		})

		return c.JSON(fiber.Map{
			"success":  true,
			"action":   action,
			"affected": affected,
		})
	}
}

func parseJoiningDate(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unsupported date: %s", v)
}
