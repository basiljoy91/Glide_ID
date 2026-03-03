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

		if kioskIDStr, ok := kioskID.(string); ok && kioskIDStr != "" {
			ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
			defer cancel()
			tenantStr, _ := tenantID.(string)
			if tenantStr == "" {
				tenantStr = ""
			}
			_, _ = db.Exec(ctx, `
				UPDATE kiosks
				SET last_heartbeat_at = NOW(), updated_at = NOW()
				WHERE id = $1 AND ($2 = '' OR tenant_id = $2)
			`, kioskIDStr, tenantStr)
		}

		return c.JSON(fiber.Map{
			"status":     "ok",
			"kiosk_code": kioskCode,
		})
	}
}

