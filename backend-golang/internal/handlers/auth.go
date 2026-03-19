package handlers

import (
	"errors"
	"strings"
	"time"

	"enterprise-attendance-api/internal/models"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// Login handles user login
func Login(authSvc *services.AuthService, userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		req.Email = strings.TrimSpace(req.Email)

		if req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Email and password are required",
			})
		}

		user, err := userSvc.FindLoginUserByEmail(c.Context(), req.Email)
		if err != nil {
			if errors.Is(err, services.ErrAmbiguousLoginUser) {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "Multiple organizations use this email. Contact your administrator to consolidate access before using password login.",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}
		if !user.IsActive || user.DeletedAt != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "This account is inactive",
			})
		}

		// Verify password if user has password auth enabled
		if user.PasswordHash == nil || *user.PasswordHash == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Password login not enabled for this user",
			})
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}
		if err := userSvc.UpdateLastLogin(c.Context(), user.TenantID.String(), user.ID.String()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to finalize login",
			})
		}

		// Generate token
		token, err := authSvc.GenerateToken(
			user.ID.String(),
			user.TenantID.String(),
			user.Role,
			user.Email,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate token",
			})
		}

		tenantID := user.TenantID
		userID := user.ID
		action := "admin_login"
		resourceType := "user"
		now := time.Now().UTC()
		_ = auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantID,
			UserID:       &userID,
			TargetUserID: &userID,
			Action:       action,
			ResourceType: &resourceType,
			ResourceID:   &userID,
			Details: map[string]interface{}{
				"role":  user.Role,
				"email": user.Email,
			},
			IPAddress: func(v string) *string { return &v }(c.IP()),
			UserAgent: func(v string) *string { return &v }(c.Get("User-Agent")),
			CreatedAt: now,
		})

		return c.JSON(fiber.Map{
			"token": token,
			"user":  user,
		})
	}
}
