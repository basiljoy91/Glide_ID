package services

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phpdave11/gofpdf"
)

type AttendanceReportDay struct {
	Date            string `json:"date"`
	CheckIns        int    `json:"check_ins"`
	CheckOuts       int    `json:"check_outs"`
	Anomalies       int    `json:"anomalies"`
	LateArrivals    int    `json:"late_arrivals"`
	EarlyDepartures int    `json:"early_departures"`
}

type ShiftSummaryRow struct {
	ShiftStart      *string `json:"shift_start_time,omitempty"`
	ShiftEnd        *string `json:"shift_end_time,omitempty"`
	Users           int     `json:"users"`
	CheckIns        int     `json:"check_ins"`
	CheckOuts       int     `json:"check_outs"`
	LateArrivals    int     `json:"late_arrivals"`
	EarlyDepartures int     `json:"early_departures"`
}

type AttendanceReportResponse struct {
	StartDate string                `json:"start_date"`
	EndDate   string                `json:"end_date"`
	Filters   map[string]string     `json:"filters,omitempty"`
	Days      []AttendanceReportDay `json:"days"`
	Totals    struct {
		CheckIns        int `json:"check_ins"`
		CheckOuts       int `json:"check_outs"`
		Anomalies       int `json:"anomalies"`
		Logs            int `json:"logs"`
		LateArrivals    int `json:"late_arrivals"`
		EarlyDepartures int `json:"early_departures"`
	} `json:"totals"`
	ShiftSummary []ShiftSummaryRow `json:"shift_summary,omitempty"`
}

type ReportingService struct {
	db *pgxpool.Pool
}

func NewReportingService(db *pgxpool.Pool) *ReportingService {
	return &ReportingService{db: db}
}

func (s *ReportingService) GetDB() *pgxpool.Pool {
	return s.db
}

func (s *ReportingService) LogReportDelivery(ctx context.Context, tenantID, scheduleID, reportType, status, message string) error {
	var scheduleRef *string
	if scheduleID != "" {
		scheduleRef = &scheduleID
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO report_delivery_logs (tenant_id, schedule_id, report_type, status, message)
		VALUES ($1, $2, $3, $4, $5)
	`, tenantID, scheduleRef, reportType, status, message)
	return err
}

func (s *ReportingService) BuildAttendanceReport(ctx context.Context, tenantID, start, end, departmentID, userID, employeeID string, includeShift bool, lateGrace, earlyGrace int) (AttendanceReportResponse, error) {
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

	rows, err := s.db.Query(ctx, fmt.Sprintf(`
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
					  AND first_in::time > (shift_start_time::time + ($%d::int * INTERVAL '1 minute'))
				) AS late_arrivals,
				COUNT(*) FILTER (
					WHERE last_out IS NOT NULL
					  AND shift_end_time IS NOT NULL
					  AND last_out::time < (shift_end_time::time - ($%d::int * INTERVAL '1 minute'))
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
		return AttendanceReportResponse{}, fmt.Errorf("failed to generate report: %w", err)
	}
	defer rows.Close()

	resp := AttendanceReportResponse{
		StartDate: start,
		EndDate:   end,
		Days:      []AttendanceReportDay{},
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
			return AttendanceReportResponse{}, fmt.Errorf("failed to read report: %w", err)
		}
		resp.Days = append(resp.Days, AttendanceReportDay{
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
		shiftRows, err := s.db.Query(ctx, fmt.Sprintf(`
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
					  AND first_in::time > (shift_start_time::time + ($%d::int * INTERVAL '1 minute'))
				) AS late_arrivals,
				COUNT(*) FILTER (
					WHERE last_out IS NOT NULL
					  AND shift_end_time IS NOT NULL
					  AND last_out::time < (shift_end_time::time - ($%d::int * INTERVAL '1 minute'))
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
					return AttendanceReportResponse{}, fmt.Errorf("failed to read shift summary: %w", err)
				}
				resp.ShiftSummary = append(resp.ShiftSummary, ShiftSummaryRow{
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

func (s *ReportingService) BuildAttendanceReportPDF(ctx context.Context, tenantID, start, end, departmentID, userID, employeeID string, lateGrace, earlyGrace int) ([]byte, error) {
	report, err := s.BuildAttendanceReport(ctx, tenantID, start, end, departmentID, userID, employeeID, true, lateGrace, earlyGrace)
	if err != nil {
		return nil, err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Attendance Report", false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, "Attendance Report")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 7, fmt.Sprintf("Date range: %s to %s", report.StartDate, report.EndDate))
	pdf.Ln(6)
	pdf.Cell(0, 7, fmt.Sprintf("Totals: %d check-ins, %d check-outs, %d anomalies", report.Totals.CheckIns, report.Totals.CheckOuts, report.Totals.Anomalies))
	pdf.Ln(6)
	pdf.Cell(0, 7, fmt.Sprintf("Late arrivals: %d, Early departures: %d", report.Totals.LateArrivals, report.Totals.EarlyDepartures))
	pdf.Ln(10)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(30, 7, "Date")
	pdf.Cell(25, 7, "In")
	pdf.Cell(25, 7, "Out")
	pdf.Cell(25, 7, "Anom")
	pdf.Cell(25, 7, "Late")
	pdf.Cell(25, 7, "Early")
	pdf.Ln(7)
	pdf.SetFont("Helvetica", "", 10)
	for _, d := range report.Days {
		pdf.Cell(30, 6, d.Date)
		pdf.Cell(25, 6, fmt.Sprintf("%d", d.CheckIns))
		pdf.Cell(25, 6, fmt.Sprintf("%d", d.CheckOuts))
		pdf.Cell(25, 6, fmt.Sprintf("%d", d.Anomalies))
		pdf.Cell(25, 6, fmt.Sprintf("%d", d.LateArrivals))
		pdf.Cell(25, 6, fmt.Sprintf("%d", d.EarlyDepartures))
		pdf.Ln(6)
	}

	buf := &bytes.Buffer{}
	if err := pdf.Output(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
