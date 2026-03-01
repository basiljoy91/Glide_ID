package handlers

import (
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
)

// ProcessHRMSWebhook processes an HRMS webhook
func ProcessHRMSWebhook(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		provider := c.Params("provider")

		var payload map[string]interface{}
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid payload",
			})
		}

		signature := c.Get("X-Webhook-Signature")
		if err := hrmsSvc.ProcessWebhook(c.Context(), tenantID, provider, payload, signature); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{"status": "processed"})
	}
}

// HRMSWebhook handles public HRMS webhooks
func HRMSWebhook(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		provider := c.Params("provider")
		tenantID := c.Get("X-Tenant-ID")

		var payload map[string]interface{}
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid payload",
			})
		}

		signature := c.Get("X-Webhook-Signature")
		if err := hrmsSvc.ProcessWebhook(c.Context(), tenantID, provider, payload, signature); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{"status": "processed"})
	}
}

// ListHRMSIntegrations lists HRMS integrations
func ListHRMSIntegrations(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Implementation
		return c.JSON(fiber.Map{"message": "Not implemented"})
	}
}

// CreateHRMSIntegration creates an HRMS integration
func CreateHRMSIntegration(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Implementation
		return c.JSON(fiber.Map{"message": "Not implemented"})
	}
}

// ExportTimesheet exports timesheet data
func ExportTimesheet(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

		var req struct {
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		data, err := hrmsSvc.ExportTimesheet(c.Context(), tenantID, req.StartDate, req.EndDate)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(data)
	}
}

