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
		// Dummy initialization for shift and leave since we don't have DB tables for them yet
		resp.UpcomingShift = &UpcomingShift{
			Title:     "Regular Shift",
			StartTime: "09:00 AM",
			EndTime:   "05:00 PM",
		}
		resp.LeaveBalance = LeaveBalance{
			Annual: 12.5,
			Sick:   4.0,
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
