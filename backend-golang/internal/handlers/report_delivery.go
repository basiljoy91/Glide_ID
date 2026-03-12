package handlers

import (
	"context"
	"time"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
)

type sendReportNowRequest struct {
	Recipients        []string `json:"recipients"`
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
	DepartmentID      string   `json:"department_id"`
	UserID            string   `json:"user_id"`
	EmployeeID        string   `json:"employee_id"`
	LateGraceMinutes  int      `json:"late_grace_minutes"`
	EarlyGraceMinutes int      `json:"early_grace_minutes"`
}

// SendReportNow generates and emails the attendance report immediately.
func SendReportNow(reporting *services.ReportingService, email services.EmailService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if email == nil {
			return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
				"error": "Email provider not configured",
			})
		}
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}

		var body sendReportNowRequest
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if len(body.Recipients) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Recipients are required"})
		}
		start := body.StartDate
		end := body.EndDate
		if start == "" || end == "" {
			start = time.Now().AddDate(0, 0, -6).Format("2006-01-02")
			end = time.Now().Format("2006-01-02")
		}
		late := body.LateGraceMinutes
		if late <= 0 {
			late = 10
		}
		early := body.EarlyGraceMinutes
		if early <= 0 {
			early = 10
		}

		ctx, cancel := context.WithTimeout(c.Context(), 20*time.Second)
		defer cancel()

		pdf, err := reporting.BuildAttendanceReportPDF(ctx, tenantID, start, end, body.DepartmentID, body.UserID, body.EmployeeID, late, early)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		subject := "Attendance report " + start + " to " + end
		msg := services.EmailMessage{
			To:          body.Recipients,
			Subject:     subject,
			HTMLContent: "<p>Your attendance report is attached.</p>",
			Attachments: []services.EmailAttachment{{
				Filename:    "attendance-" + start + "-to-" + end + ".pdf",
				ContentType: "application/pdf",
				Content:     pdf,
			}},
		}
		messageID := ""
		if emailWithID, ok := email.(services.EmailServiceWithID); ok {
			var sendErr error
			messageID, sendErr = emailWithID.SendEmailWithID(ctx, msg)
			if sendErr != nil {
				_ = reporting.LogReportDelivery(ctx, tenantID, "", "attendance", "failed", sendErr.Error())
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": sendErr.Error()})
			}
		} else {
			if err := email.SendEmail(ctx, msg); err != nil {
				_ = reporting.LogReportDelivery(ctx, tenantID, "", "attendance", "failed", err.Error())
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}

		logMessage := "send-now"
		if messageID != "" {
			logMessage = "send-now messageId=" + messageID
		}
		_ = reporting.LogReportDelivery(ctx, tenantID, "", "attendance", "sent", logMessage)

		return c.JSON(fiber.Map{"success": true, "message_id": messageID})
	}
}
