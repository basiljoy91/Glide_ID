package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
)

// CheckIn handles kiosk check-in/check-out requests
func CheckIn(svc *services.AttendanceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Tenant ID not found",
			})
		}

		var req services.CheckInRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Get IP address
		ip := c.IP()
		req.IPAddress = &ip

		resp, err := svc.ProcessCheckIn(c.Context(), tenantID, req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(resp)
	}
}

// ListAttendance lists attendance records
func ListAttendance(svc *services.AttendanceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Tenant ID not found",
			})
		}

		startDate := c.Query("start_date", time.Now().AddDate(0, 0, -6).Format("2006-01-02"))
		endDate := c.Query("end_date", time.Now().Format("2006-01-02"))
		status := c.Query("status")
		userID := c.Query("user_id")
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		startT, err1 := time.Parse("2006-01-02", startDate)
		endT, err2 := time.Parse("2006-01-02", endDate)
		if err1 != nil || err2 != nil || endT.Before(startT) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date range"})
		}
		if limit <= 0 || limit > 500 {
			limit = 50
		}
		if offset < 0 {
			offset = 0
		}

		ctx, cancel := context.WithTimeout(c.Context(), 7*time.Second)
		defer cancel()

		args := []interface{}{tenantID, startDate, endDate}
		where := `
			al.tenant_id = $1
			AND al.punch_time::date >= $2::date
			AND al.punch_time::date <= $3::date
		`
		if status != "" {
			where += fmt.Sprintf(" AND al.status = $%d", len(args)+1)
			args = append(args, status)
		}
		if userID != "" {
			where += fmt.Sprintf(" AND al.user_id = $%d", len(args)+1)
			args = append(args, userID)
		}
		args = append(args, limit, offset)
		limitArg := len(args) - 1
		offsetArg := len(args)

		rows, err := svc.GetDB().Query(ctx, fmt.Sprintf(`
			SELECT
				al.id, al.user_id, u.employee_id, u.first_name, u.last_name,
				al.status, al.punch_time, al.verification_method, al.pin_used,
				al.anomaly_detected, al.anomaly_reason, k.code
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			LEFT JOIN kiosks k ON k.id = al.kiosk_id
			WHERE %s
			ORDER BY al.punch_time DESC
			LIMIT $%d OFFSET $%d
		`, where, limitArg, offsetArg), args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list attendance"})
		}
		defer rows.Close()

		type attendanceRow struct {
			ID                 string  `json:"id"`
			UserID             string  `json:"user_id"`
			EmployeeID         string  `json:"employee_id"`
			FirstName          string  `json:"first_name"`
			LastName           string  `json:"last_name"`
			Status             string  `json:"status"`
			PunchTime          string  `json:"punch_time"`
			VerificationMethod string  `json:"verification_method"`
			PinUsed            bool    `json:"pin_used"`
			AnomalyDetected    bool    `json:"anomaly_detected"`
			AnomalyReason      *string `json:"anomaly_reason"`
			KioskCode          *string `json:"kiosk_code"`
		}

		out := make([]attendanceRow, 0, limit)
		for rows.Next() {
			var r attendanceRow
			var punchTime time.Time
			if err := rows.Scan(
				&r.ID, &r.UserID, &r.EmployeeID, &r.FirstName, &r.LastName,
				&r.Status, &punchTime, &r.VerificationMethod, &r.PinUsed,
				&r.AnomalyDetected, &r.AnomalyReason, &r.KioskCode,
			); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read attendance rows"})
			}
			r.PunchTime = punchTime.UTC().Format(time.RFC3339)
			out = append(out, r)
		}

		return c.JSON(fiber.Map{
			"start_date": startDate,
			"end_date":   endDate,
			"limit":      limit,
			"offset":     offset,
			"rows":       out,
		})
	}
}

// GetAttendance gets a specific attendance record
func GetAttendance(svc *services.AttendanceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Attendance ID is required"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var row struct {
			ID                 string  `json:"id"`
			UserID             string  `json:"user_id"`
			EmployeeID         string  `json:"employee_id"`
			FirstName          string  `json:"first_name"`
			LastName           string  `json:"last_name"`
			Status             string  `json:"status"`
			PunchTime          string  `json:"punch_time"`
			LocalTime          *string `json:"local_time"`
			MonotonicOffsetMs  *int64  `json:"monotonic_offset_ms"`
			VerificationMethod string  `json:"verification_method"`
			PinUsed            bool    `json:"pin_used"`
			AnomalyDetected    bool    `json:"anomaly_detected"`
			AnomalyReason      *string `json:"anomaly_reason"`
			KioskCode          *string `json:"kiosk_code"`
		}
		var punchTime time.Time
		var localTime *time.Time
		err := svc.GetDB().QueryRow(ctx, `
			SELECT
				al.id, al.user_id, u.employee_id, u.first_name, u.last_name,
				al.status, al.punch_time, al.local_time, al.monotonic_offset_ms,
				al.verification_method, al.pin_used, al.anomaly_detected, al.anomaly_reason,
				k.code
			FROM attendance_logs al
			JOIN users u ON u.id = al.user_id
			LEFT JOIN kiosks k ON k.id = al.kiosk_id
			WHERE al.id = $1 AND al.tenant_id = $2
			LIMIT 1
		`, id, tenantID).Scan(
			&row.ID, &row.UserID, &row.EmployeeID, &row.FirstName, &row.LastName,
			&row.Status, &punchTime, &localTime, &row.MonotonicOffsetMs,
			&row.VerificationMethod, &row.PinUsed, &row.AnomalyDetected, &row.AnomalyReason,
			&row.KioskCode,
		)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Attendance not found"})
		}
		row.PunchTime = punchTime.UTC().Format(time.RFC3339)
		if localTime != nil {
			t := localTime.UTC().Format(time.RFC3339)
			row.LocalTime = &t
		}
		return c.JSON(row)
	}
}

// ExportAttendance exports attendance data
func ExportAttendance(svc *services.AttendanceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return exportAttendanceCSV(c, svc, "attendance-export")
	}
}

// GenerateAttendanceReport generates an attendance report
func GenerateAttendanceReport(svc *services.AttendanceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Implementation
		return c.JSON(fiber.Map{"message": "Not implemented"})
	}
}

// ExportReport exports a report
func ExportReport(svc *services.AttendanceService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return exportAttendanceCSV(c, svc, "report-export")
	}
}

func exportAttendanceCSV(c *fiber.Ctx, svc *services.AttendanceService, filePrefix string) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
	}

	startDate := c.Query("start_date", time.Now().AddDate(0, 0, -6).Format("2006-01-02"))
	endDate := c.Query("end_date", time.Now().Format("2006-01-02"))
	status := c.Query("status")
	departmentID := c.Query("department_id")
	userID := c.Query("user_id")
	employeeID := c.Query("employee_id")

	startT, err1 := time.Parse("2006-01-02", startDate)
	endT, err2 := time.Parse("2006-01-02", endDate)
	if err1 != nil || err2 != nil || endT.Before(startT) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid date range"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	args := []interface{}{tenantID, startDate, endDate}
	where := `
		al.tenant_id = $1
		AND al.punch_time::date >= $2::date
		AND al.punch_time::date <= $3::date
	`
	if status != "" {
		where += fmt.Sprintf(" AND al.status = $%d", len(args)+1)
		args = append(args, status)
	}
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

	rows, err := svc.GetDB().Query(ctx, fmt.Sprintf(`
		SELECT
			al.id, u.employee_id, u.first_name, u.last_name, al.status, al.punch_time,
			al.verification_method, al.pin_used, al.anomaly_detected, al.anomaly_reason, k.code
		FROM attendance_logs al
		JOIN users u ON u.id = al.user_id
		LEFT JOIN kiosks k ON k.id = al.kiosk_id
		WHERE %s
		ORDER BY al.punch_time DESC
	`, where), args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to export attendance"})
	}
	defer rows.Close()

	csvRows := [][]string{
		{
			"attendance_id", "employee_id", "first_name", "last_name", "status",
			"punch_time_utc", "verification_method", "pin_used", "anomaly_detected",
			"anomaly_reason", "kiosk_code",
		},
	}

	for rows.Next() {
		var id, empID, firstName, lastName, st, method string
		var punchTime time.Time
		var pinUsed, anomalyDetected bool
		var anomalyReason, kioskCode *string
		if err := rows.Scan(
			&id, &empID, &firstName, &lastName, &st, &punchTime,
			&method, &pinUsed, &anomalyDetected, &anomalyReason, &kioskCode,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read export rows"})
		}

		csvRows = append(csvRows, []string{
			id,
			empID,
			firstName,
			lastName,
			st,
			punchTime.UTC().Format(time.RFC3339),
			method,
			strconv.FormatBool(pinUsed),
			strconv.FormatBool(anomalyDetected),
			valueOrEmpty(anomalyReason),
			valueOrEmpty(kioskCode),
		})
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s-to-%s.csv"`, filePrefix, startDate, endDate))

	w := csv.NewWriter(c)
	if err := w.WriteAll(csvRows); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate CSV"})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to flush CSV"})
	}
	return nil
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
