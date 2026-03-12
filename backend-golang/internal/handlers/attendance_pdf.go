package handlers

import (
	"context"
	"fmt"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phpdave11/gofpdf"
)

// ExportAttendancePDF exports the attendance report as a PDF.
func ExportAttendancePDF(db *pgxpool.Pool) fiber.Handler {
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
		report, err := buildAttendanceReport(ctx, db, tenantID, start, end, departmentID, userID, employeeID, true, lateGrace, earlyGrace)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to build report"})
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

		c.Set("Content-Type", "application/pdf")
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="attendance-report-%s-to-%s.pdf"`, report.StartDate, report.EndDate))
		return pdf.Output(c)
	}
}
