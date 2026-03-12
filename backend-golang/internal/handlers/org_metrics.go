package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orgMetricsResponse struct {
	TotalEmployees      int    `json:"totalEmployees"`
	TodayCheckIns       int    `json:"todayCheckIns"`
	AnomaliesPending    int    `json:"anomaliesPending"`
	ActiveKiosks        int    `json:"activeKiosks"`
	HealthyKiosks       int    `json:"healthyKiosks"`
	OfflineKiosks       int    `json:"offlineKiosks"`
	TotalAttendanceLogs int    `json:"totalAttendanceLogs"`
	RangeCheckIns       int    `json:"rangeCheckIns"`
	RangeAnomalies      int    `json:"rangeAnomalies"`
	RangeAttendanceLogs int    `json:"rangeAttendanceLogs"`
	RangeStart          string `json:"rangeStart"`
	RangeEnd            string `json:"rangeEnd"`
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

		startStr := strings.TrimSpace(c.Query("start", ""))
		endStr := strings.TrimSpace(c.Query("end", ""))
		var rangeStart time.Time
		var rangeEnd time.Time
		var hasRange bool
		if startStr != "" && endStr != "" {
			var err error
			rangeStart, err = time.Parse("2006-01-02", startStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start date"})
			}
			rangeEnd, err = time.Parse("2006-01-02", endStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end date"})
			}
			if rangeEnd.Before(rangeStart) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "End date must be after start date"})
			}
			hasRange = true
		}

		var totalEmployees int
		var todayCheckIns int
		var anomaliesPending int
		var activeKiosks int
		var healthyKiosks int
		var totalAttendance int
		var rangeCheckIns int
		var rangeAnomalies int
		var rangeAttendance int

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

		if hasRange {
			rangeStartTime := rangeStart
			rangeEndTime := rangeEnd.Add(24 * time.Hour)
			if err := db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM attendance_logs
				WHERE tenant_id = $1
				  AND punch_time >= $2 AND punch_time < $3
			`, tenantID, rangeStartTime, rangeEndTime).Scan(&rangeCheckIns); err != nil {
				rangeCheckIns = 0
			}
			if err := db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM attendance_logs
				WHERE tenant_id = $1
				  AND anomaly_detected = true
				  AND punch_time >= $2 AND punch_time < $3
			`, tenantID, rangeStartTime, rangeEndTime).Scan(&rangeAnomalies); err != nil {
				rangeAnomalies = 0
			}
			if err := db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM attendance_logs
				WHERE tenant_id = $1
				  AND punch_time >= $2 AND punch_time < $3
			`, tenantID, rangeStartTime, rangeEndTime).Scan(&rangeAttendance); err != nil {
				rangeAttendance = 0
			}
		} else {
			rangeCheckIns = todayCheckIns
			rangeAnomalies = anomaliesPending
			rangeAttendance = totalAttendance
			rangeStart = time.Now().AddDate(0, 0, -6)
			rangeEnd = time.Now()
		}

		offlineKiosks := activeKiosks - healthyKiosks
		if offlineKiosks < 0 {
			offlineKiosks = 0
		}

		resp := orgMetricsResponse{
			TotalEmployees:      totalEmployees,
			TodayCheckIns:       todayCheckIns,
			AnomaliesPending:    anomaliesPending,
			ActiveKiosks:        activeKiosks,
			HealthyKiosks:       healthyKiosks,
			OfflineKiosks:       offlineKiosks,
			TotalAttendanceLogs: totalAttendance,
			RangeCheckIns:       rangeCheckIns,
			RangeAnomalies:      rangeAnomalies,
			RangeAttendanceLogs: rangeAttendance,
		}
		if hasRange {
			resp.RangeStart = rangeStart.Format("2006-01-02")
			resp.RangeEnd = rangeEnd.Format("2006-01-02")
		} else {
			resp.RangeStart = rangeStart.Format("2006-01-02")
			resp.RangeEnd = rangeEnd.Format("2006-01-02")
		}
		return c.JSON(resp)
	}
}

// ExportOrgMetrics returns a CSV export for the dashboard metrics.
func ExportOrgMetrics(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Tenant ID not found",
			})
		}

		startStr := strings.TrimSpace(c.Query("start", ""))
		endStr := strings.TrimSpace(c.Query("end", ""))
		var rangeStart time.Time
		var rangeEnd time.Time
		var hasRange bool
		if startStr != "" && endStr != "" {
			var err error
			rangeStart, err = time.Parse("2006-01-02", startStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start date"})
			}
			rangeEnd, err = time.Parse("2006-01-02", endStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end date"})
			}
			if rangeEnd.Before(rangeStart) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "End date must be after start date"})
			}
			hasRange = true
		} else {
			rangeStart = time.Now().AddDate(0, 0, -6)
			rangeEnd = time.Now()
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var totalEmployees int
		var todayCheckIns int
		var anomaliesPending int
		var activeKiosks int
		var healthyKiosks int
		var totalAttendance int
		var rangeCheckIns int
		var rangeAnomalies int
		var rangeAttendance int

		_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND deleted_at IS NULL AND is_active = true`, tenantID).Scan(&totalEmployees)
		_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM attendance_logs WHERE tenant_id = $1 AND punch_time::date = CURRENT_DATE`, tenantID).Scan(&todayCheckIns)
		_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM attendance_logs WHERE tenant_id = $1 AND anomaly_detected = true`, tenantID).Scan(&anomaliesPending)
		_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM kiosks WHERE tenant_id = $1 AND status = 'active'`, tenantID).Scan(&activeKiosks)
		_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM kiosks WHERE tenant_id = $1 AND status = 'active' AND (last_heartbeat_at IS NOT NULL AND last_heartbeat_at >= NOW() - INTERVAL '10 minutes')`, tenantID).Scan(&healthyKiosks)
		_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM attendance_logs WHERE tenant_id = $1`, tenantID).Scan(&totalAttendance)

		if hasRange || true {
			rangeStartTime := rangeStart
			rangeEndTime := rangeEnd.Add(24 * time.Hour)
			_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM attendance_logs WHERE tenant_id = $1 AND punch_time >= $2 AND punch_time < $3`, tenantID, rangeStartTime, rangeEndTime).Scan(&rangeCheckIns)
			_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM attendance_logs WHERE tenant_id = $1 AND anomaly_detected = true AND punch_time >= $2 AND punch_time < $3`, tenantID, rangeStartTime, rangeEndTime).Scan(&rangeAnomalies)
			_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM attendance_logs WHERE tenant_id = $1 AND punch_time >= $2 AND punch_time < $3`, tenantID, rangeStartTime, rangeEndTime).Scan(&rangeAttendance)
		}

		offlineKiosks := activeKiosks - healthyKiosks
		if offlineKiosks < 0 {
			offlineKiosks = 0
		}

		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", "attachment; filename=org-metrics.csv")
		writer := csv.NewWriter(c.Response().BodyWriter())
		_ = writer.Write([]string{"metric", "value"})
		_ = writer.Write([]string{"total_employees", fmt.Sprintf("%d", totalEmployees)})
		_ = writer.Write([]string{"today_check_ins", fmt.Sprintf("%d", todayCheckIns)})
		_ = writer.Write([]string{"anomalies_pending", fmt.Sprintf("%d", anomaliesPending)})
		_ = writer.Write([]string{"active_kiosks", fmt.Sprintf("%d", activeKiosks)})
		_ = writer.Write([]string{"healthy_kiosks", fmt.Sprintf("%d", healthyKiosks)})
		_ = writer.Write([]string{"offline_kiosks", fmt.Sprintf("%d", offlineKiosks)})
		_ = writer.Write([]string{"total_attendance_logs", fmt.Sprintf("%d", totalAttendance)})
		_ = writer.Write([]string{"range_start", rangeStart.Format("2006-01-02")})
		_ = writer.Write([]string{"range_end", rangeEnd.Format("2006-01-02")})
		_ = writer.Write([]string{"range_check_ins", fmt.Sprintf("%d", rangeCheckIns)})
		_ = writer.Write([]string{"range_anomalies", fmt.Sprintf("%d", rangeAnomalies)})
		_ = writer.Write([]string{"range_attendance_logs", fmt.Sprintf("%d", rangeAttendance)})
		writer.Flush()
		return nil
	}
}
