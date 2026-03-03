package handlers

import (
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
			EmployeeID      string  `json:"employee_id"`
			Email           string  `json:"email"`
			Phone           *string `json:"phone"`
			FirstName       string  `json:"first_name"`
			LastName        string  `json:"last_name"`
			DepartmentID    *string `json:"department_id"`
			Designation     *string `json:"designation"`
			DateOfJoining   *string `json:"date_of_joining"`
			Role            string  `json:"role"`
			IsActive        *bool   `json:"is_active"`
			Password        *string `json:"password"`
			AuthMethod      *string `json:"auth_method"` // "password" | "sso"
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
			EmployeeID: body.EmployeeID,
			Email:      body.Email,
			Phone:      body.Phone,
			FirstName:  body.FirstName,
			LastName:   body.LastName,
			Designation: body.Designation,
			Role:       body.Role,
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
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
		auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &userUUID,
			TargetUserID: &user.ID,
			Action:       "user_created",
			ResourceType:  stringPtr("user"),
			ResourceID:   &user.ID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		})

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

// ListUsers lists all users
func ListUsers(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		users, err := userSvc.ListUsers(c.Context(), tenantID, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(users)
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
		auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &actorUUID,
			TargetUserID: &targetUUID,
			Action:       "user_updated",
			ResourceType: stringPtr("user"),
			ResourceID:   &targetUUID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		})

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
		auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &userUUID,
			TargetUserID: &targetUUID,
			Action:       "user_deleted",
			ResourceType:  stringPtr("user"),
			ResourceID:   &targetUUID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		})

		return c.JSON(fiber.Map{"message": "User deleted"})
	}
}

func stringPtr(s string) *string {
	return &s
}

