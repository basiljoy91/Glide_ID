package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// KioskHeartbeat handles kiosk heartbeat
func KioskHeartbeat(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		kioskCode := c.Locals("kiosk_code")
		kioskID := c.Locals("kiosk_id")
		tenantID := c.Locals("tenant_id")
		var telemetry kioskTelemetryPayload
		if len(c.Body()) > 0 {
			_ = json.Unmarshal(c.Body(), &telemetry)
		}
		var organizationName string
		var kioskName string
		var kioskStatus string
		var lastHeartbeatAt *time.Time
		commands := []kioskCommandRow{}

		if kioskIDStr, ok := kioskID.(string); ok && kioskIDStr != "" {
			ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
			defer cancel()
			_ = db.QueryRow(ctx, `
				UPDATE kiosks
				SET last_heartbeat_at = NOW(), updated_at = NOW()
				WHERE id = $1
				RETURNING name, status, last_heartbeat_at
			`, kioskIDStr).Scan(&kioskName, &kioskStatus, &lastHeartbeatAt)
			_ = db.QueryRow(ctx, `
				SELECT t.name
				FROM kiosks k
				JOIN tenants t ON t.id = k.tenant_id
				WHERE k.id = $1
			`, kioskIDStr).Scan(&organizationName)
			if tenantIDStr, ok := tenantID.(string); ok {
				_ = persistKioskTelemetry(ctx, db, tenantIDStr, kioskIDStr, kioskStatus, telemetry)
				if delivered, err := deliverQueuedKioskCommands(ctx, db, tenantIDStr, kioskIDStr); err == nil {
					commands = delivered
				}
			}
			if kioskName == "" {
				_ = db.QueryRow(ctx, `
				SELECT k.name, k.status, k.last_heartbeat_at, t.name
				FROM kiosks k
				JOIN tenants t ON t.id = k.tenant_id
				WHERE k.id = $1
				`, kioskIDStr).Scan(&kioskName, &kioskStatus, &lastHeartbeatAt, &organizationName)
			}
		}

		tenantIDStr, _ := tenantID.(string)

		return c.JSON(fiber.Map{
			"status":            "ok",
			"connected":         true,
			"kiosk_code":        kioskCode,
			"kiosk_name":        kioskName,
			"kiosk_status":      kioskStatus,
			"organization_name": organizationName,
			"tenant_id":         tenantIDStr,
			"last_heartbeat_at": lastHeartbeatAt,
			"commands":          commands,
		})
	}
}
