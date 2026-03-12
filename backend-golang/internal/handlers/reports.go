package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/middleware"

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
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT d::date AS day, COALESCE(COUNT(al.id), 0) AS cnt
			FROM generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day') d
			LEFT JOIN attendance_logs al
			  ON al.tenant_id = $1
			 AND al.punch_time::date = d::date
			GROUP BY day
			ORDER BY day ASC
		`, tenantID)
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
		id := c.Params("id")

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var r anomalyRow
		err := db.QueryRow(ctx, `
			SELECT
				al.id, al.punch_time, al.status, al.verification_method, al.anomaly_detected, al.anomaly_reason, al.notes,
				al.user_id,
				u.employee_id, u.first_name, u.last_name,
				k.code as kiosk_code
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			LEFT JOIN kiosks k ON k.id = al.kiosk_id
			WHERE al.tenant_id = $1
			  AND al.id = $2
		`, tenantID, id).Scan(
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

		ids := make([]uuid.UUID, 0, len(body.IDs))
		for _, id := range body.IDs {
			p, err := uuid.Parse(id)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid anomaly id: " + id})
			}
			ids = append(ids, p)
		}

		tag, err := db.Exec(ctx, `
			UPDATE attendance_logs
			SET
				anomaly_detected = false,
				notes = CASE
					WHEN $3 = '' THEN notes
					WHEN notes IS NULL OR notes = '' THEN '[Resolved] ' || $3
					ELSE notes || E'\n' || '[Resolved] ' || $3
				END,
				updated_at = NOW()
			WHERE tenant_id = $1
			  AND id = ANY($2::uuid[])
		`, tenantID, ids, body.Note)
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
		id := c.Params("id")

		var body struct {
			Note string `json:"note"`
		}
		_ = c.BodyParser(&body)

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		_, err := db.Exec(ctx, `
			UPDATE attendance_logs
			SET
				anomaly_detected = false,
				notes = CASE
					WHEN $3 = '' THEN notes
					WHEN notes IS NULL OR notes = '' THEN '[Resolved] ' || $3
					ELSE notes || E'\n' || '[Resolved] ' || $3
				END,
				updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2
		`, id, tenantID, body.Note)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve anomaly"})
		}

		return c.JSON(fiber.Map{"success": true})
	}
}

type attendanceReportDay struct {
	Date            string `json:"date"`
	CheckIns        int    `json:"check_ins"`
	CheckOuts       int    `json:"check_outs"`
	Anomalies       int    `json:"anomalies"`
	LateArrivals    int    `json:"late_arrivals"`
	EarlyDepartures int    `json:"early_departures"`
}

type shiftSummaryRow struct {
	ShiftStart      *string `json:"shift_start_time,omitempty"`
	ShiftEnd        *string `json:"shift_end_time,omitempty"`
	Users           int     `json:"users"`
	CheckIns        int     `json:"check_ins"`
	CheckOuts       int     `json:"check_outs"`
	LateArrivals    int     `json:"late_arrivals"`
	EarlyDepartures int     `json:"early_departures"`
}

type attendanceReportResponse struct {
	StartDate string                `json:"start_date"`
	EndDate   string                `json:"end_date"`
	Filters   map[string]string     `json:"filters,omitempty"`
	Days      []attendanceReportDay `json:"days"`
	Totals    struct {
		CheckIns        int `json:"check_ins"`
		CheckOuts       int `json:"check_outs"`
		Anomalies       int `json:"anomalies"`
		Logs            int `json:"logs"`
		LateArrivals    int `json:"late_arrivals"`
		EarlyDepartures int `json:"early_departures"`
	} `json:"totals"`
	ShiftSummary []shiftSummaryRow `json:"shift_summary,omitempty"`
}

// GetAttendanceReport returns daily attendance counts for a date range (default last 7 days).
func GetAttendanceReport(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

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

		resp, err := buildAttendanceReport(ctx, db, tenantID, start, end, departmentID, userID, employeeID, strings.ToLower(includeShift) != "false", lateGrace, earlyGrace)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(resp)
	}
}

func buildAttendanceReport(ctx context.Context, db *pgxpool.Pool, tenantID, start, end, departmentID, userID, employeeID string, includeShift bool, lateGrace, earlyGrace int) (attendanceReportResponse, error) {
	args := []interface{}{tenantID, start, end}
	where := `
		al.tenant_id = $1
		AND al.punch_time::date >= $2::date
		AND al.punch_time::date <= $3::date
	`
	if userID != "" {
		where += fmt.Sprintf(" AND al.user_id = $%d", len(args)+1)
		args = append(args, userID)
	}
	if departmentID != "" {
		where += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
		args = append(args, departmentID)
	}
	if employeeID != "" {
		where += fmt.Sprintf(" AND u.employee_id = $%d", len(args)+1)
		args = append(args, employeeID)
	}

	args = append(args, lateGrace, earlyGrace)
	lateArg := len(args) - 1
	earlyArg := len(args)

	rows, err := db.Query(ctx, fmt.Sprintf(`
		WITH filtered_logs AS (
			SELECT al.id, al.user_id, al.status, al.punch_time, al.anomaly_detected,
				u.shift_start_time, u.shift_end_time
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			WHERE %s
		),
		daily_counts AS (
			SELECT punch_time::date AS day,
				COUNT(*) FILTER (WHERE status = 'check_in') AS check_ins,
				COUNT(*) FILTER (WHERE status = 'check_out') AS check_outs,
				COUNT(*) FILTER (WHERE anomaly_detected = true) AS anomalies,
				COUNT(*) AS logs
			FROM filtered_logs
			GROUP BY day
		),
		user_day AS (
			SELECT user_id, punch_time::date AS day,
				MIN(punch_time) FILTER (WHERE status = 'check_in') AS first_in,
				MAX(punch_time) FILTER (WHERE status = 'check_out') AS last_out,
				MAX(shift_start_time) AS shift_start_time,
				MAX(shift_end_time) AS shift_end_time
			FROM filtered_logs
			GROUP BY user_id, day
		),
		late_early AS (
			SELECT day,
				COUNT(*) FILTER (
					WHERE first_in IS NOT NULL
					  AND shift_start_time IS NOT NULL
					  AND first_in::time > (shift_start_time::time + ($%d || ' minutes')::interval)
				) AS late_arrivals,
				COUNT(*) FILTER (
					WHERE last_out IS NOT NULL
					  AND shift_end_time IS NOT NULL
					  AND last_out::time < (shift_end_time::time - ($%d || ' minutes')::interval)
				) AS early_departures
			FROM user_day
			GROUP BY day
		)
		SELECT d::date AS day,
			COALESCE(dc.check_ins, 0) AS check_ins,
			COALESCE(dc.check_outs, 0) AS check_outs,
			COALESCE(dc.anomalies, 0) AS anomalies,
			COALESCE(dc.logs, 0) AS logs,
			COALESCE(le.late_arrivals, 0) AS late_arrivals,
			COALESCE(le.early_departures, 0) AS early_departures
		FROM generate_series($2::date, $3::date, INTERVAL '1 day') d
		LEFT JOIN daily_counts dc ON dc.day = d::date
		LEFT JOIN late_early le ON le.day = d::date
		ORDER BY day ASC
	`, where, lateArg, earlyArg), args...)
	if err != nil {
		return attendanceReportResponse{}, fmt.Errorf("Failed to generate report")
	}
	defer rows.Close()

	resp := attendanceReportResponse{
		StartDate: start,
		EndDate:   end,
		Days:      []attendanceReportDay{},
		Filters: map[string]string{
			"department_id": departmentID,
			"user_id":       userID,
			"employee_id":   employeeID,
		},
	}

	for rows.Next() {
		var day time.Time
		var ci, co, an, logs, late, early int
		if err := rows.Scan(&day, &ci, &co, &an, &logs, &late, &early); err != nil {
			return attendanceReportResponse{}, fmt.Errorf("Failed to read report")
		}
		resp.Days = append(resp.Days, attendanceReportDay{
			Date:            day.Format("2006-01-02"),
			CheckIns:        ci,
			CheckOuts:       co,
			Anomalies:       an,
			LateArrivals:    late,
			EarlyDepartures: early,
		})
		resp.Totals.CheckIns += ci
		resp.Totals.CheckOuts += co
		resp.Totals.Anomalies += an
		resp.Totals.Logs += logs
		resp.Totals.LateArrivals += late
		resp.Totals.EarlyDepartures += early
	}

	if includeShift {
		shiftRows, err := db.Query(ctx, fmt.Sprintf(`
			WITH filtered_logs AS (
				SELECT al.user_id, al.status, al.punch_time,
					u.shift_start_time, u.shift_end_time
				FROM attendance_logs al
				JOIN users u ON u.id = al.user_id
				WHERE %s
			),
			user_day AS (
				SELECT user_id, punch_time::date AS day,
					MIN(punch_time) FILTER (WHERE status = 'check_in') AS first_in,
					MAX(punch_time) FILTER (WHERE status = 'check_out') AS last_out,
					MAX(shift_start_time) AS shift_start_time,
					MAX(shift_end_time) AS shift_end_time
				FROM filtered_logs
				GROUP BY user_id, day
			)
			SELECT
				shift_start_time, shift_end_time,
				COUNT(DISTINCT user_id) AS users,
				COUNT(*) FILTER (WHERE first_in IS NOT NULL) AS check_ins,
				COUNT(*) FILTER (WHERE last_out IS NOT NULL) AS check_outs,
				COUNT(*) FILTER (
					WHERE first_in IS NOT NULL
					  AND shift_start_time IS NOT NULL
					  AND first_in::time > (shift_start_time::time + ($%d || ' minutes')::interval)
				) AS late_arrivals,
				COUNT(*) FILTER (
					WHERE last_out IS NOT NULL
					  AND shift_end_time IS NOT NULL
					  AND last_out::time < (shift_end_time::time - ($%d || ' minutes')::interval)
				) AS early_departures
			FROM user_day
			GROUP BY shift_start_time, shift_end_time
			ORDER BY shift_start_time, shift_end_time
		`, where, lateArg, earlyArg), args...)
		if err == nil {
			defer shiftRows.Close()
			for shiftRows.Next() {
				var startTime *string
				var endTime *string
				var users, ci, co, late, early int
				if err := shiftRows.Scan(&startTime, &endTime, &users, &ci, &co, &late, &early); err != nil {
					return attendanceReportResponse{}, fmt.Errorf("Failed to read shift summary")
				}
				resp.ShiftSummary = append(resp.ShiftSummary, shiftSummaryRow{
					ShiftStart:      startTime,
					ShiftEnd:        endTime,
					Users:           users,
					CheckIns:        ci,
					CheckOuts:       co,
					LateArrivals:    late,
					EarlyDepartures: early,
				})
			}
		}
	}

	return resp, nil
}
