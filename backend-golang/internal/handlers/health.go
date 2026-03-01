package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// HealthCheck returns a lightweight health check response for silent ping
func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "enterprise-attendance-api",
		"version":   "1.0.0",
	})
}

