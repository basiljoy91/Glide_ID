package handlers

import (
	"net/http"
	"time"

	"enterprise-attendance-api/internal/config"
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type enrollClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Scope    string `json:"scope"`
	jwt.RegisteredClaims
}

// GenerateEnrollToken issues a short-lived JWT for face enrollment of a specific user.
func GenerateEnrollToken(authSvc *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := middleware.GetRole(c)
		if role != "org_admin" && role != "hr" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
		}

		tenantID := middleware.GetTenantID(c)
		userID := c.Params("id")
		if tenantID == "" || userID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing tenant or user id"})
		}

		now := time.Now()
		claims := enrollClaims{
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
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to sign token"})
		}

		return c.JSON(fiber.Map{
			"token": signed,
		})
	}
}

// EnrollFace handles remote face enrollment given an enrollment token.
func EnrollFace(attSvc *services.AttendanceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Params("token")
		if tokenStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing enrollment token"})
		}

		cfg := config.Load()
		token, err := jwt.ParseWithClaims(tokenStr, &enrollClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}
		claims, ok := token.Claims.(*enrollClaims)
		if !ok || claims.Scope != "face_enroll" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token scope"})
		}

		var body struct {
			ImageBase64 string `json:"image_base64"`
		}
		if err := c.BodyParser(&body); err != nil || body.ImageBase64 == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image_base64 is required"})
		}

		// Call AI service to vectorize and store the face vector
		ctx := c.Context()
		err = attSvc.VectorizeAndStore(ctx, claims.TenantID, claims.UserID, body.ImageBase64)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(http.StatusOK).JSON(fiber.Map{"success": true})
	}
}

// EnrollInfo returns limited information about the target user for an enrollment token.
func EnrollInfo(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Params("token")
		if tokenStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing enrollment token"})
		}

		cfg := config.Load()
		token, err := jwt.ParseWithClaims(tokenStr, &enrollClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}
		claims, ok := token.Claims.(*enrollClaims)
		if !ok || claims.Scope != "face_enroll" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token scope"})
		}

		user, err := userSvc.GetUser(c.Context(), claims.TenantID, claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found for enrollment token"})
		}

		return c.JSON(fiber.Map{
			"id":          user.ID,
			"employee_id": user.EmployeeID,
			"first_name":  user.FirstName,
			"last_name":   user.LastName,
			"role":        user.Role,
		})
	}
}

