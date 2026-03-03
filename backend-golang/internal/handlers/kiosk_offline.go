package handlers

import (
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
)

// KioskOfflineSync accepts an encrypted offline payload, decrypts it server-side, and processes it.
// This route is HMAC authenticated (same as /kiosk/check-in).
func KioskOfflineSync(svc *services.AttendanceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		kioskCode := c.Locals("kiosk_code")
		kioskCodeStr, _ := kioskCode.(string)
		if kioskCodeStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Kiosk code not found"})
		}

		var body struct {
			EncryptedPayload string `json:"encrypted_payload"`
		}
		if err := c.BodyParser(&body); err != nil || body.EncryptedPayload == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "encrypted_payload is required"})
		}

		resp, err := svc.ProcessOfflineSync(c.Context(), tenantID, kioskCodeStr, body.EncryptedPayload)
		if err != nil {
			// If private key isn't configured, surface as 501 to make it obvious.
			if err.Error() == "offline decryption not configured" {
				return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(resp)
	}
}

