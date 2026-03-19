package handlers

import (
	"context"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type reportScheduleRow struct {
	ID         uuid.UUID              `json:"id"`
	ReportType string                 `json:"report_type"`
	Name       *string                `json:"name"`
	Frequency  string                 `json:"frequency"`
	DayOfWeek  *int                   `json:"day_of_week,omitempty"`
	TimeOfDay  string                 `json:"time_of_day"`
	Timezone   string                 `json:"timezone"`
	Recipients []string               `json:"recipients"`
	Filters    map[string]interface{} `json:"filters"`
	IsActive   bool                   `json:"is_active"`
	LastSentAt *string                `json:"last_sent_at,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

type reportScheduleBody struct {
	ReportType string                 `json:"report_type"`
	Name       *string                `json:"name"`
	Frequency  string                 `json:"frequency"`
	DayOfWeek  *int                   `json:"day_of_week"`
	TimeOfDay  string                 `json:"time_of_day"`
	Timezone   string                 `json:"timezone"`
	Recipients []string               `json:"recipients"`
	Filters    map[string]interface{} `json:"filters"`
	IsActive   *bool                  `json:"is_active"`
}

func loadReportSchedule(ctx context.Context, db *pgxpool.Pool, tenantID, id string) (reportScheduleRow, error) {
	var r reportScheduleRow
	var timeOfDay time.Time
	var lastSent *time.Time
	var created time.Time
	var updated time.Time
	err := db.QueryRow(ctx, `
		SELECT id, report_type, name, frequency, day_of_week, time_of_day, timezone,
			recipients, filters, is_active, last_sent_at, created_at, updated_at
		FROM report_schedules
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(
		&r.ID, &r.ReportType, &r.Name, &r.Frequency, &r.DayOfWeek, &timeOfDay, &r.Timezone,
		&r.Recipients, &r.Filters, &r.IsActive, &lastSent, &created, &updated,
	)
	if err != nil {
		return reportScheduleRow{}, err
	}
	r.TimeOfDay = timeOfDay.Format("15:04")
	r.CreatedAt = created.UTC().Format(time.RFC3339)
	r.UpdatedAt = updated.UTC().Format(time.RFC3339)
	if lastSent != nil {
		s := lastSent.UTC().Format(time.RFC3339)
		r.LastSentAt = &s
	}
	return r, nil
}

// ListReportSchedules lists report schedules for the tenant.
func ListReportSchedules(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT id, report_type, name, frequency, day_of_week, time_of_day, timezone,
				recipients, filters, is_active, last_sent_at, created_at, updated_at
			FROM report_schedules
			WHERE tenant_id = $1
			ORDER BY created_at DESC
		`, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list schedules"})
		}
		defer rows.Close()

		out := []reportScheduleRow{}
		for rows.Next() {
			var r reportScheduleRow
			var timeOfDay time.Time
			var lastSent *time.Time
			var created time.Time
			var updated time.Time
			if err := rows.Scan(&r.ID, &r.ReportType, &r.Name, &r.Frequency, &r.DayOfWeek, &timeOfDay, &r.Timezone, &r.Recipients, &r.Filters, &r.IsActive, &lastSent, &created, &updated); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read schedules"})
			}
			r.TimeOfDay = timeOfDay.Format("15:04")
			r.CreatedAt = created.UTC().Format(time.RFC3339)
			r.UpdatedAt = updated.UTC().Format(time.RFC3339)
			if lastSent != nil {
				s := lastSent.UTC().Format(time.RFC3339)
				r.LastSentAt = &s
			}
			out = append(out, r)
		}
		return c.JSON(out)
	}
}

// CreateReportSchedule creates a new schedule.
func CreateReportSchedule(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		var body reportScheduleBody
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if body.ReportType == "" || body.Frequency == "" || len(body.Recipients) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "report_type, frequency, recipients required"})
		}
		if body.TimeOfDay == "" {
			body.TimeOfDay = "08:00"
		}
		if body.Timezone == "" {
			body.Timezone = "UTC"
		}
		if body.Filters == nil {
			body.Filters = map[string]interface{}{}
		}
		isActive := true
		if body.IsActive != nil {
			isActive = *body.IsActive
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var r reportScheduleRow
		var timeOfDay time.Time
		var created time.Time
		var updated time.Time
		var lastSent *time.Time
		err := db.QueryRow(ctx, `
			INSERT INTO report_schedules (tenant_id, report_type, name, frequency, day_of_week, time_of_day, timezone, recipients, filters, is_active, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6::time,$7,$8,$9,$10,NOW())
			RETURNING id, report_type, name, frequency, day_of_week, time_of_day, timezone, recipients, filters, is_active, last_sent_at, created_at, updated_at
		`, tenantID, body.ReportType, body.Name, body.Frequency, body.DayOfWeek, body.TimeOfDay, body.Timezone, body.Recipients, body.Filters, isActive).Scan(
			&r.ID, &r.ReportType, &r.Name, &r.Frequency, &r.DayOfWeek, &timeOfDay, &r.Timezone, &r.Recipients, &r.Filters, &r.IsActive, &lastSent, &created, &updated,
		)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to create schedule"})
		}
		r.TimeOfDay = timeOfDay.Format("15:04")
		r.CreatedAt = created.UTC().Format(time.RFC3339)
		r.UpdatedAt = updated.UTC().Format(time.RFC3339)
		if lastSent != nil {
			s := lastSent.UTC().Format(time.RFC3339)
			r.LastSentAt = &s
		}
		return c.Status(fiber.StatusCreated).JSON(r)
	}
}

// UpdateReportSchedule updates an existing schedule, including recipients and active status.
func UpdateReportSchedule(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Schedule ID is required"})
		}

		var body reportScheduleBody
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		current, err := loadReportSchedule(ctx, db, tenantID, id)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Schedule not found"})
		}

		if body.ReportType == "" {
			body.ReportType = current.ReportType
		}
		if body.Name == nil {
			body.Name = current.Name
		}
		if body.Frequency == "" {
			body.Frequency = current.Frequency
		}
		if body.DayOfWeek == nil {
			body.DayOfWeek = current.DayOfWeek
		}
		if body.TimeOfDay == "" {
			body.TimeOfDay = current.TimeOfDay
		}
		if body.Timezone == "" {
			body.Timezone = current.Timezone
		}
		if body.Recipients == nil {
			body.Recipients = current.Recipients
		}
		if len(body.Recipients) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "recipients required"})
		}
		if body.Filters == nil {
			body.Filters = current.Filters
		}
		isActive := current.IsActive
		if body.IsActive != nil {
			isActive = *body.IsActive
		}

		updated, err := db.Exec(ctx, `
			UPDATE report_schedules
			SET report_type = $1,
				name = $2,
				frequency = $3,
				day_of_week = $4,
				time_of_day = $5::time,
				timezone = $6,
				recipients = $7,
				filters = $8,
				is_active = $9,
				updated_at = NOW()
			WHERE tenant_id = $10 AND id = $11
		`, body.ReportType, body.Name, body.Frequency, body.DayOfWeek, body.TimeOfDay, body.Timezone, body.Recipients, body.Filters, isActive, tenantID, id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to update schedule"})
		}
		if updated.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Schedule not found"})
		}

		schedule, err := loadReportSchedule(ctx, db, tenantID, id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load updated schedule"})
		}
		return c.JSON(schedule)
	}
}

// DeleteReportSchedule deletes a schedule.
func DeleteReportSchedule(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Schedule ID is required"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		if _, err := db.Exec(ctx, `DELETE FROM report_schedules WHERE tenant_id = $1 AND id = $2`, tenantID, id); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete schedule"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

// RunReportSchedule simulates sending a report and logs it.
func RunReportSchedule(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Schedule ID is required"})
		}
		var body struct {
			Message string `json:"message"`
		}
		_ = c.BodyParser(&body)
		if body.Message == "" {
			body.Message = "Scheduled report queued"
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var reportType string
		err := db.QueryRow(ctx, `SELECT report_type FROM report_schedules WHERE id = $1 AND tenant_id = $2`, id, tenantID).Scan(&reportType)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Schedule not found"})
		}

		_, err = db.Exec(ctx, `
			INSERT INTO report_delivery_logs (tenant_id, schedule_id, report_type, status, message)
			VALUES ($1,$2,$3,'queued',$4)
		`, tenantID, id, reportType, body.Message)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to log delivery"})
		}

		_, _ = db.Exec(ctx, `UPDATE report_schedules SET last_sent_at = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2`, id, tenantID)

		return c.JSON(fiber.Map{"success": true, "status": "queued"})
	}
}

// ListReportDeliveryLogs lists delivery logs for a schedule.
func ListReportDeliveryLogs(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		id := c.Params("id")
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Schedule ID is required"})
		}
		limit := c.QueryInt("limit", 50)
		if limit <= 0 || limit > 200 {
			limit = 50
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT id, report_type, status, message, delivered_at
			FROM report_delivery_logs
			WHERE tenant_id = $1 AND schedule_id = $2
			ORDER BY delivered_at DESC
			LIMIT $3
		`, tenantID, id, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list delivery logs"})
		}
		defer rows.Close()

		out := []fiber.Map{}
		for rows.Next() {
			var logID uuid.UUID
			var reportType, status string
			var message *string
			var delivered time.Time
			if err := rows.Scan(&logID, &reportType, &status, &message, &delivered); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read delivery logs"})
			}
			out = append(out, fiber.Map{
				"id":           logID,
				"report_type":  reportType,
				"status":       status,
				"message":      message,
				"delivered_at": delivered.UTC().Format(time.RFC3339),
			})
		}
		return c.JSON(out)
	}
}
