package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type checkinsDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// GetCheckins7d returns check-in counts per day for last 7 days (including today).
func GetCheckins7d(db *pgxpool.Pool) fiber.Handler {
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

		query := `
			SELECT d::date AS day, COALESCE(COUNT(al.id), 0) AS cnt
			FROM generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day') d
			LEFT JOIN attendance_logs al
			  ON al.tenant_id = $1
			 AND al.punch_time::date = d::date
		`
		args := []interface{}{tenantID}
		if scopedDepartmentID != "" {
			query += `
			LEFT JOIN users u
			  ON u.id = al.user_id
			`
		}
		query += `
			WHERE 1=1
		`
		if scopedDepartmentID != "" {
			query += fmt.Sprintf(" AND (al.id IS NULL OR u.department_id = $%d)", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		query += `
			GROUP BY day
			ORDER BY day ASC
		`

		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch chart data"})
		}
		defer rows.Close()

		out := make([]checkinsDay, 0, 7)
		for rows.Next() {
			var day time.Time
			var cnt int
			if err := rows.Scan(&day, &cnt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read chart data"})
			}
			out = append(out, checkinsDay{
				Date:  day.Format("2006-01-02"),
				Count: cnt,
			})
		}

		return c.JSON(out)
	}
}

type anomalyRow struct {
	ID                 uuid.UUID `json:"id"`
	PunchTime          time.Time `json:"punch_time"`
	Status             string    `json:"status"`
	VerificationMethod string    `json:"verification_method"`
	AnomalyDetected    bool      `json:"anomaly_detected"`
	AnomalyReason      *string   `json:"anomaly_reason"`
	Notes              *string   `json:"notes"`
	UserID             uuid.UUID `json:"user_id"`
	EmployeeID         string    `json:"employee_id"`
	FirstName          string    `json:"first_name"`
	LastName           string    `json:"last_name"`
	KioskCode          *string   `json:"kiosk_code"`
}

// ListAnomalies lists recent anomalies for the tenant.
func ListAnomalies(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		limit := c.QueryInt("limit", 20)
		offset := c.QueryInt("offset", 0)
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")
		searchQ := strings.TrimSpace(c.Query("q"))
		method := strings.TrimSpace(c.Query("method"))
		state := strings.ToLower(strings.TrimSpace(c.Query("state", "unresolved"))) // unresolved | resolved | all
		sort := strings.ToLower(strings.TrimSpace(c.Query("sort", "desc")))
		if limit <= 0 || limit > 200 {
			limit = 20
		}
		if offset < 0 {
			offset = 0
		}
		if state != "unresolved" && state != "resolved" && state != "all" {
			state = "unresolved"
		}
		if sort != "asc" {
			sort = "desc"
		}
		if startDate != "" {
			if _, err := time.Parse("2006-01-02", startDate); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start_date"})
			}
		}
		if endDate != "" {
			if _, err := time.Parse("2006-01-02", endDate); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end_date"})
			}
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

		args := []interface{}{tenantID}
		where := "al.tenant_id = $1"
		switch state {
		case "unresolved":
			where += " AND al.anomaly_detected = true"
		case "resolved":
			where += " AND al.anomaly_detected = false AND al.anomaly_reason IS NOT NULL"
		}
		if startDate != "" {
			where += fmt.Sprintf(" AND al.punch_time::date >= $%d::date", len(args)+1)
			args = append(args, startDate)
		}
		if endDate != "" {
			where += fmt.Sprintf(" AND al.punch_time::date <= $%d::date", len(args)+1)
			args = append(args, endDate)
		}
		if method != "" {
			where += fmt.Sprintf(" AND al.verification_method = $%d", len(args)+1)
			args = append(args, method)
		}
		if searchQ != "" {
			where += fmt.Sprintf(" AND (u.employee_id ILIKE $%d OR u.first_name ILIKE $%d OR u.last_name ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1)
			args = append(args, "%"+searchQ+"%")
		}
		if scopedDepartmentID != "" {
			where += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		args = append(args, limit, offset)
		limitArg := len(args) - 1
		offsetArg := len(args)

		rows, err := db.Query(ctx, fmt.Sprintf(`
			SELECT
				al.id, al.punch_time, al.status, al.verification_method, al.anomaly_detected, al.anomaly_reason, al.notes,
				al.user_id,
				u.employee_id, u.first_name, u.last_name,
				k.code as kiosk_code
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			LEFT JOIN kiosks k ON k.id = al.kiosk_id
			WHERE %s
			ORDER BY al.punch_time %s
			LIMIT $%d OFFSET $%d
		`, where, sort, limitArg, offsetArg), args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list anomalies"})
		}
		defer rows.Close()

		out := make([]anomalyRow, 0)
		for rows.Next() {
			var r anomalyRow
			if err := rows.Scan(&r.ID, &r.PunchTime, &r.Status, &r.VerificationMethod, &r.AnomalyDetected, &r.AnomalyReason, &r.Notes, &r.UserID, &r.EmployeeID, &r.FirstName, &r.LastName, &r.KioskCode); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read anomalies"})
			}
			out = append(out, r)
		}

		return c.JSON(out)
	}
}

// GetAnomaly returns anomaly details by id.
func GetAnomaly(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		id := c.Params("id")

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
		}

		var r anomalyRow
		args := []interface{}{tenantID, id}
		where := `
			al.tenant_id = $1
			  AND al.id = $2
		`
		if scopedDepartmentID != "" {
			where += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		err = db.QueryRow(ctx, fmt.Sprintf(`
			SELECT
				al.id, al.punch_time, al.status, al.verification_method, al.anomaly_detected, al.anomaly_reason, al.notes,
				al.user_id,
				u.employee_id, u.first_name, u.last_name,
				k.code as kiosk_code
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			LEFT JOIN kiosks k ON k.id = al.kiosk_id
			WHERE %s
		`, where), args...).Scan(
			&r.ID, &r.PunchTime, &r.Status, &r.VerificationMethod, &r.AnomalyDetected, &r.AnomalyReason, &r.Notes,
			&r.UserID, &r.EmployeeID, &r.FirstName, &r.LastName, &r.KioskCode,
		)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Anomaly not found"})
		}

		return c.JSON(r)
	}
}

// BulkResolveAnomalies resolves multiple anomalies in one operation.
func BulkResolveAnomalies(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		var body struct {
			IDs  []string `json:"ids"`
			Note string   `json:"note"`
		}
		if err := c.BodyParser(&body); err != nil || len(body.IDs) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ids array is required"})
		}
		if len(body.IDs) > 500 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bulk resolve limit is 500"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 7*time.Second)
		defer cancel()

		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
		}

		ids := make([]uuid.UUID, 0, len(body.IDs))
		for _, id := range body.IDs {
			p, err := uuid.Parse(id)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid anomaly id: " + id})
			}
			ids = append(ids, p)
		}

		query := `
			UPDATE attendance_logs
			SET
				anomaly_detected = false,
				notes = CASE
					WHEN $3 = '' THEN notes
					WHEN notes IS NULL OR notes = '' THEN '[Resolved] ' || $3
					ELSE notes || E'\n' || '[Resolved] ' || $3
				END,
				updated_at = NOW()
			FROM users u
			WHERE attendance_logs.tenant_id = $1
			  AND attendance_logs.id = ANY($2::uuid[])
			  AND u.id = attendance_logs.user_id
		`
		args := []interface{}{tenantID, ids, body.Note}
		if scopedDepartmentID != "" {
			query += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		tag, err := db.Exec(ctx, query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to bulk resolve anomalies"})
		}

		return c.JSON(fiber.Map{
			"success":  true,
			"affected": tag.RowsAffected(),
		})
	}
}

// ResolveAnomaly marks anomaly as resolved (sets anomaly_detected=false and appends a note).
func ResolveAnomaly(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		id := c.Params("id")

		var body struct {
			Note string `json:"note"`
		}
		_ = c.BodyParser(&body)

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		scopedDepartmentID, err := resolveManagedDepartmentID(ctx, db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve access scope"})
		}

		query := `
			UPDATE attendance_logs
			SET
				anomaly_detected = false,
				notes = CASE
					WHEN $3 = '' THEN notes
					WHEN notes IS NULL OR notes = '' THEN '[Resolved] ' || $3
					ELSE notes || E'\n' || '[Resolved] ' || $3
				END,
				updated_at = NOW()
			FROM users u
			WHERE attendance_logs.id = $1
			  AND attendance_logs.tenant_id = $2
			  AND u.id = attendance_logs.user_id
		`
		args := []interface{}{id, tenantID, body.Note}
		if scopedDepartmentID != "" {
			query += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		tag, err := db.Exec(ctx, query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve anomaly"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Anomaly not found"})
		}

		return c.JSON(fiber.Map{"success": true})
	}
}

// GetAttendanceReport returns daily attendance counts for a date range (default last 7 days).
func GetAttendanceReport(reporting *services.ReportingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)

		start := c.Query("start_date", time.Now().AddDate(0, 0, -6).Format("2006-01-02"))
		end := c.Query("end_date", time.Now().Format("2006-01-02"))
		departmentID := strings.TrimSpace(c.Query("department_id"))
		userID := strings.TrimSpace(c.Query("user_id"))
		employeeID := strings.TrimSpace(c.Query("employee_id"))
		includeShift := c.Query("include_shift_summary", "true")
		lateGrace := c.QueryInt("late_grace_minutes", 10)
		earlyGrace := c.QueryInt("early_grace_minutes", 10)

		startT, err1 := time.Parse("2006-01-02", start)
		endT, err2 := time.Parse("2006-01-02", end)
		if err1 != nil || err2 != nil || endT.Before(startT) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date range"})
		}
		if lateGrace < 0 || lateGrace > 180 {
			lateGrace = 10
		}
		if earlyGrace < 0 || earlyGrace > 180 {
			earlyGrace = 10
		}
		if userID != "" {
			if _, err := uuid.Parse(userID); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user_id"})
			}
		}
		if departmentID != "" {
			if _, err := uuid.Parse(departmentID); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid department_id"})
			}
		}

		ctx, cancel := context.WithTimeout(c.Context(), 7*time.Second)
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

		resp, err := reporting.BuildAttendanceReport(ctx, tenantID, start, end, departmentID, userID, employeeID, strings.ToLower(includeShift) != "false", lateGrace, earlyGrace)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(resp)
	}
}
