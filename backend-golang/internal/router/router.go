package router

import (
	"enterprise-attendance-api/internal/config"
	"enterprise-attendance-api/internal/handlers"
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
)

type Services struct {
	Auth       *services.AuthService
	Attendance *services.AttendanceService
	User       *services.UserService
	HRMS       *services.HRMSService
	Audit      *services.AuditService
	Reporting  *services.ReportingService
	Email      services.EmailService
}

func SetupRoutes(app *fiber.App, svc *Services, cfg *config.Config) {
	// Health check endpoint (for silent ping)
	app.Get("/health", handlers.HealthCheck)

	// Public routes
	public := app.Group("/api/v1/public")
	{
		public.Post("/auth/login", handlers.Login(svc.Auth, svc.User))
		public.Post("/auth/sso/initiate", handlers.InitiateSSO(svc.Attendance.GetDB()))
		public.Post("/auth/sso/callback", handlers.SSOCallback(svc.Auth))
		public.Post("/onboarding/provision", handlers.ProvisionOrganization(svc.Attendance.GetDB()))
		public.Get("/enroll/info/:token", handlers.EnrollInfo(svc.User))
		public.Post("/enroll/face/:token", handlers.EnrollFace(svc.Attendance))
	}

	// Kiosk routes (HMAC authenticated)
	kiosk := app.Group("/api/v1/kiosk")
	kiosk.Use(middleware.HMACAuth(svc.Attendance.GetDB(), cfg.HMACMaxSkewSeconds))
	{
		kiosk.Post("/check-in", handlers.CheckIn(svc.Attendance))
		kiosk.Post("/offline/sync", handlers.KioskOfflineSync(svc.Attendance))
		kiosk.Get("/heartbeat", handlers.KioskHeartbeat(svc.Attendance.GetDB()))
	}

	// Protected routes (JWT authenticated)
	api := app.Group("/api/v1")
	api.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		// Super Admin
		superAdmin := api.Group("/admin/super")
		superAdmin.Use(middleware.RequireRole("super_admin"))
		{
			superAdmin.Get("/metrics", handlers.GetSuperAdminMetrics(svc.Attendance.GetDB()))
			superAdmin.Get("/organizations", handlers.ListSuperAdminOrganizations(svc.Attendance.GetDB()))
			superAdmin.Patch("/organizations/:id/subscription", handlers.UpdateOrganizationSubscription(svc.Attendance.GetDB()))
			superAdmin.Patch("/organizations/:id/status", handlers.SetOrganizationStatus(svc.Attendance.GetDB()))
			superAdmin.Get("/billing/overview", handlers.GetBillingOverview(svc.Attendance.GetDB()))
			superAdmin.Get("/billing/invoices", handlers.ListBillingInvoices(svc.Attendance.GetDB()))
			superAdmin.Post("/billing/invoices", handlers.CreateBillingInvoice(svc.Attendance.GetDB()))
			superAdmin.Patch("/billing/invoices/:id/mark-paid", handlers.MarkBillingInvoicePaid(svc.Attendance.GetDB()))
		}

		// Attendance
		attendance := api.Group("/attendance")
		{
			attendance.Get("/", handlers.ListAttendance(svc.Attendance))
			attendance.Get("/:id", handlers.GetAttendance(svc.Attendance))
			attendance.Post("/export", handlers.ExportAttendance(svc.Attendance))
		}

		// Users
		users := api.Group("/users")
		users.Use(middleware.RequireRole("org_admin", "hr"))
		{
			users.Get("/", handlers.ListUsers(svc.User))
			users.Get("/export", handlers.ExportUsersCSV(svc.User))
			users.Get("/:id", handlers.GetUser(svc.User))
			users.Post("/", handlers.CreateUser(svc.User, svc.Audit))
			users.Post("/bulk/import", handlers.BulkImportUsers(svc.User, svc.Audit))
			users.Post("/bulk/action", handlers.BulkUserAction(svc.User, svc.Audit))
			users.Post("/:id/reset-password", handlers.ResetUserPassword(svc.User, svc.Audit))
			users.Put("/:id", handlers.UpdateUser(svc.User, svc.Audit))
			users.Delete("/:id", handlers.DeleteUser(svc.User, svc.Audit))
			users.Post("/:id/enroll-link", handlers.GenerateEnrollToken(svc.Auth))
		}

		// Departments
		departments := api.Group("/departments")
		departments.Use(middleware.RequireRole("org_admin", "hr"))
		{
			departments.Get("/", handlers.ListDepartments(svc.Attendance.GetDB()))
			departments.Post("/", handlers.CreateDepartment(svc.Attendance.GetDB()))
			departments.Put("/:id", handlers.UpdateDepartment(svc.Attendance.GetDB()))
			departments.Delete("/:id", handlers.DeleteDepartment(svc.Attendance.GetDB()))
		}

		// Kiosks
		kiosks := api.Group("/kiosks")
		kiosks.Use(middleware.RequireRole("org_admin"))
		{
			kiosks.Get("/", handlers.ListKiosks(svc.Attendance.GetDB()))
			kiosks.Post("/", handlers.CreateKiosk(svc.Attendance.GetDB()))
			kiosks.Get("/:id/history", handlers.GetKioskHistory(svc.Attendance.GetDB()))
			kiosks.Post("/:id/rotate-secret", handlers.RotateKioskSecret(svc.Attendance.GetDB()))
			kiosks.Put("/:id", handlers.UpdateKiosk(svc.Attendance.GetDB()))
			kiosks.Delete("/:id", handlers.RevokeKiosk(svc.Attendance.GetDB()))
		}

		// HRMS Integration
		hrms := api.Group("/hrms")
		hrms.Use(middleware.RequireRole("org_admin", "hr"))
		{
			hrms.Get("/integrations", handlers.ListHRMSIntegrations(svc.HRMS))
			hrms.Post("/integrations", handlers.CreateHRMSIntegration(svc.HRMS))
			hrms.Put("/integrations/:id", handlers.UpdateHRMSIntegration(svc.HRMS))
			hrms.Patch("/integrations/:id/toggle", handlers.ToggleHRMSIntegration(svc.HRMS))
			hrms.Post("/integrations/:id/test", handlers.TestHRMSIntegration(svc.HRMS))
			hrms.Get("/integrations/:id/schedule", handlers.GetHRMSSyncSchedule(svc.HRMS))
			hrms.Put("/integrations/:id/schedule", handlers.UpsertHRMSSyncSchedule(svc.HRMS))
			hrms.Delete("/integrations/:id/schedule", handlers.DeleteHRMSSyncSchedule(svc.HRMS))
			hrms.Get("/integrations/:id/sync-logs", handlers.ListHRMSSyncLogs(svc.HRMS))
			hrms.Post("/integrations/:id/sync", handlers.RunHRMSSync(svc.HRMS))
			hrms.Post("/webhooks/:provider", handlers.ProcessHRMSWebhook(svc.HRMS))
			hrms.Post("/export/timesheet", handlers.ExportTimesheet(svc.HRMS))
		}

		// Audit Logs
		audit := api.Group("/audit")
		audit.Use(middleware.RequireRole("org_admin", "hr"))
		{
			audit.Get("/", handlers.ListAuditLogs(svc.Audit))
		}

		// Reports & Exports
		reports := api.Group("/reports")
		reports.Use(middleware.RequireRole("org_admin", "hr", "dept_manager"))
		{
			reports.Get("/org-metrics", handlers.GetOrgMetrics(svc.Attendance.GetDB()))
			reports.Get("/org-metrics/export", handlers.ExportOrgMetrics(svc.Attendance.GetDB()))
			reports.Get("/checkins-7d", handlers.GetCheckins7d(svc.Attendance.GetDB()))
			reports.Get("/anomalies", handlers.ListAnomalies(svc.Attendance.GetDB()))
			reports.Get("/anomalies/:id", handlers.GetAnomaly(svc.Attendance.GetDB()))
			reports.Patch("/anomalies/:id/resolve", handlers.ResolveAnomaly(svc.Attendance.GetDB()))
			reports.Patch("/anomalies/bulk-resolve", handlers.BulkResolveAnomalies(svc.Attendance.GetDB()))
			reports.Get("/attendance", handlers.GetAttendanceReport(svc.Reporting))
			reports.Get("/attendance/pdf", handlers.ExportAttendancePDF(svc.Reporting))
			reports.Post("/send-now", handlers.SendReportNow(svc.Reporting, svc.Email))
			reports.Post("/export", handlers.ExportReport(svc.Attendance))
			reports.Get("/schedules", handlers.ListReportSchedules(svc.Attendance.GetDB()))
			reports.Post("/schedules", handlers.CreateReportSchedule(svc.Attendance.GetDB()))
			reports.Delete("/schedules/:id", handlers.DeleteReportSchedule(svc.Attendance.GetDB()))
			reports.Post("/schedules/:id/run", handlers.RunReportSchedule(svc.Attendance.GetDB()))
			reports.Get("/schedules/:id/logs", handlers.ListReportDeliveryLogs(svc.Attendance.GetDB()))
		}
		// Employee Dashboard
		employee := api.Group("/employee")
		{
			employee.Get("/dashboard", handlers.GetEmployeeDashboard(svc.Attendance.GetDB()))
		}
	}

	// HRMS Webhook endpoints (public, signature verified)
	webhooks := app.Group("/webhooks/hrms")
	{
		webhooks.Post("/:provider", handlers.HRMSWebhook(svc.HRMS))
	}
}
