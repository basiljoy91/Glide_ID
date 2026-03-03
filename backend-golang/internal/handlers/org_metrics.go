package handlers

import (
	"context"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orgMetricsResponse struct {
	TotalEmployees      int `json:"totalEmployees"`
	TodayCheckIns       int `json:"todayCheckIns"`
	AnomaliesPending    int `json:"anomaliesPending"`
	ActiveKiosks        int `json:"activeKiosks"`
	HealthyKiosks       int `json:"healthyKiosks"`
	OfflineKiosks       int `json:"offlineKiosks"`
	TotalAttendanceLogs int `json:"totalAttendanceLogs"`
}

// GetOrgMetrics returns per-tenant metrics for Org Admin dashboard.
func GetOrgMetrics(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Tenant ID not found",
			})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var totalEmployees int
		var todayCheckIns int
		var anomaliesPending int
		var activeKiosks int
		var healthyKiosks int
		var totalAttendance int

		// Total active employees (not deleted)
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM users
			WHERE tenant_id = $1 AND deleted_at IS NULL AND is_active = true
		`, tenantID).Scan(&totalEmployees); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count employees",
			})
		}

		// Today's check-ins
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM attendance_logs
			WHERE tenant_id = $1
			  AND punch_time::date = CURRENT_DATE
		`, tenantID).Scan(&todayCheckIns); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count today's check-ins",
			})
		}

		// Anomalies pending review (anomaly_detected = true)
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM attendance_logs
			WHERE tenant_id = $1
			  AND anomaly_detected = true
		`, tenantID).Scan(&anomaliesPending); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count anomalies",
			})
		}

		// Active kiosks
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM kiosks
			WHERE tenant_id = $1
			  AND status = 'active'
		`, tenantID).Scan(&activeKiosks); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count active kiosks",
			})
		}

		// Healthy kiosks: active with recent heartbeat (last 10 minutes)
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM kiosks
			WHERE tenant_id = $1
			  AND status = 'active'
			  AND (last_heartbeat_at IS NOT NULL AND last_heartbeat_at >= NOW() - INTERVAL '10 minutes')
		`, tenantID).Scan(&healthyKiosks); err != nil {
			// On error, just treat healthy as 0
			healthyKiosks = 0
		}

		// Total attendance logs (for context)
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM attendance_logs
			WHERE tenant_id = $1
		`, tenantID).Scan(&totalAttendance); err != nil {
			totalAttendance = 0
		}

		offlineKiosks := activeKiosks - healthyKiosks
		if offlineKiosks < 0 {
			offlineKiosks = 0
		}

		return c.JSON(orgMetricsResponse{
			TotalEmployees:      totalEmployees,
			TodayCheckIns:       todayCheckIns,
			AnomaliesPending:    anomaliesPending,
			ActiveKiosks:        activeKiosks,
			HealthyKiosks:       healthyKiosks,
			OfflineKiosks:       offlineKiosks,
			TotalAttendanceLogs: totalAttendance,
		})
	}
}

