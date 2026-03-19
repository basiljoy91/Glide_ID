package middleware

import (
	"errors"
	"strings"

	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// JWTAuth middleware validates JWT tokens
func JWTAuth(jwtSecret string, authSvc *services.AuthService, adminSvc *services.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		// Store claims in context
		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("role", claims.Role)
		c.Locals("email", claims.Email)
		c.Locals("token_jti", claims.ID)

		if authSvc != nil {
			if err := authSvc.ValidateSession(c.Context(), claims.ID, claims.UserID, claims.TenantID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Session is no longer active",
					})
				}
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Failed to validate session",
				})
			}
		}

		if adminSvc != nil {
			allowed, _, err := adminSvc.IsIPAllowed(c.Context(), claims.TenantID, c.IP())
			if err != nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Failed to validate trusted network rules",
				})
			}
			if !allowed {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Access from this network is not allowed",
				})
			}
			permissions, customRole, err := adminSvc.GetEffectivePermissions(c.Context(), claims.TenantID, claims.UserID, claims.Role)
			if err != nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Failed to resolve permissions",
				})
			}
			c.Locals("permissions", permissions)
			c.Locals("has_custom_role", customRole != nil)
			if customRole != nil {
				c.Locals("custom_role_name", customRole.Name)
			}
		}

		return c.Next()
	}
}

// RequireAccess checks whether the user can access a route by default role or explicit permission.
func RequireAccess(permission string, allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role")
		if userRole == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User role not found",
			})
		}

		roleStr := userRole.(string)
		if roleStr == "super_admin" {
			return c.Next()
		}
		if hasPermission(c, permission) {
			return c.Next()
		}
		if hasCustomRole := c.Locals("has_custom_role"); hasCustomRole != nil && hasCustomRole.(bool) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}
		for _, allowedRole := range allowedRoles {
			if roleStr == allowedRole {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

// RequireRole preserves the existing behavior for routes that are still purely role-based.
func RequireRole(allowedRoles ...string) fiber.Handler {
	return RequireAccess("", allowedRoles...)
}

func hasPermission(c *fiber.Ctx, permission string) bool {
	if strings.TrimSpace(permission) == "" {
		return false
	}
	raw := c.Locals("permissions")
	if raw == nil {
		return false
	}
	permissions, ok := raw.([]string)
	if !ok {
		return false
	}
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}

// GetUserID retrieves user ID from context
func GetUserID(c *fiber.Ctx) string {
	if userID := c.Locals("user_id"); userID != nil {
		return userID.(string)
	}
	return ""
}

// GetTenantID retrieves tenant ID from context
func GetTenantID(c *fiber.Ctx) string {
	if tenantID := c.Locals("tenant_id"); tenantID != nil {
		return tenantID.(string)
	}
	return ""
}

// GetRole retrieves user role from context
func GetRole(c *fiber.Ctx) string {
	if role := c.Locals("role"); role != nil {
		return role.(string)
	}
	return ""
}

func GetPermissions(c *fiber.Ctx) []string {
	if permissions := c.Locals("permissions"); permissions != nil {
		if cast, ok := permissions.([]string); ok {
			return cast
		}
	}
	return nil
}
