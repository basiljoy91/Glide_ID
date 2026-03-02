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
	}

	// Kiosk routes (HMAC authenticated)
	kiosk := app.Group("/api/v1/kiosk")
	kiosk.Use(middleware.HMACAuth(svc.Attendance.GetDB()))
	{
		kiosk.Post("/check-in", handlers.CheckIn(svc.Attendance))
		kiosk.Get("/heartbeat", handlers.KioskHeartbeat)
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
			users.Get("/:id", handlers.GetUser(svc.User))
			users.Post("/", handlers.CreateUser(svc.User, svc.Audit))
			users.Put("/:id", handlers.UpdateUser(svc.User, svc.Audit))
			users.Delete("/:id", handlers.DeleteUser(svc.User, svc.Audit))
		}

		// Departments
		departments := api.Group("/departments")
		departments.Use(middleware.RequireRole("org_admin", "hr"))
		{
			departments.Get("/", handlers.ListDepartments)
			departments.Post("/", handlers.CreateDepartment)
			departments.Put("/:id", handlers.UpdateDepartment)
			departments.Delete("/:id", handlers.DeleteDepartment)
		}

		// Kiosks
		kiosks := api.Group("/kiosks")
		kiosks.Use(middleware.RequireRole("org_admin"))
		{
			kiosks.Get("/", handlers.ListKiosks)
			kiosks.Post("/", handlers.CreateKiosk)
			kiosks.Put("/:id", handlers.UpdateKiosk)
			kiosks.Delete("/:id", handlers.RevokeKiosk)
		}

		// HRMS Integration
		hrms := api.Group("/hrms")
		hrms.Use(middleware.RequireRole("org_admin", "hr"))
		{
			hrms.Get("/integrations", handlers.ListHRMSIntegrations(svc.HRMS))
			hrms.Post("/integrations", handlers.CreateHRMSIntegration(svc.HRMS))
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
			reports.Get("/attendance", handlers.GenerateAttendanceReport(svc.Attendance))
			reports.Post("/export", handlers.ExportReport(svc.Attendance))
		}
	}

	// HRMS Webhook endpoints (public, signature verified)
	webhooks := app.Group("/webhooks/hrms")
	{
		webhooks.Post("/:provider", handlers.HRMSWebhook(svc.HRMS))
	}
}

