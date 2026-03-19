package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"enterprise-attendance-api/internal/config"
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type reportViewRow struct {
	ID         uuid.UUID              `json:"id"`
	ReportType string                 `json:"report_type"`
	Name       string                 `json:"name"`
	Filters    map[string]interface{} `json:"filters"`
	IsDefault  bool                   `json:"is_default"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

type reportViewBody struct {
	ReportType string                 `json:"report_type"`
	Name       string                 `json:"name"`
	Filters    map[string]interface{} `json:"filters"`
	IsDefault  bool                   `json:"is_default"`
}

type comparisonMetric struct {
	Current      int     `json:"current"`
	Previous     int     `json:"previous"`
	Delta        int     `json:"delta"`
	DeltaPercent float64 `json:"delta_percent"`
}

type leaderboardEntry struct {
	Label     string `json:"label"`
	CheckIns  int    `json:"check_ins"`
	Anomalies int    `json:"anomalies"`
	Late      int    `json:"late_arrivals"`
}

func buildComparisonMetric(current, previous int) comparisonMetric {
	metric := comparisonMetric{
		Current:  current,
		Previous: previous,
		Delta:    current - previous,
	}
	if previous == 0 {
		if current > 0 {
			metric.DeltaPercent = 100
		}
		return metric
	}
	metric.DeltaPercent = float64(metric.Delta) * 100 / float64(previous)
	return metric
}

func buildAttendanceComparison(ctx context.Context, reporting *services.ReportingService, tenantID, start, end, departmentID, userID, employeeID string, lateGrace, earlyGrace int) (fiber.Map, error) {
	startDate, _ := time.Parse("2006-01-02", start)
	endDate, _ := time.Parse("2006-01-02", end)
	daySpan := int(endDate.Sub(startDate).Hours()/24) + 1
	prevEnd := startDate.AddDate(0, 0, -1)
	prevStart := prevEnd.AddDate(0, 0, -(daySpan - 1))

	current, err := reporting.BuildAttendanceReport(ctx, tenantID, start, end, departmentID, userID, employeeID, false, lateGrace, earlyGrace)
	if err != nil {
		return nil, err
	}
	previous, err := reporting.BuildAttendanceReport(ctx, tenantID, prevStart.Format("2006-01-02"), prevEnd.Format("2006-01-02"), departmentID, userID, employeeID, false, lateGrace, earlyGrace)
	if err != nil {
		return nil, err
	}

	return fiber.Map{
		"previous_start_date": prevStart.Format("2006-01-02"),
		"previous_end_date":   prevEnd.Format("2006-01-02"),
		"check_ins":           buildComparisonMetric(current.Totals.CheckIns, previous.Totals.CheckIns),
		"check_outs":          buildComparisonMetric(current.Totals.CheckOuts, previous.Totals.CheckOuts),
		"anomalies":           buildComparisonMetric(current.Totals.Anomalies, previous.Totals.Anomalies),
		"late_arrivals":       buildComparisonMetric(current.Totals.LateArrivals, previous.Totals.LateArrivals),
		"early_departures":    buildComparisonMetric(current.Totals.EarlyDepartures, previous.Totals.EarlyDepartures),
	}, nil
}

func buildAttendanceLeaderboards(ctx context.Context, db *pgxpool.Pool, tenantID, start, end, departmentID, scopedDepartmentID string, lateGrace int) (fiber.Map, error) {
	baseArgs := []interface{}{tenantID, start, end, lateGrace}
	where := `
		al.tenant_id = $1
		AND al.punch_time::date >= $2::date
		AND al.punch_time::date <= $3::date
	`
	if departmentID != "" {
		where += fmt.Sprintf(" AND u.department_id = $%d", len(baseArgs)+1)
		baseArgs = append(baseArgs, departmentID)
	}
	if scopedDepartmentID != "" {
		where += fmt.Sprintf(" AND u.department_id = $%d", len(baseArgs)+1)
		baseArgs = append(baseArgs, scopedDepartmentID)
	}

	departments, err := loadLeaderboardRows(ctx, db, fmt.Sprintf(`
		WITH filtered AS (
			SELECT al.status, al.anomaly_detected, al.punch_time, u.shift_start_time, d.name AS label
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			LEFT JOIN departments d ON d.id = u.department_id
			WHERE %s
		)
		SELECT
			COALESCE(label, 'Unassigned') AS label,
			COUNT(*) FILTER (WHERE status = 'check_in') AS check_ins,
			COUNT(*) FILTER (WHERE anomaly_detected = true) AS anomalies,
			COUNT(*) FILTER (
				WHERE status = 'check_in'
				  AND shift_start_time IS NOT NULL
				  AND punch_time::time > (shift_start_time::time + ($4::int * INTERVAL '1 minute'))
			) AS late_arrivals
		FROM filtered
		GROUP BY label
		ORDER BY check_ins DESC, anomalies ASC, label ASC
		LIMIT 5
	`, where), baseArgs...)
	if err != nil {
		return nil, err
	}

	managers, err := loadLeaderboardRows(ctx, db, fmt.Sprintf(`
		WITH filtered AS (
			SELECT al.status, al.anomaly_detected, al.punch_time, u.shift_start_time,
				TRIM(COALESCE(m.first_name, '') || ' ' || COALESCE(m.last_name, '')) AS label
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			LEFT JOIN users m ON m.id = u.manager_id
			WHERE %s
		)
		SELECT
			CASE WHEN label = '' THEN 'Unassigned manager' ELSE label END AS label,
			COUNT(*) FILTER (WHERE status = 'check_in') AS check_ins,
			COUNT(*) FILTER (WHERE anomaly_detected = true) AS anomalies,
			COUNT(*) FILTER (
				WHERE status = 'check_in'
				  AND shift_start_time IS NOT NULL
				  AND punch_time::time > (shift_start_time::time + ($4::int * INTERVAL '1 minute'))
			) AS late_arrivals
		FROM filtered
		GROUP BY label
		ORDER BY check_ins DESC, anomalies ASC, label ASC
		LIMIT 5
	`, where), baseArgs...)
	if err != nil {
		return nil, err
	}

	return fiber.Map{
		"departments": departments,
		"managers":    managers,
	}, nil
}

func loadLeaderboardRows(ctx context.Context, db *pgxpool.Pool, query string, args ...interface{}) ([]leaderboardEntry, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []leaderboardEntry{}
	for rows.Next() {
		var row leaderboardEntry
		if err := rows.Scan(&row.Label, &row.CheckIns, &row.Anomalies, &row.Late); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func signReportPayload(payload []byte) string {
	cfg := config.Load()
	secret := cfg.JWTSecret
	if secret == "" {
		secret = "enterprise-attendance-api"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func ListReportViews(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
		}

		rows, err := db.Query(ctx, `
			SELECT id, report_type, name, filters, is_default, created_at, updated_at
			FROM report_saved_views
			WHERE tenant_id = $1
			ORDER BY is_default DESC, created_at DESC
		`, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list report views"})
		}
		defer rows.Close()

		out := []reportViewRow{}
		for rows.Next() {
			var view reportViewRow
			var created time.Time
			var updated time.Time
			if err := rows.Scan(&view.ID, &view.ReportType, &view.Name, &view.Filters, &view.IsDefault, &created, &updated); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read report views"})
			}
			if !scheduleVisibleToDepartment(view.Filters, scopedDepartmentID) {
				continue
			}
			view.CreatedAt = created.UTC().Format(time.RFC3339)
			view.UpdatedAt = updated.UTC().Format(time.RFC3339)
			out = append(out, view)
		}
		return c.JSON(out)
	}
}

func CreateReportView(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		var body reportViewBody
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.ReportType) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and report_type are required"})
		}
		if body.Filters == nil {
			body.Filters = map[string]interface{}{}
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
		if requestedDepartmentID := scheduleDepartmentID(body.Filters); requestedDepartmentID != "" && scopedDepartmentID != "" && requestedDepartmentID != scopedDepartmentID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Requested department is outside your scope"})
		}
		body.Filters = scopeScheduleFilters(body.Filters, scopedDepartmentID)

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start report view transaction"})
		}
		defer tx.Rollback(ctx)
		if body.IsDefault {
			if _, err := tx.Exec(ctx, `UPDATE report_saved_views SET is_default = false, updated_at = NOW() WHERE tenant_id = $1 AND report_type = $2`, tenantID, body.ReportType); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update existing defaults"})
			}
		}

		var view reportViewRow
		var created time.Time
		var updated time.Time
		if err := tx.QueryRow(ctx, `
			INSERT INTO report_saved_views (tenant_id, report_type, name, filters, is_default, created_by, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			RETURNING id, report_type, name, filters, is_default, created_at, updated_at
		`, tenantID, body.ReportType, strings.TrimSpace(body.Name), body.Filters, body.IsDefault, actorUserID).Scan(
			&view.ID, &view.ReportType, &view.Name, &view.Filters, &view.IsDefault, &created, &updated,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create report view"})
		}
		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize report view"})
		}
		view.CreatedAt = created.UTC().Format(time.RFC3339)
		view.UpdatedAt = updated.UTC().Format(time.RFC3339)
		return c.Status(fiber.StatusCreated).JSON(view)
	}
}

func UpdateReportView(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		viewID := c.Params("id")
		var body reportViewBody
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
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

		var current reportViewRow
		var created time.Time
		var updated time.Time
		err = db.QueryRow(ctx, `
			SELECT id, report_type, name, filters, is_default, created_at, updated_at
			FROM report_saved_views
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, viewID).Scan(&current.ID, &current.ReportType, &current.Name, &current.Filters, &current.IsDefault, &created, &updated)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report view not found"})
		}
		if !scheduleVisibleToDepartment(current.Filters, scopedDepartmentID) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report view not found"})
		}

		if strings.TrimSpace(body.Name) == "" {
			body.Name = current.Name
		}
		if strings.TrimSpace(body.ReportType) == "" {
			body.ReportType = current.ReportType
		}
		if body.Filters == nil {
			body.Filters = current.Filters
		}
		if requestedDepartmentID := scheduleDepartmentID(body.Filters); requestedDepartmentID != "" && scopedDepartmentID != "" && requestedDepartmentID != scopedDepartmentID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Requested department is outside your scope"})
		}
		body.Filters = scopeScheduleFilters(body.Filters, scopedDepartmentID)

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start report view update"})
		}
		defer tx.Rollback(ctx)
		if body.IsDefault {
			if _, err := tx.Exec(ctx, `UPDATE report_saved_views SET is_default = false, updated_at = NOW() WHERE tenant_id = $1 AND report_type = $2 AND id <> $3`, tenantID, body.ReportType, viewID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update existing defaults"})
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE report_saved_views
			SET report_type = $1, name = $2, filters = $3, is_default = $4, updated_at = NOW()
			WHERE tenant_id = $5 AND id = $6
		`, body.ReportType, strings.TrimSpace(body.Name), body.Filters, body.IsDefault, tenantID, viewID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update report view"})
		}
		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize report view update"})
		}
		return ListReportViews(db)(c)
	}
}

func DeleteReportView(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		viewID := c.Params("id")

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
		}

		var filters map[string]interface{}
		if err := db.QueryRow(ctx, `SELECT filters FROM report_saved_views WHERE tenant_id = $1 AND id = $2`, tenantID, viewID).Scan(&filters); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report view not found"})
		}
		if !scheduleVisibleToDepartment(filters, scopedDepartmentID) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Report view not found"})
		}
		if _, err := db.Exec(ctx, `DELETE FROM report_saved_views WHERE tenant_id = $1 AND id = $2`, tenantID, viewID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete report view"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func ExportPayrollReport(reporting *services.ReportingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		start := c.Query("start_date", time.Now().AddDate(0, 0, -13).Format("2006-01-02"))
		end := c.Query("end_date", time.Now().Format("2006-01-02"))
		departmentID := strings.TrimSpace(c.Query("department_id"))

		ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
		defer cancel()
		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, reporting.GetDB(), tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
		}
		departmentID, err = enforceDepartmentScope(departmentID, scopedDepartmentID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Requested department is outside your scope"})
		}

		args := []interface{}{tenantID, start, end}
		where := `
			al.tenant_id = $1
			AND al.punch_time::date >= $2::date
			AND al.punch_time::date <= $3::date
		`
		if departmentID != "" {
			where += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, departmentID)
		}

		rows, err := reporting.GetDB().Query(ctx, fmt.Sprintf(`
			WITH daily AS (
				SELECT
					u.id AS user_id,
					u.employee_id,
					u.first_name,
					u.last_name,
					al.punch_time::date AS work_date,
					MIN(al.punch_time) FILTER (WHERE al.status = 'check_in') AS first_check_in,
					MAX(al.punch_time) FILTER (WHERE al.status = 'check_out') AS last_check_out,
					COUNT(*) FILTER (WHERE al.anomaly_detected = true) AS anomalies
				FROM attendance_logs al
				JOIN users u ON u.id = al.user_id
				WHERE %s
				GROUP BY u.id, u.employee_id, u.first_name, u.last_name, al.punch_time::date
			)
			SELECT
				d.employee_id,
				d.first_name,
				d.last_name,
				d.work_date,
				d.first_check_in,
				d.last_check_out,
				d.anomalies,
				COALESCE((
					SELECT approved_minutes
					FROM overtime_requests ot
					WHERE ot.tenant_id = $1
					  AND ot.user_id = d.user_id
					  AND ot.work_date = d.work_date
					  AND ot.status = 'approved'
					ORDER BY ot.reviewed_at DESC NULLS LAST
					LIMIT 1
				), 0) AS approved_overtime_minutes,
				COALESCE((
					SELECT leave_type
					FROM leave_requests lr
					WHERE lr.tenant_id = $1
					  AND lr.user_id = d.user_id
					  AND lr.status = 'approved'
					  AND d.work_date BETWEEN lr.start_date AND lr.end_date
					ORDER BY lr.reviewed_at DESC NULLS LAST
					LIMIT 1
				), '') AS approved_leave_type
			FROM daily d
			ORDER BY d.work_date DESC, d.employee_id ASC
		`, where), args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to build payroll export"})
		}
		defer rows.Close()

		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		_ = writer.Write([]string{"employee_id", "employee_name", "work_date", "first_check_in", "last_check_out", "anomalies", "approved_overtime_minutes", "approved_leave_type"})
		rowCount := 0
		for rows.Next() {
			var employeeID, firstName, lastName, leaveType string
			var workDate time.Time
			var firstIn *time.Time
			var lastOut *time.Time
			var anomalies, overtimeMinutes int
			if err := rows.Scan(&employeeID, &firstName, &lastName, &workDate, &firstIn, &lastOut, &anomalies, &overtimeMinutes, &leaveType); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read payroll export"})
			}
			rowCount++
			firstInValue := ""
			if firstIn != nil {
				firstInValue = firstIn.UTC().Format(time.RFC3339)
			}
			lastOutValue := ""
			if lastOut != nil {
				lastOutValue = lastOut.UTC().Format(time.RFC3339)
			}
			_ = writer.Write([]string{
				employeeID,
				strings.TrimSpace(firstName + " " + lastName),
				workDate.Format("2006-01-02"),
				firstInValue,
				lastOutValue,
				strconv.Itoa(anomalies),
				strconv.Itoa(overtimeMinutes),
				leaveType,
			})
		}
		writer.Flush()
		if err := rows.Err(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize payroll export"})
		}

		if requestedBy, err := uuid.Parse(actorUserID); err == nil {
			payload := map[string]interface{}{
				"row_count":      rowCount,
				"department_id":  departmentID,
				"generated_from": "attendance_report",
			}
			_, _ = reporting.GetDB().Exec(ctx, `
				INSERT INTO payroll_exports (tenant_id, export_type, date_range_start, date_range_end, requested_by, status, payload, completed_at)
				VALUES ($1, 'attendance_payroll', $2::date, $3::date, $4, 'completed', $5, NOW())
			`, tenantID, start, end, requestedBy, payload)
		}

		filename := fmt.Sprintf("payroll-ready-%s-to-%s.csv", start, end)
		c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		return c.Send(buf.Bytes())
	}
}

func ExportComplianceReport(reporting *services.ReportingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		start := c.Query("start_date", time.Now().AddDate(0, 0, -29).Format("2006-01-02"))
		end := c.Query("end_date", time.Now().Format("2006-01-02"))
		departmentID := strings.TrimSpace(c.Query("department_id"))
		lateGrace := c.QueryInt("late_grace_minutes", 10)
		earlyGrace := c.QueryInt("early_grace_minutes", 10)

		ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
		defer cancel()
		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, reporting.GetDB(), tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
		}
		departmentID, err = enforceDepartmentScope(departmentID, scopedDepartmentID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Requested department is outside your scope"})
		}

		report, err := reporting.BuildAttendanceReport(ctx, tenantID, start, end, departmentID, "", "", true, lateGrace, earlyGrace)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		var unresolvedAnomalies int
		if err := reporting.GetDB().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			WHERE al.tenant_id = $1
			  AND al.punch_time::date >= $2::date
			  AND al.punch_time::date <= $3::date
			  AND al.anomaly_detected = true
			  AND ($4 = '' OR u.department_id = $4::uuid)
		`, tenantID, start, end, departmentID).Scan(&unresolvedAnomalies); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to build compliance summary"})
		}

		payload := fiber.Map{
			"tenant_id":    tenantID,
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"generated_by": actorUserID,
			"report_type":  "attendance_compliance",
			"date_range": fiber.Map{
				"start_date": start,
				"end_date":   end,
			},
			"filters": fiber.Map{
				"department_id": departmentID,
			},
			"summary": fiber.Map{
				"attendance":           report,
				"unresolved_anomalies": unresolvedAnomalies,
			},
		}
		raw, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to serialize compliance report"})
		}
		signature := signReportPayload(raw)
		filename := fmt.Sprintf("compliance-report-%s-to-%s.json", start, end)
		c.Set(fiber.HeaderContentType, "application/json; charset=utf-8")
		c.Set("X-Report-Signature", signature)
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
		return c.Send(raw)
	}
}
