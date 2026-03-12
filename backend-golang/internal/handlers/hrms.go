package handlers

import (
	"context"
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"
	"time"

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
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		list, err := hrmsSvc.ListIntegrations(ctx, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load integrations"})
		}

		return c.JSON(list)
	}
}

// CreateHRMSIntegration creates an HRMS integration
func CreateHRMSIntegration(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}

		var body services.UpsertIntegrationInput
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		out, err := hrmsSvc.UpsertIntegration(ctx, tenantID, body)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(fiber.StatusCreated).JSON(out)
	}
}

// UpdateHRMSIntegration updates integration fields.
func UpdateHRMSIntegration(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Integration ID is required"})
		}

		var body services.UpdateIntegrationInput
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.UpdateIntegrationByID(ctx, tenantID, id, body)
		if err != nil {
			if err.Error() == "integration not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(out)
	}
}

// ToggleHRMSIntegration toggles integration active status.
func ToggleHRMSIntegration(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")

		var body struct {
			IsActive bool `json:"is_active"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.ToggleIntegration(ctx, tenantID, id, body.IsActive)
		if err != nil {
			if err.Error() == "integration not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(out)
	}
}

// TestHRMSIntegration runs connection/config checks.
func TestHRMSIntegration(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Integration ID is required"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.TestIntegration(ctx, tenantID, id)
		if err != nil {
			if err.Error() == "integration not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(out)
	}
}

// GetHRMSSyncSchedule returns sync schedule for an integration.
func GetHRMSSyncSchedule(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Integration ID is required"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.GetSyncSchedule(ctx, tenantID, id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load schedule"})
		}
		if out == nil {
			return c.JSON(fiber.Map{"schedule": nil})
		}
		return c.JSON(out)
	}
}

// UpsertHRMSSyncSchedule creates or updates sync schedule.
func UpsertHRMSSyncSchedule(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Integration ID is required"})
		}

		var body services.UpsertHRMSSyncScheduleInput
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.UpsertSyncSchedule(ctx, tenantID, id, body)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(out)
	}
}

// DeleteHRMSSyncSchedule removes sync schedule for an integration.
func DeleteHRMSSyncSchedule(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Integration ID is required"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		if err := hrmsSvc.DeleteSyncSchedule(ctx, tenantID, id); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete schedule"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

// ListHRMSSyncLogs returns sync history for an integration.
func ListHRMSSyncLogs(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Integration ID is required"})
		}
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.ListSyncLogs(ctx, tenantID, id, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load sync logs"})
		}
		return c.JSON(out)
	}
}

// RunHRMSSync runs a manual sync (stub) and logs it.
func RunHRMSSync(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Integration ID is required"})
		}
		var body struct {
			Message string `json:"message"`
		}
		_ = c.BodyParser(&body)
		if body.Message == "" {
			body.Message = "Manual sync triggered"
		}
		ctx, cancel := context.WithTimeout(c.Context(), 7*time.Second)
		defer cancel()
		out, err := hrmsSvc.RunSync(ctx, tenantID, id, body.Message)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to run sync"})
		}
		return c.JSON(out)
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
