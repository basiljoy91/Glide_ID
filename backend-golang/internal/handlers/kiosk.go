package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// KioskHeartbeat handles kiosk heartbeat
func KioskHeartbeat(c *fiber.Ctx) error {
	kioskCode := c.Locals("kiosk_code")
	return c.JSON(fiber.Map{
		"status":     "ok",
		"kiosk_code": kioskCode,
	})
}

