package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportScheduler struct {
	db     *pgxpool.Pool
	report *ReportingService
	email  EmailService
}

func NewReportScheduler(db *pgxpool.Pool, report *ReportingService, email EmailService) *ReportScheduler {
	return &ReportScheduler{db: db, report: report, email: email}
}

type reportScheduleRow struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	ReportType string
	Frequency  string
	DayOfWeek  *int
	TimeOfDay  time.Time
	Timezone   string
	Recipients []string
	Filters    map[string]interface{}
	LastSentAt *time.Time
}

func (s *ReportScheduler) RunOnce(ctx context.Context) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, report_type, frequency, day_of_week, time_of_day, timezone, recipients, filters, last_sent_at
		FROM report_schedules
		WHERE is_active = true
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var r reportScheduleRow
		if err := rows.Scan(&r.ID, &r.TenantID, &r.ReportType, &r.Frequency, &r.DayOfWeek, &r.TimeOfDay, &r.Timezone, &r.Recipients, &r.Filters, &r.LastSentAt); err != nil {
			continue
		}
		if !scheduleDue(r) {
			continue
		}
		_ = s.sendSchedule(ctx, r)
	}
}

func scheduleDue(s reportScheduleRow) bool {
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	last := time.Time{}
	if s.LastSentAt != nil {
		last = s.LastSentAt.In(loc)
	}

	matchTime := now.Hour() == s.TimeOfDay.Hour() && now.Minute() == s.TimeOfDay.Minute()
	if !matchTime {
		return false
	}

	switch strings.ToLower(s.Frequency) {
	case "daily":
		return last.IsZero() || now.Format("2006-01-02") != last.Format("2006-01-02")
	case "weekly":
		if s.DayOfWeek != nil && int(now.Weekday()) != *s.DayOfWeek {
			return false
		}
		_, nowWeek := now.ISOWeek()
		_, lastWeek := last.ISOWeek()
		return last.IsZero() || nowWeek != lastWeek
	case "monthly":
		return last.IsZero() || now.Month() != last.Month() || now.Year() != last.Year()
	default:
		return false
	}
}

func (s *ReportScheduler) sendSchedule(ctx context.Context, r reportScheduleRow) error {
	// Build report params
	filters := r.Filters
	getStr := func(key string) string {
		if v, ok := filters[key]; ok {
			switch vv := v.(type) {
			case string:
				return vv
			case []byte:
				return string(vv)
			}
		}
		return ""
	}

	start := getStr("start_date")
	end := getStr("end_date")
	if start == "" || end == "" {
		today := time.Now().Format("2006-01-02")
		start = time.Now().AddDate(0, 0, -6).Format("2006-01-02")
		end = today
	}

	departmentID := getStr("department_id")
	userID := getStr("user_id")
	employeeID := getStr("employee_id")
	lateGrace := parseInt(filters["late_grace_minutes"], 10)
	earlyGrace := parseInt(filters["early_grace_minutes"], 10)

	pdf, err := s.report.BuildAttendanceReportPDF(ctx, r.TenantID.String(), start, end, departmentID, userID, employeeID, lateGrace, earlyGrace)
	if err != nil {
		s.logDelivery(ctx, r, "failed", err.Error())
		return err
	}

	subject := fmt.Sprintf("Attendance report %s to %s", start, end)
	body := fmt.Sprintf("<p>Your scheduled attendance report is ready.</p><p>Date range: %s to %s</p>", start, end)
	if err := s.email.SendEmail(ctx, EmailMessage{
		To:          r.Recipients,
		Subject:     subject,
		HTMLContent: body,
		Attachments: []EmailAttachment{{Filename: fmt.Sprintf("attendance-%s-to-%s.pdf", start, end), ContentType: "application/pdf", Content: pdf}},
	}); err != nil {
		s.logDelivery(ctx, r, "failed", err.Error())
		return err
	}

	s.logDelivery(ctx, r, "sent", "sent via Brevo")
	_, _ = s.db.Exec(ctx, `UPDATE report_schedules SET last_sent_at = NOW(), updated_at = NOW() WHERE id = $1`, r.ID)
	return nil
}

func (s *ReportScheduler) logDelivery(ctx context.Context, r reportScheduleRow, status, msg string) {
	_, _ = s.db.Exec(ctx, `
		INSERT INTO report_delivery_logs (tenant_id, schedule_id, report_type, status, message)
		VALUES ($1,$2,$3,$4,$5)
	`, r.TenantID, r.ID, r.ReportType, status, msg)
}

func parseInt(v interface{}, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float64:
		return int(t)
	case float32:
		return int(t)
	case string:
		n, err := strconv.Atoi(t)
		if err == nil {
			return n
		}
	}
	return def
}
