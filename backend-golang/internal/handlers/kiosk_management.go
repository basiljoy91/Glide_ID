package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// ListKiosks lists kiosks
func ListKiosks(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Not implemented"})
}

// CreateKiosk creates a kiosk
func CreateKiosk(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Not implemented"})
}

// UpdateKiosk updates a kiosk
func UpdateKiosk(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Not implemented"})
}

// RevokeKiosk revokes a kiosk
func RevokeKiosk(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Not implemented"})
}

