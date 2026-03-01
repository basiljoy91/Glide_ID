package handlers

import (
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
)

// Login handles user login
func Login(authSvc *services.AuthService, userSvc *services.UserService) fiber.Handler {
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

		// Verify credentials and get user
		// This is a simplified version - implement proper password verification
		// Note: tenantID should be determined from request or user lookup
		// For now, we'll need to modify GetUserByEmail to search across tenants for login
		// This is a placeholder - implement proper tenant resolution
		user, err := userSvc.GetUserByEmail(c.Context(), "", req.Email)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
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

		return c.JSON(fiber.Map{
			"token": token,
			"user":  user,
		})
	}
}

// SSOCallback handles SSO callback
func SSOCallback(authSvc *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Implementation for SSO callback
		return c.JSON(fiber.Map{"message": "SSO callback not implemented"})
	}
}

