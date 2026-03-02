package handlers

import (
	"context"
	"time"

	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InitiateSSO handles SSO login initiation
func InitiateSSO(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email  string `json:"email"`
			Domain string `json:"domain"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Email is required",
			})
		}

		// Extract domain from email if not provided
		if req.Domain == "" {
			// Simple domain extraction (in production, use proper parsing)
			for i := len(req.Email) - 1; i >= 0; i-- {
				if req.Email[i] == '@' {
					req.Domain = req.Email[i+1:]
					break
				}
			}
		}

		if req.Domain == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Could not determine email domain",
			})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var tenantID string
		var ssoProvider *string
		err := db.QueryRow(ctx, `
			SELECT id::text, NULLIF(sso_provider, '')
			FROM tenants
			WHERE deleted_at IS NULL
			  AND settings->>'sso_domain' = $1
			LIMIT 1
		`, req.Domain).Scan(&tenantID, &ssoProvider)

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "SSO not configured for this domain",
			})
		}

		if ssoProvider == nil || *ssoProvider == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Tenant found but SSO provider not configured",
			})
		}

		// TODO: Implement real SAML/OIDC initiation. For now, we confirm lookup worked.
		return c.JSON(fiber.Map{
			"tenantId":    tenantID,
			"provider":    *ssoProvider,
			"redirectUrl": "",
			"message":     "SSO tenant lookup succeeded. SAML/OIDC initiation not implemented yet.",
		})
	}
}

// SSOCallback handles SSO callback from identity provider
func SSOCallback(authSvc *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract SAML response or OIDC tokens from query params/body
		// Verify signature
		// Extract user attributes
		// Find or create user in database
		// Generate JWT token
		// Redirect to dashboard

		return c.JSON(fiber.Map{
			"message": "SSO callback handler - to be implemented",
		})
	}
}

