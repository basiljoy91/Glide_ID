package handlers

import (
	"context"
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
		var organizationName string
		var kioskName string
		var kioskStatus string
		var lastHeartbeatAt *time.Time

		if kioskIDStr, ok := kioskID.(string); ok && kioskIDStr != "" {
			ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
			defer cancel()
			_, _ = db.Exec(ctx, `
				UPDATE kiosks
				SET last_heartbeat_at = NOW(), updated_at = NOW()
				WHERE id = $1
			`, kioskIDStr)
			_ = db.QueryRow(ctx, `
				SELECT k.name, k.status, k.last_heartbeat_at, t.name
				FROM kiosks k
				JOIN tenants t ON t.id = k.tenant_id
				WHERE k.id = $1
			`, kioskIDStr).Scan(&kioskName, &kioskStatus, &lastHeartbeatAt, &organizationName)
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
		})
	}
}
