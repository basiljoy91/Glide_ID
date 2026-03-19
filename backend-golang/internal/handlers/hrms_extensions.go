package handlers

import (
	"context"
	"errors"
	"time"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

func ListHRMSWebhookEvents(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		id := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.ListWebhookEvents(ctx, tenantID, id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load webhook queue"})
		}
		return c.JSON(out)
	}
}

func RetryHRMSWebhookEvent(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		id := c.Params("id")
		eventID := c.Params("eventId")
		ctx, cancel := context.WithTimeout(c.Context(), 7*time.Second)
		defer cancel()
		if err := hrmsSvc.RetryWebhookEvent(ctx, tenantID, id, eventID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Webhook event not found"})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func TestHRMSFieldMapping(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		id := c.Params("id")
		var body struct {
			Sample       map[string]interface{} `json:"sample"`
			FieldMapping []map[string]string    `json:"field_mapping"`
		}
		if err := c.BodyParser(&body); err != nil || body.Sample == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "sample payload is required"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.TestFieldMapping(ctx, tenantID, id, body.Sample, body.FieldMapping)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(out)
	}
}

func DryRunHRMSDirectorySync(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		id := c.Params("id")
		var body struct {
			Records      []map[string]interface{} `json:"records"`
			FieldMapping []map[string]string      `json:"field_mapping"`
		}
		if err := c.BodyParser(&body); err != nil || len(body.Records) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "records array is required"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
		defer cancel()
		out, err := hrmsSvc.DryRunDirectorySync(ctx, tenantID, id, body.Records, body.FieldMapping)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(out)
	}
}

func ListHRMSSyncConflicts(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		id := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		out, err := hrmsSvc.ListSyncConflicts(ctx, tenantID, id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load conflicts"})
		}
		return c.JSON(out)
	}
}

func ResolveHRMSSyncConflict(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		id := c.Params("id")
		conflictID := c.Params("conflictId")
		var body struct {
			Resolution string `json:"resolution"`
		}
		if err := c.BodyParser(&body); err != nil || body.Resolution == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "resolution is required"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		if err := hrmsSvc.ResolveSyncConflict(ctx, tenantID, id, conflictID, body.Resolution, actorUserID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Conflict not found"})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func RotateHRMSCredentials(hrmsSvc *services.HRMSService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		id := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		creds, err := hrmsSvc.RotateCredentials(ctx, tenantID, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Integration not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to rotate credentials"})
		}
		return c.JSON(creds)
	}
}
