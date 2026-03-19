package handlers

import (
	"context"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmployeeDashboardResponse struct {
	TodayCheckIns []AttendanceLogSimple `json:"todayCheckIns"`
	RecentHistory []AttendanceLogSimple `json:"recentHistory"`
	UpcomingShift *UpcomingShift        `json:"upcomingShift"`
	LeaveBalance  LeaveBalance          `json:"leaveBalance"`
}

type AttendanceLogSimple struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	PunchTime time.Time `json:"punchTime"`
}

type UpcomingShift struct {
	Title     string `json:"title"`
	StartTime string `json:"startTime"` // e.g., "09:00 AM"
	EndTime   string `json:"endTime"`   // e.g., "05:00 PM"
}

type LeaveBalance struct {
	Annual float64 `json:"annual"`
	Sick   float64 `json:"sick"`
}

// GetEmployeeDashboard returns personal metrics for the logged-in employee.
func GetEmployeeDashboard(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := c.Locals("user_id")

		if tenantID == "" || userID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var resp EmployeeDashboardResponse
		resp.LeaveBalance = LeaveBalance{
			Annual: 18,
			Sick:   8,
		}

		var shiftTitle, shiftStart, shiftEnd *string
		err := db.QueryRow(ctx, `
			SELECT shift_name, start_time::text, end_time::text
			FROM shift_assignments
			WHERE tenant_id = $1
			  AND user_id = $2
			  AND deleted_at IS NULL
			  AND end_date >= CURRENT_DATE
			ORDER BY start_date ASC, created_at ASC
			LIMIT 1
		`, tenantID, userID).Scan(&shiftTitle, &shiftStart, &shiftEnd)
		if err == nil && shiftTitle != nil && shiftStart != nil && shiftEnd != nil {
			resp.UpcomingShift = &UpcomingShift{
				Title:     *shiftTitle,
				StartTime: *shiftStart,
				EndTime:   *shiftEnd,
			}
		}

		leaveRows, leaveErr := db.Query(ctx, `
			SELECT leave_type, COALESCE(SUM(day_count), 0)
			FROM leave_requests
			WHERE tenant_id = $1
			  AND user_id = $2
			  AND status = 'approved'
			  AND start_date >= date_trunc('year', CURRENT_DATE)::date
			  AND end_date < (date_trunc('year', CURRENT_DATE) + INTERVAL '1 year')::date
			GROUP BY leave_type
		`, tenantID, userID)
		if leaveErr == nil {
			for leaveRows.Next() {
				var leaveType string
				var used float64
				if err := leaveRows.Scan(&leaveType, &used); err == nil {
					switch leaveType {
					case "annual":
						resp.LeaveBalance.Annual -= used
					case "sick":
						resp.LeaveBalance.Sick -= used
					}
				}
			}
			leaveRows.Close()
		}
		if resp.LeaveBalance.Annual < 0 {
			resp.LeaveBalance.Annual = 0
		}
		if resp.LeaveBalance.Sick < 0 {
			resp.LeaveBalance.Sick = 0
		}

		// Fetch today's check-ins (limit to 5 just in case)
		rows, err := db.Query(ctx, `
			SELECT id, status, punch_time
			FROM attendance_logs
			WHERE tenant_id = $1 AND user_id = $2 AND punch_time::date = CURRENT_DATE
			ORDER BY punch_time ASC
			LIMIT 5
		`, tenantID, userID)
		if err == nil {
			for rows.Next() {
				var log AttendanceLogSimple
				var punchTime time.Time
				if err := rows.Scan(&log.ID, &log.Status, &punchTime); err == nil {
					log.PunchTime = punchTime.UTC()
					resp.TodayCheckIns = append(resp.TodayCheckIns, log)
				}
			}
			rows.Close()
		}

		// Fetch recent history (limit 10)
		rowsHist, errHist := db.Query(ctx, `
			SELECT id, status, punch_time
			FROM attendance_logs
			WHERE tenant_id = $1 AND user_id = $2
			ORDER BY punch_time DESC
			LIMIT 10
		`, tenantID, userID)
		if errHist == nil {
			for rowsHist.Next() {
				var log AttendanceLogSimple
				var punchTime time.Time
				if err := rowsHist.Scan(&log.ID, &log.Status, &punchTime); err == nil {
					log.PunchTime = punchTime.UTC()
					resp.RecentHistory = append(resp.RecentHistory, log)
				}
			}
			rowsHist.Close()
		}

		if resp.TodayCheckIns == nil {
			resp.TodayCheckIns = []AttendanceLogSimple{}
		}
		if resp.RecentHistory == nil {
			resp.RecentHistory = []AttendanceLogSimple{}
		}

		return c.JSON(resp)
	}
}
