package handlers

import (
	"context"
	"fmt"
	"time"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
)

// ExportAttendancePDF exports the attendance report as a PDF.
func ExportAttendancePDF(reporting *services.ReportingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}

		start := c.Query("start_date", time.Now().AddDate(0, 0, -6).Format("2006-01-02"))
		end := c.Query("end_date", time.Now().Format("2006-01-02"))
		departmentID := c.Query("department_id")
		userID := c.Query("user_id")
		employeeID := c.Query("employee_id")
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

		ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
		defer cancel()
		pdf, err := reporting.BuildAttendanceReportPDF(ctx, tenantID, start, end, departmentID, userID, employeeID, lateGrace, earlyGrace)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to build report"})
		}

		c.Set("Content-Type", "application/pdf")
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="attendance-report-%s-to-%s.pdf"`, start, end))
		return c.Send(pdf)
	}
}
