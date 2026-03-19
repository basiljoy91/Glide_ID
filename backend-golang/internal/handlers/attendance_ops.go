package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type attendanceOperationsSettings struct {
	AllowRemoteAttendance bool     `json:"allow_remote_attendance"`
	GeofencingEnabled     bool     `json:"geofencing_enabled"`
	GeofenceLatitude      *float64 `json:"geofence_latitude"`
	GeofenceLongitude     *float64 `json:"geofence_longitude"`
	GeofenceRadiusMeters  int      `json:"geofence_radius_meters"`
	BreakTrackingEnabled  bool     `json:"break_tracking_enabled"`
	ExceptionSLAHours     int      `json:"exception_sla_hours"`
}

type workflowUserRow struct {
	UserID     uuid.UUID `json:"user_id"`
	EmployeeID string    `json:"employee_id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Department *string   `json:"department_name,omitempty"`
}

type leaveRequestRow struct {
	ID          uuid.UUID       `json:"id"`
	User        workflowUserRow `json:"user"`
	LeaveType   string          `json:"leave_type"`
	StartDate   string          `json:"start_date"`
	EndDate     string          `json:"end_date"`
	DayCount    float64         `json:"day_count"`
	Reason      *string         `json:"reason"`
	Status      string          `json:"status"`
	SubmittedAt time.Time       `json:"submitted_at"`
	ReviewedAt  *time.Time      `json:"reviewed_at"`
	ReviewNote  *string         `json:"review_note"`
}

type regularizationRequestRow struct {
	ID                 uuid.UUID       `json:"id"`
	User               workflowUserRow `json:"user"`
	AttendanceLogID    *uuid.UUID      `json:"attendance_log_id"`
	RequestDate        string          `json:"request_date"`
	RequestedStatus    string          `json:"requested_status"`
	RequestedPunchTime time.Time       `json:"requested_punch_time"`
	Reason             string          `json:"reason"`
	Status             string          `json:"status"`
	SubmittedAt        time.Time       `json:"submitted_at"`
	ReviewedAt         *time.Time      `json:"reviewed_at"`
	ReviewNote         *string         `json:"review_note"`
}

type overtimeRequestRow struct {
	ID               uuid.UUID       `json:"id"`
	User             workflowUserRow `json:"user"`
	WorkDate         string          `json:"work_date"`
	RequestedMinutes int             `json:"requested_minutes"`
	ApprovedMinutes  int             `json:"approved_minutes"`
	Reason           *string         `json:"reason"`
	Status           string          `json:"status"`
	SubmittedAt      time.Time       `json:"submitted_at"`
	ReviewedAt       *time.Time      `json:"reviewed_at"`
	ReviewNote       *string         `json:"review_note"`
}

type shiftAssignmentRow struct {
	ID        uuid.UUID       `json:"id"`
	User      workflowUserRow `json:"user"`
	ShiftName string          `json:"shift_name"`
	StartDate string          `json:"start_date"`
	EndDate   string          `json:"end_date"`
	StartTime string          `json:"start_time"`
	EndTime   string          `json:"end_time"`
	WorkDays  []string        `json:"work_days"`
	IsRota    bool            `json:"is_rota"`
	Notes     *string         `json:"notes"`
	CreatedAt time.Time       `json:"created_at"`
}

type exceptionAssignmentRow struct {
	ID                 uuid.UUID       `json:"id"`
	AttendanceLogID    uuid.UUID       `json:"attendance_log_id"`
	AssignedTo         workflowUserRow `json:"assigned_to"`
	Employee           workflowUserRow `json:"employee"`
	PunchTime          time.Time       `json:"punch_time"`
	Status             string          `json:"status"`
	SLADueAt           *time.Time      `json:"sla_due_at"`
	Note               *string         `json:"note"`
	ResolvedAt         *time.Time      `json:"resolved_at"`
	AnomalyReason      *string         `json:"anomaly_reason"`
	VerificationMethod string          `json:"verification_method"`
}

func defaultAttendanceOperationsSettings() attendanceOperationsSettings {
	return attendanceOperationsSettings{
		AllowRemoteAttendance: false,
		GeofencingEnabled:     false,
		GeofenceRadiusMeters:  200,
		BreakTrackingEnabled:  true,
		ExceptionSLAHours:     24,
	}
}

func loadAttendanceOperationsSettings(ctx fiber.Ctx, db *pgxpool.Pool, tenantID string) (attendanceOperationsSettings, error) {
	settings := defaultAttendanceOperationsSettings()
	var raw []byte
	if err := db.QueryRow(ctx.Context(), `SELECT COALESCE(settings, '{}'::jsonb) FROM tenants WHERE id = $1 AND deleted_at IS NULL`, tenantID).Scan(&raw); err != nil {
		return settings, err
	}
	if len(raw) == 0 {
		return settings, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return settings, err
	}
	if section, ok := doc["attendance_operations"].(map[string]any); ok {
		parsed, _ := json.Marshal(section)
		_ = json.Unmarshal(parsed, &settings)
	}
	if settings.GeofenceRadiusMeters <= 0 {
		settings.GeofenceRadiusMeters = 200
	}
	if settings.ExceptionSLAHours <= 0 {
		settings.ExceptionSLAHours = 24
	}
	return settings, nil
}

func saveAttendanceOperationsSettings(ctx fiber.Ctx, db *pgxpool.Pool, tenantID string, settings attendanceOperationsSettings) error {
	var raw []byte
	if err := db.QueryRow(ctx.Context(), `SELECT COALESCE(settings, '{}'::jsonb) FROM tenants WHERE id = $1 AND deleted_at IS NULL`, tenantID).Scan(&raw); err != nil {
		return err
	}
	var doc map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &doc)
	}
	if doc == nil {
		doc = map[string]any{}
	}
	doc["attendance_operations"] = settings
	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx.Context(), `UPDATE tenants SET settings = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, payload, tenantID)
	return err
}

func resolveWorkflowUserID(c *fiber.Ctx, selfService bool, bodyUserID *string) (string, error) {
	if selfService {
		return middleware.GetUserID(c), nil
	}
	if bodyUserID == nil || strings.TrimSpace(*bodyUserID) == "" {
		return "", errors.New("user_id is required")
	}
	return strings.TrimSpace(*bodyUserID), nil
}

func ensureUserWithinScope(ctx fiber.Ctx, db *pgxpool.Pool, tenantID, userID, scopedDepartmentID string) error {
	if scopedDepartmentID == "" {
		return nil
	}
	var departmentID *uuid.UUID
	if err := db.QueryRow(ctx.Context(), `SELECT department_id FROM users WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, userID, tenantID).Scan(&departmentID); err != nil {
		return err
	}
	if departmentID == nil || departmentID.String() != scopedDepartmentID {
		return errDepartmentScopeMismatch
	}
	return nil
}

func parseStatusForReview(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "approved", "rejected":
		return normalized, nil
	default:
		return "", errors.New("status must be approved or rejected")
	}
}

func parseDateValue(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func GetAttendanceOperationsSettings(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		settings, err := loadAttendanceOperationsSettings(*c, db, middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load attendance operations settings"})
		}
		return c.JSON(settings)
	}
}

func UpdateAttendanceOperationsSettings(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		settings := defaultAttendanceOperationsSettings()
		if err := c.BodyParser(&settings); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if settings.GeofenceRadiusMeters <= 0 {
			settings.GeofenceRadiusMeters = 200
		}
		if settings.ExceptionSLAHours <= 0 {
			settings.ExceptionSLAHours = 24
		}
		if err := saveAttendanceOperationsSettings(*c, db, middleware.GetTenantID(c), settings); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save attendance operations settings"})
		}
		return c.JSON(settings)
	}
}

func CreateLeaveRequest(db *pgxpool.Pool, selfService bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			UserID    *string `json:"user_id"`
			LeaveType string  `json:"leave_type"`
			StartDate string  `json:"start_date"`
			EndDate   string  `json:"end_date"`
			DayCount  float64 `json:"day_count"`
			Reason    *string `json:"reason"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		userID, err := resolveWorkflowUserID(c, selfService, body.UserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		tenantID := middleware.GetTenantID(c)
		actorRole := middleware.GetRole(c)
		actorUserID := middleware.GetUserID(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		if err := ensureUserWithinScope(*c, db, tenantID, userID, scopedDepartmentID); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Requested employee is outside your scope"})
		}
		startDate, err := parseDateValue(body.StartDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start_date"})
		}
		endDate, err := parseDateValue(body.EndDate)
		if err != nil || endDate.Before(startDate) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end_date"})
		}
		dayCount := body.DayCount
		if dayCount <= 0 {
			dayCount = endDate.Sub(startDate).Hours()/24 + 1
		}
		_, err = db.Exec(c.Context(), `
			INSERT INTO leave_requests (id, tenant_id, user_id, leave_type, start_date, end_date, day_count, reason)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, uuid.New(), tenantID, userID, strings.TrimSpace(body.LeaveType), startDate, endDate, dayCount, nullableString(body.Reason))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create leave request"})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true})
	}
}

func ListLeaveRequests(db *pgxpool.Pool, selfService bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		query := `
			SELECT lr.id, u.id, u.employee_id, u.first_name, u.last_name, d.name,
				lr.leave_type, lr.start_date, lr.end_date, lr.day_count, lr.reason, lr.status, lr.submitted_at, lr.reviewed_at, lr.review_note
			FROM leave_requests lr
			JOIN users u ON u.id = lr.user_id
			LEFT JOIN departments d ON d.id = u.department_id
			WHERE lr.tenant_id = $1
		`
		args := []any{tenantID}
		if selfService {
			query += fmt.Sprintf(" AND lr.user_id = $%d", len(args)+1)
			args = append(args, actorUserID)
		} else if scopedDepartmentID != "" {
			query += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		query += ` ORDER BY lr.submitted_at DESC LIMIT 200`
		rows, err := db.Query(c.Context(), query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list leave requests"})
		}
		defer rows.Close()
		result := []leaveRequestRow{}
		for rows.Next() {
			var row leaveRequestRow
			if err := rows.Scan(&row.ID, &row.User.UserID, &row.User.EmployeeID, &row.User.FirstName, &row.User.LastName, &row.User.Department, &row.LeaveType, &row.StartDate, &row.EndDate, &row.DayCount, &row.Reason, &row.Status, &row.SubmittedAt, &row.ReviewedAt, &row.ReviewNote); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read leave requests"})
			}
			result = append(result, row)
		}
		return c.JSON(result)
	}
}

func ReviewLeaveRequest(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Status     string  `json:"status"`
			ReviewNote *string `json:"review_note"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		status, err := parseStatusForReview(body.Status)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		requestID := c.Params("id")
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		if scopedDepartmentID != "" {
			var allowed bool
			if err := db.QueryRow(c.Context(), `
				SELECT EXISTS(
					SELECT 1 FROM leave_requests lr
					JOIN users u ON u.id = lr.user_id
					WHERE lr.id = $1 AND lr.tenant_id = $2 AND u.department_id = $3
				)
			`, requestID, tenantID, scopedDepartmentID).Scan(&allowed); err != nil || !allowed {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Leave request is outside your scope"})
			}
		}
		tag, err := db.Exec(c.Context(), `
			UPDATE leave_requests
			SET status = $1, reviewed_by = $2, reviewed_at = NOW(), review_note = $3, updated_at = NOW()
			WHERE id = $4 AND tenant_id = $5
		`, status, actorUserID, nullableString(body.ReviewNote), requestID, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to review leave request"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Leave request not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func CreateRegularizationRequest(db *pgxpool.Pool, selfService bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			UserID             *string `json:"user_id"`
			AttendanceLogID    *string `json:"attendance_log_id"`
			RequestDate        string  `json:"request_date"`
			RequestedStatus    string  `json:"requested_status"`
			RequestedPunchTime string  `json:"requested_punch_time"`
			Reason             string  `json:"reason"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		userID, err := resolveWorkflowUserID(c, selfService, body.UserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		if err := ensureUserWithinScope(*c, db, tenantID, userID, scopedDepartmentID); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Requested employee is outside your scope"})
		}
		requestDate, err := parseDateValue(body.RequestDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request_date"})
		}
		requestedPunchTime, err := time.Parse(time.RFC3339, strings.TrimSpace(body.RequestedPunchTime))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid requested_punch_time"})
		}
		requestedStatus := strings.TrimSpace(body.RequestedStatus)
		switch requestedStatus {
		case "check_in", "check_out", "break_start", "break_end":
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid requested_status"})
		}
		var attendanceLogID *uuid.UUID
		if body.AttendanceLogID != nil && strings.TrimSpace(*body.AttendanceLogID) != "" {
			parsed, err := uuid.Parse(strings.TrimSpace(*body.AttendanceLogID))
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid attendance_log_id"})
			}
			attendanceLogID = &parsed
		}
		_, err = db.Exec(c.Context(), `
			INSERT INTO attendance_regularization_requests (id, tenant_id, user_id, attendance_log_id, request_date, requested_status, requested_punch_time, reason)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, uuid.New(), tenantID, userID, attendanceLogID, requestDate, requestedStatus, requestedPunchTime, strings.TrimSpace(body.Reason))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create regularization request"})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true})
	}
}

func ListRegularizationRequests(db *pgxpool.Pool, selfService bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		query := `
			SELECT rr.id, u.id, u.employee_id, u.first_name, u.last_name, d.name,
				rr.attendance_log_id, rr.request_date, rr.requested_status::text, rr.requested_punch_time, rr.reason,
				rr.status, rr.submitted_at, rr.reviewed_at, rr.review_note
			FROM attendance_regularization_requests rr
			JOIN users u ON u.id = rr.user_id
			LEFT JOIN departments d ON d.id = u.department_id
			WHERE rr.tenant_id = $1
		`
		args := []any{tenantID}
		if selfService {
			query += fmt.Sprintf(" AND rr.user_id = $%d", len(args)+1)
			args = append(args, actorUserID)
		} else if scopedDepartmentID != "" {
			query += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		query += ` ORDER BY rr.submitted_at DESC LIMIT 200`
		rows, err := db.Query(c.Context(), query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list regularization requests"})
		}
		defer rows.Close()
		result := []regularizationRequestRow{}
		for rows.Next() {
			var row regularizationRequestRow
			if err := rows.Scan(&row.ID, &row.User.UserID, &row.User.EmployeeID, &row.User.FirstName, &row.User.LastName, &row.User.Department, &row.AttendanceLogID, &row.RequestDate, &row.RequestedStatus, &row.RequestedPunchTime, &row.Reason, &row.Status, &row.SubmittedAt, &row.ReviewedAt, &row.ReviewNote); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read regularization requests"})
			}
			result = append(result, row)
		}
		return c.JSON(result)
	}
}

func ReviewRegularizationRequest(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Status     string  `json:"status"`
			ReviewNote *string `json:"review_note"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		status, err := parseStatusForReview(body.Status)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		requestID := c.Params("id")
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		tx, err := db.Begin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start regularization review"})
		}
		defer tx.Rollback(c.Context())
		var userID string
		var attendanceLogID *uuid.UUID
		var requestedStatus string
		var requestedPunchTime time.Time
		if err := tx.QueryRow(c.Context(), `
			SELECT rr.user_id, rr.attendance_log_id, rr.requested_status::text, rr.requested_punch_time
			FROM attendance_regularization_requests rr
			WHERE rr.id = $1 AND rr.tenant_id = $2
		`, requestID, tenantID).Scan(&userID, &attendanceLogID, &requestedStatus, &requestedPunchTime); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Regularization request not found"})
		}
		if err := ensureUserWithinScope(*c, db, tenantID, userID, scopedDepartmentID); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Regularization request is outside your scope"})
		}
		if status == "approved" {
			if attendanceLogID != nil {
				if _, err := tx.Exec(c.Context(), `
					UPDATE attendance_logs
					SET status = $1, punch_time = $2, verification_method = 'manual', updated_at = NOW()
					WHERE id = $3 AND tenant_id = $4
				`, requestedStatus, requestedPunchTime, *attendanceLogID, tenantID); err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to apply regularization to attendance log"})
				}
			} else {
				if _, err := tx.Exec(c.Context(), `
					INSERT INTO attendance_logs (id, tenant_id, user_id, status, punch_time, verification_method, notes)
					VALUES ($1, $2, $3, $4, $5, 'manual', $6)
				`, uuid.New(), tenantID, userID, requestedStatus, requestedPunchTime, "Created from approved regularization request"); err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create corrected attendance log"})
				}
			}
		}
		if _, err := tx.Exec(c.Context(), `
			UPDATE attendance_regularization_requests
			SET status = $1, reviewed_by = $2, reviewed_at = NOW(), review_note = $3, updated_at = NOW()
			WHERE id = $4 AND tenant_id = $5
		`, status, actorUserID, nullableString(body.ReviewNote), requestID, tenantID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to review regularization request"})
		}
		if err := tx.Commit(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize regularization review"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func CreateOvertimeRequest(db *pgxpool.Pool, selfService bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			UserID           *string `json:"user_id"`
			WorkDate         string  `json:"work_date"`
			RequestedMinutes int     `json:"requested_minutes"`
			Reason           *string `json:"reason"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		userID, err := resolveWorkflowUserID(c, selfService, body.UserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		if err := ensureUserWithinScope(*c, db, tenantID, userID, scopedDepartmentID); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Requested employee is outside your scope"})
		}
		workDate, err := parseDateValue(body.WorkDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid work_date"})
		}
		if body.RequestedMinutes <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "requested_minutes must be greater than zero"})
		}
		_, err = db.Exec(c.Context(), `
			INSERT INTO overtime_requests (id, tenant_id, user_id, work_date, requested_minutes, reason)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), tenantID, userID, workDate, body.RequestedMinutes, nullableString(body.Reason))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create overtime request"})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true})
	}
}

func ListOvertimeRequests(db *pgxpool.Pool, selfService bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		query := `
			SELECT ot.id, u.id, u.employee_id, u.first_name, u.last_name, d.name,
				ot.work_date, ot.requested_minutes, ot.approved_minutes, ot.reason,
				ot.status, ot.submitted_at, ot.reviewed_at, ot.review_note
			FROM overtime_requests ot
			JOIN users u ON u.id = ot.user_id
			LEFT JOIN departments d ON d.id = u.department_id
			WHERE ot.tenant_id = $1
		`
		args := []any{tenantID}
		if selfService {
			query += fmt.Sprintf(" AND ot.user_id = $%d", len(args)+1)
			args = append(args, actorUserID)
		} else if scopedDepartmentID != "" {
			query += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		query += ` ORDER BY ot.submitted_at DESC LIMIT 200`
		rows, err := db.Query(c.Context(), query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list overtime requests"})
		}
		defer rows.Close()
		result := []overtimeRequestRow{}
		for rows.Next() {
			var row overtimeRequestRow
			if err := rows.Scan(&row.ID, &row.User.UserID, &row.User.EmployeeID, &row.User.FirstName, &row.User.LastName, &row.User.Department, &row.WorkDate, &row.RequestedMinutes, &row.ApprovedMinutes, &row.Reason, &row.Status, &row.SubmittedAt, &row.ReviewedAt, &row.ReviewNote); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read overtime requests"})
			}
			result = append(result, row)
		}
		return c.JSON(result)
	}
}

func ReviewOvertimeRequest(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Status          string  `json:"status"`
			ApprovedMinutes int     `json:"approved_minutes"`
			ReviewNote      *string `json:"review_note"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		status, err := parseStatusForReview(body.Status)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		requestID := c.Params("id")
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		if scopedDepartmentID != "" {
			var allowed bool
			if err := db.QueryRow(c.Context(), `
				SELECT EXISTS(
					SELECT 1 FROM overtime_requests ot
					JOIN users u ON u.id = ot.user_id
					WHERE ot.id = $1 AND ot.tenant_id = $2 AND u.department_id = $3
				)
			`, requestID, tenantID, scopedDepartmentID).Scan(&allowed); err != nil || !allowed {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Overtime request is outside your scope"})
			}
		}
		approvedMinutes := body.ApprovedMinutes
		if status == "approved" && approvedMinutes <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "approved_minutes must be greater than zero for approved overtime"})
		}
		tag, err := db.Exec(c.Context(), `
			UPDATE overtime_requests
			SET status = $1, approved_minutes = $2, reviewed_by = $3, reviewed_at = NOW(), review_note = $4, updated_at = NOW()
			WHERE id = $5 AND tenant_id = $6
		`, status, approvedMinutes, actorUserID, nullableString(body.ReviewNote), requestID, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to review overtime request"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Overtime request not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func CreateShiftAssignment(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			UserID    string   `json:"user_id"`
			ShiftName string   `json:"shift_name"`
			StartDate string   `json:"start_date"`
			EndDate   string   `json:"end_date"`
			StartTime string   `json:"start_time"`
			EndTime   string   `json:"end_time"`
			WorkDays  []string `json:"work_days"`
			IsRota    bool     `json:"is_rota"`
			Notes     *string  `json:"notes"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		if err := ensureUserWithinScope(*c, db, tenantID, strings.TrimSpace(body.UserID), scopedDepartmentID); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Requested employee is outside your scope"})
		}
		startDate, err := parseDateValue(body.StartDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start_date"})
		}
		endDate, err := parseDateValue(body.EndDate)
		if err != nil || endDate.Before(startDate) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end_date"})
		}
		_, err = db.Exec(c.Context(), `
			INSERT INTO shift_assignments (id, tenant_id, user_id, shift_name, start_date, end_date, start_time, end_time, work_days, is_rota, notes, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7::time, $8::time, $9, $10, $11, $12)
		`, uuid.New(), tenantID, strings.TrimSpace(body.UserID), strings.TrimSpace(body.ShiftName), startDate, endDate, strings.TrimSpace(body.StartTime), strings.TrimSpace(body.EndTime), body.WorkDays, body.IsRota, nullableString(body.Notes), actorUserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create shift assignment"})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true})
	}
}

func UpdateShiftAssignment(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			ShiftName string   `json:"shift_name"`
			StartDate string   `json:"start_date"`
			EndDate   string   `json:"end_date"`
			StartTime string   `json:"start_time"`
			EndTime   string   `json:"end_time"`
			WorkDays  []string `json:"work_days"`
			IsRota    bool     `json:"is_rota"`
			Notes     *string  `json:"notes"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		startDate, err := parseDateValue(body.StartDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid start_date"})
		}
		endDate, err := parseDateValue(body.EndDate)
		if err != nil || endDate.Before(startDate) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid end_date"})
		}
		tag, err := db.Exec(c.Context(), `
			UPDATE shift_assignments
			SET shift_name = $1, start_date = $2, end_date = $3, start_time = $4::time, end_time = $5::time, work_days = $6, is_rota = $7, notes = $8, updated_at = NOW()
			WHERE id = $9 AND tenant_id = $10 AND deleted_at IS NULL
		`, strings.TrimSpace(body.ShiftName), startDate, endDate, strings.TrimSpace(body.StartTime), strings.TrimSpace(body.EndTime), body.WorkDays, body.IsRota, nullableString(body.Notes), c.Params("id"), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update shift assignment"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Shift assignment not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func DeleteShiftAssignment(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tag, err := db.Exec(c.Context(), `UPDATE shift_assignments SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, c.Params("id"), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete shift assignment"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Shift assignment not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func ListShiftAssignments(db *pgxpool.Pool, selfService bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		query := `
			SELECT sa.id, u.id, u.employee_id, u.first_name, u.last_name, d.name,
				sa.shift_name, sa.start_date, sa.end_date, sa.start_time::text, sa.end_time::text, sa.work_days, sa.is_rota, sa.notes, sa.created_at
			FROM shift_assignments sa
			JOIN users u ON u.id = sa.user_id
			LEFT JOIN departments d ON d.id = u.department_id
			WHERE sa.tenant_id = $1 AND sa.deleted_at IS NULL
		`
		args := []any{tenantID}
		if selfService {
			query += fmt.Sprintf(" AND sa.user_id = $%d", len(args)+1)
			args = append(args, actorUserID)
		} else if scopedDepartmentID != "" {
			query += fmt.Sprintf(" AND u.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		query += ` ORDER BY sa.start_date DESC LIMIT 200`
		rows, err := db.Query(c.Context(), query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list shift assignments"})
		}
		defer rows.Close()
		result := []shiftAssignmentRow{}
		for rows.Next() {
			var row shiftAssignmentRow
			if err := rows.Scan(&row.ID, &row.User.UserID, &row.User.EmployeeID, &row.User.FirstName, &row.User.LastName, &row.User.Department, &row.ShiftName, &row.StartDate, &row.EndDate, &row.StartTime, &row.EndTime, &row.WorkDays, &row.IsRota, &row.Notes, &row.CreatedAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read shift assignments"})
			}
			result = append(result, row)
		}
		return c.JSON(result)
	}
}

func ListAttendanceExceptions(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		query := `
			SELECT ea.id, ea.attendance_log_id,
				assignee.id, assignee.employee_id, assignee.first_name, assignee.last_name, ad.name,
				employee.id, employee.employee_id, employee.first_name, employee.last_name, ed.name,
				al.punch_time, ea.status, ea.sla_due_at, ea.note, ea.resolved_at, al.anomaly_reason, al.verification_method
			FROM attendance_exception_assignments ea
			JOIN attendance_logs al ON al.id = ea.attendance_log_id
			JOIN users assignee ON assignee.id = ea.assigned_to
			LEFT JOIN departments ad ON ad.id = assignee.department_id
			JOIN users employee ON employee.id = al.user_id
			LEFT JOIN departments ed ON ed.id = employee.department_id
			WHERE ea.tenant_id = $1
		`
		args := []any{tenantID}
		if scopedDepartmentID != "" {
			query += fmt.Sprintf(" AND employee.department_id = $%d", len(args)+1)
			args = append(args, scopedDepartmentID)
		}
		query += ` ORDER BY COALESCE(ea.sla_due_at, ea.created_at) ASC LIMIT 200`
		rows, err := db.Query(c.Context(), query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list attendance exceptions"})
		}
		defer rows.Close()
		result := []exceptionAssignmentRow{}
		for rows.Next() {
			var row exceptionAssignmentRow
			if err := rows.Scan(&row.ID, &row.AttendanceLogID, &row.AssignedTo.UserID, &row.AssignedTo.EmployeeID, &row.AssignedTo.FirstName, &row.AssignedTo.LastName, &row.AssignedTo.Department, &row.Employee.UserID, &row.Employee.EmployeeID, &row.Employee.FirstName, &row.Employee.LastName, &row.Employee.Department, &row.PunchTime, &row.Status, &row.SLADueAt, &row.Note, &row.ResolvedAt, &row.AnomalyReason, &row.VerificationMethod); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read attendance exceptions"})
			}
			result = append(result, row)
		}
		return c.JSON(result)
	}
}

func AssignAttendanceException(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			AttendanceLogID string  `json:"attendance_log_id"`
			AssignedTo      string  `json:"assigned_to"`
			SLADueAt        *string `json:"sla_due_at"`
			Note            *string `json:"note"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		actorRole := middleware.GetRole(c)
		scopedDepartmentID, err := resolveManagedDepartmentID(c.Context(), db, tenantID, actorUserID, actorRole)
		if err != nil {
			if errors.Is(err, errDepartmentScopeRequired) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Department manager is not assigned to a department"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve department scope"})
		}
		if scopedDepartmentID != "" {
			var allowed bool
			if err := db.QueryRow(c.Context(), `
				SELECT EXISTS(
					SELECT 1 FROM attendance_logs al
					JOIN users u ON u.id = al.user_id
					WHERE al.id = $1 AND al.tenant_id = $2 AND u.department_id = $3
				)
			`, strings.TrimSpace(body.AttendanceLogID), tenantID, scopedDepartmentID).Scan(&allowed); err != nil || !allowed {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Attendance exception is outside your scope"})
			}
		}
		slaDueAt, err := parseOptionalDateTime(body.SLADueAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid sla_due_at"})
		}
		if slaDueAt == nil {
			settings, _ := loadAttendanceOperationsSettings(*c, db, tenantID)
			computed := time.Now().UTC().Add(time.Duration(settings.ExceptionSLAHours) * time.Hour)
			slaDueAt = &computed
		}
		_, err = db.Exec(c.Context(), `
			INSERT INTO attendance_exception_assignments (id, tenant_id, attendance_log_id, assigned_to, assigned_by, sla_due_at, note)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (attendance_log_id)
			DO UPDATE SET assigned_to = EXCLUDED.assigned_to, assigned_by = EXCLUDED.assigned_by, sla_due_at = EXCLUDED.sla_due_at, note = EXCLUDED.note, status = 'open', resolved_at = NULL, updated_at = NOW()
		`, uuid.New(), tenantID, strings.TrimSpace(body.AttendanceLogID), strings.TrimSpace(body.AssignedTo), actorUserID, slaDueAt, nullableString(body.Note))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign attendance exception"})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true})
	}
}

func ResolveAttendanceExceptionAssignment(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Status string  `json:"status"`
			Note   *string `json:"note"`
		}
		_ = c.BodyParser(&body)
		status := strings.ToLower(strings.TrimSpace(body.Status))
		if status == "" {
			status = "resolved"
		}
		if status != "resolved" && status != "open" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be resolved or open"})
		}
		tag, err := db.Exec(c.Context(), `
			UPDATE attendance_exception_assignments
			SET status = $1,
				note = COALESCE($2, note),
				resolved_at = CASE WHEN $1 = 'resolved' THEN NOW() ELSE NULL END,
				updated_at = NOW()
			WHERE id = $3 AND tenant_id = $4
		`, status, nullableString(body.Note), c.Params("id"), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update attendance exception"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Attendance exception not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}
