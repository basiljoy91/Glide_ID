package handlers

import (
	"context"
	"encoding/csv"
	"errors"
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
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Tenant ID not found",
			})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
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

		employeeArgs := []interface{}{tenantID}
		employeeWhere := "tenant_id = $1 AND deleted_at IS NULL AND is_active = true"
		if scopedDepartmentID != "" {
			employeeWhere += fmt.Sprintf(" AND department_id = $%d", len(employeeArgs)+1)
			employeeArgs = append(employeeArgs, scopedDepartmentID)
		}
		if err := db.QueryRow(ctx, fmt.Sprintf(`
			SELECT COUNT(*)
			FROM users
			WHERE %s
		`, employeeWhere), employeeArgs...).Scan(&totalEmployees); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count employees",
			})
		}

		attendanceArgs := []interface{}{tenantID}
		attendanceJoin := ""
		attendanceWhere := "al.tenant_id = $1"
		if scopedDepartmentID != "" {
			attendanceJoin = "JOIN users u ON u.id = al.user_id"
			attendanceWhere += fmt.Sprintf(" AND u.department_id = $%d", len(attendanceArgs)+1)
			attendanceArgs = append(attendanceArgs, scopedDepartmentID)
		}

		if err := db.QueryRow(ctx, fmt.Sprintf(`
			SELECT COUNT(*)
			FROM attendance_logs al
			%s
			WHERE %s
			  AND al.punch_time::date = CURRENT_DATE
		`, attendanceJoin, attendanceWhere), attendanceArgs...).Scan(&todayCheckIns); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count today's check-ins",
			})
		}

		if err := db.QueryRow(ctx, fmt.Sprintf(`
			SELECT COUNT(*)
			FROM attendance_logs al
			%s
			WHERE %s
			  AND al.anomaly_detected = true
		`, attendanceJoin, attendanceWhere), attendanceArgs...).Scan(&anomaliesPending); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count anomalies",
			})
		}

		if scopedDepartmentID == "" {
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

			if err := db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM kiosks
				WHERE tenant_id = $1
				  AND status = 'active'
				  AND (last_heartbeat_at IS NOT NULL AND last_heartbeat_at >= NOW() - INTERVAL '10 minutes')
			`, tenantID).Scan(&healthyKiosks); err != nil {
				healthyKiosks = 0
			}
		}

		if err := db.QueryRow(ctx, fmt.Sprintf(`
			SELECT COUNT(*)
			FROM attendance_logs al
			%s
			WHERE %s
		`, attendanceJoin, attendanceWhere), attendanceArgs...).Scan(&totalAttendance); err != nil {
			totalAttendance = 0
		}

		if hasRange {
			rangeStartTime := rangeStart
			rangeEndTime := rangeEnd.Add(24 * time.Hour)
			rangeArgs := append([]interface{}{}, attendanceArgs...)
			rangeArgs = append(rangeArgs, rangeStartTime, rangeEndTime)
			startArg := len(rangeArgs) - 1
			endArg := len(rangeArgs)
			if err := db.QueryRow(ctx, fmt.Sprintf(`
				SELECT COUNT(*)
				FROM attendance_logs al
				%s
				WHERE %s
				  AND al.punch_time >= $%d AND al.punch_time < $%d
			`, attendanceJoin, attendanceWhere, startArg, endArg), rangeArgs...).Scan(&rangeCheckIns); err != nil {
				rangeCheckIns = 0
			}
			if err := db.QueryRow(ctx, fmt.Sprintf(`
				SELECT COUNT(*)
				FROM attendance_logs al
				%s
				WHERE %s
				  AND al.anomaly_detected = true
				  AND al.punch_time >= $%d AND al.punch_time < $%d
			`, attendanceJoin, attendanceWhere, startArg, endArg), rangeArgs...).Scan(&rangeAnomalies); err != nil {
				rangeAnomalies = 0
			}
			if err := db.QueryRow(ctx, fmt.Sprintf(`
				SELECT COUNT(*)
				FROM attendance_logs al
				%s
				WHERE %s
				  AND al.punch_time >= $%d AND al.punch_time < $%d
			`, attendanceJoin, attendanceWhere, startArg, endArg), rangeArgs...).Scan(&rangeAttendance); err != nil {
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
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
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

		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
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

		employeeArgs := []interface{}{tenantID}
		employeeWhere := "tenant_id = $1 AND deleted_at IS NULL AND is_active = true"
		if scopedDepartmentID != "" {
			employeeWhere += fmt.Sprintf(" AND department_id = $%d", len(employeeArgs)+1)
			employeeArgs = append(employeeArgs, scopedDepartmentID)
		}
		attendanceArgs := []interface{}{tenantID}
		attendanceJoin := ""
		attendanceWhere := "al.tenant_id = $1"
		if scopedDepartmentID != "" {
			attendanceJoin = "JOIN users u ON u.id = al.user_id"
			attendanceWhere += fmt.Sprintf(" AND u.department_id = $%d", len(attendanceArgs)+1)
			attendanceArgs = append(attendanceArgs, scopedDepartmentID)
		}

		_ = db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM users WHERE %s`, employeeWhere), employeeArgs...).Scan(&totalEmployees)
		_ = db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM attendance_logs al %s WHERE %s AND al.punch_time::date = CURRENT_DATE`, attendanceJoin, attendanceWhere), attendanceArgs...).Scan(&todayCheckIns)
		_ = db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM attendance_logs al %s WHERE %s AND al.anomaly_detected = true`, attendanceJoin, attendanceWhere), attendanceArgs...).Scan(&anomaliesPending)
		if scopedDepartmentID == "" {
			_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM kiosks WHERE tenant_id = $1 AND status = 'active'`, tenantID).Scan(&activeKiosks)
			_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM kiosks WHERE tenant_id = $1 AND status = 'active' AND (last_heartbeat_at IS NOT NULL AND last_heartbeat_at >= NOW() - INTERVAL '10 minutes')`, tenantID).Scan(&healthyKiosks)
		}
		_ = db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM attendance_logs al %s WHERE %s`, attendanceJoin, attendanceWhere), attendanceArgs...).Scan(&totalAttendance)

		if hasRange || true {
			rangeStartTime := rangeStart
			rangeEndTime := rangeEnd.Add(24 * time.Hour)
			rangeArgs := append([]interface{}{}, attendanceArgs...)
			rangeArgs = append(rangeArgs, rangeStartTime, rangeEndTime)
			startArg := len(rangeArgs) - 1
			endArg := len(rangeArgs)
			_ = db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM attendance_logs al %s WHERE %s AND al.punch_time >= $%d AND al.punch_time < $%d`, attendanceJoin, attendanceWhere, startArg, endArg), rangeArgs...).Scan(&rangeCheckIns)
			_ = db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM attendance_logs al %s WHERE %s AND al.anomaly_detected = true AND al.punch_time >= $%d AND al.punch_time < $%d`, attendanceJoin, attendanceWhere, startArg, endArg), rangeArgs...).Scan(&rangeAnomalies)
			_ = db.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM attendance_logs al %s WHERE %s AND al.punch_time >= $%d AND al.punch_time < $%d`, attendanceJoin, attendanceWhere, startArg, endArg), rangeArgs...).Scan(&rangeAttendance)
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
