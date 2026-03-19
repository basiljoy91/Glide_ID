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
	Admin      *services.AdminService
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
		public.Post("/auth/login", handlers.Login(svc.Auth, svc.User, svc.Admin, svc.Audit, svc.Email))
		public.Post("/auth/mfa/verify", handlers.VerifyMFALogin(svc.Auth, svc.User, svc.Admin, svc.Audit))
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
	api.Use(middleware.JWTAuth(cfg.JWTSecret, svc.Auth, svc.Admin))
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
		attendance.Use(middleware.RequireAccess(services.PermissionAttendanceView, "org_admin", "hr", "dept_manager"))
		{
			attendance.Get("/", handlers.ListAttendance(svc.Attendance))
			attendance.Get("/:id", handlers.GetAttendance(svc.Attendance))
			attendance.Post("/export", handlers.ExportAttendance(svc.Attendance))
		}

		// Users
		users := api.Group("/users")
		users.Use(middleware.RequireAccess(services.PermissionUsersManage, "org_admin", "hr"))
		{
			users.Get("/", handlers.ListUsers(svc.User))
			users.Get("/export", handlers.ExportUsersCSV(svc.User))
			users.Get("/:id", handlers.GetUser(svc.User))
			users.Post("/", handlers.CreateUser(svc.User, svc.Admin, svc.Audit))
			users.Post("/bulk/import", handlers.BulkImportUsers(svc.User, svc.Admin, svc.Audit))
			users.Post("/bulk/action", handlers.BulkUserAction(svc.User, svc.Audit))
			users.Post("/:id/reset-password", handlers.ResetUserPassword(svc.User, svc.Admin, svc.Audit))
			users.Put("/:id", handlers.UpdateUser(svc.User, svc.Audit))
			users.Delete("/:id", handlers.DeleteUser(svc.User, svc.Audit))
			users.Post("/:id/enroll-link", handlers.GenerateEnrollToken(svc.Auth))
		}

		workforce := api.Group("/workforce")
		workforce.Use(middleware.RequireAccess(services.PermissionUsersManage, "org_admin", "hr"))
		{
			workforce.Get("/employees/:id/profile", handlers.GetEmployeeProfile(svc.User))
			workforce.Put("/employees/:id/profile", handlers.UpdateEmployeeProfile(svc.User, svc.Audit))
			workforce.Get("/employees/:id/emergency-contacts", handlers.ListEmployeeEmergencyContacts(svc.User))
			workforce.Post("/employees/:id/emergency-contacts", handlers.CreateEmployeeEmergencyContact(svc.User))
			workforce.Put("/employees/:id/emergency-contacts/:contactId", handlers.UpdateEmployeeEmergencyContact(svc.User))
			workforce.Delete("/employees/:id/emergency-contacts/:contactId", handlers.DeleteEmployeeEmergencyContact(svc.User))
			workforce.Get("/employees/:id/documents", handlers.ListEmployeeDocuments(svc.User))
			workforce.Post("/employees/:id/documents", handlers.CreateEmployeeDocument(svc.User))
			workforce.Delete("/employees/:id/documents/:documentId", handlers.DeleteEmployeeDocument(svc.User))
			workforce.Post("/employees/:id/invite/resend", handlers.ResendEmployeeInvite(svc.User, svc.Audit))
			workforce.Post("/employees/:id/offboard", handlers.OffboardEmployee(svc.User, svc.Admin, svc.Audit))
			workforce.Get("/bulk-edits", handlers.ListBulkEmployeeEditBatches(svc.Attendance.GetDB()))
			workforce.Post("/bulk-edits/preview", handlers.PreviewBulkEmployeeEdit(svc.User))
			workforce.Post("/bulk-edits/:id/apply", handlers.ApplyBulkEmployeeEdit(svc.Attendance.GetDB()))
			workforce.Post("/bulk-edits/:id/rollback", handlers.RollbackBulkEmployeeEdit(svc.Attendance.GetDB()))
		}

		// Departments
		departments := api.Group("/departments")
		departments.Use(middleware.RequireAccess(services.PermissionDepartmentsManage, "org_admin", "hr"))
		{
			departments.Get("/", handlers.ListDepartments(svc.Attendance.GetDB()))
			departments.Post("/", handlers.CreateDepartment(svc.Attendance.GetDB()))
			departments.Put("/:id", handlers.UpdateDepartment(svc.Attendance.GetDB()))
			departments.Delete("/:id", handlers.DeleteDepartment(svc.Attendance.GetDB()))
		}

		// Kiosks
		kiosks := api.Group("/kiosks")
		kiosks.Use(middleware.RequireAccess(services.PermissionKiosksManage, "org_admin"))
		{
			kiosks.Get("/", handlers.ListKiosks(svc.Attendance.GetDB()))
			kiosks.Get("/dashboard", handlers.GetKioskFleetDashboard(svc.Attendance.GetDB()))
			kiosks.Post("/", handlers.CreateKiosk(svc.Attendance.GetDB()))
			kiosks.Get("/:id/history", handlers.GetKioskHistory(svc.Attendance.GetDB()))
			kiosks.Get("/:id/incidents", handlers.ListKioskIncidents(svc.Attendance.GetDB()))
			kiosks.Patch("/:id/incidents/:incidentId", handlers.UpdateKioskIncident(svc.Attendance.GetDB()))
			kiosks.Get("/:id/commands", handlers.ListKioskCommands(svc.Attendance.GetDB()))
			kiosks.Post("/:id/commands", handlers.QueueKioskCommand(svc.Attendance.GetDB()))
			kiosks.Post("/:id/rotate-secret", handlers.RotateKioskSecret(svc.Attendance.GetDB()))
			kiosks.Put("/:id", handlers.UpdateKiosk(svc.Attendance.GetDB()))
			kiosks.Delete("/:id", handlers.RevokeKiosk(svc.Attendance.GetDB()))
		}

		// HRMS Integration
		hrms := api.Group("/hrms")
		hrms.Use(middleware.RequireAccess(services.PermissionIntegrationsManage, "org_admin", "hr"))
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
			hrms.Get("/integrations/:id/webhook-events", handlers.ListHRMSWebhookEvents(svc.HRMS))
			hrms.Post("/integrations/:id/webhook-events/:eventId/retry", handlers.RetryHRMSWebhookEvent(svc.HRMS))
			hrms.Post("/integrations/:id/mapping-test", handlers.TestHRMSFieldMapping(svc.HRMS))
			hrms.Post("/integrations/:id/dry-run", handlers.DryRunHRMSDirectorySync(svc.HRMS))
			hrms.Get("/integrations/:id/conflicts", handlers.ListHRMSSyncConflicts(svc.HRMS))
			hrms.Patch("/integrations/:id/conflicts/:conflictId", handlers.ResolveHRMSSyncConflict(svc.HRMS))
			hrms.Post("/integrations/:id/rotate-credentials", handlers.RotateHRMSCredentials(svc.HRMS))
			hrms.Post("/webhooks/:provider", handlers.ProcessHRMSWebhook(svc.HRMS))
			hrms.Post("/export/timesheet", handlers.ExportTimesheet(svc.HRMS))
		}

		// Audit Logs
		audit := api.Group("/audit")
		audit.Use(middleware.RequireAccess(services.PermissionAuditView, "org_admin", "hr"))
		{
			audit.Get("/", handlers.ListAuditLogs(svc.Audit))
		}

		// Reports & Exports
		reports := api.Group("/reports")
		reports.Use(middleware.RequireAccess(services.PermissionReportsView, "org_admin", "hr", "dept_manager"))
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
			reports.Get("/views", handlers.ListReportViews(svc.Attendance.GetDB()))
			reports.Post("/views", handlers.CreateReportView(svc.Attendance.GetDB()))
			reports.Put("/views/:id", handlers.UpdateReportView(svc.Attendance.GetDB()))
			reports.Delete("/views/:id", handlers.DeleteReportView(svc.Attendance.GetDB()))
			reports.Get("/schedules", handlers.ListReportSchedules(svc.Attendance.GetDB()))
			reports.Post("/schedules", handlers.CreateReportSchedule(svc.Attendance.GetDB()))
			reports.Put("/schedules/:id", handlers.UpdateReportSchedule(svc.Attendance.GetDB()))
			reports.Delete("/schedules/:id", handlers.DeleteReportSchedule(svc.Attendance.GetDB()))
			reports.Post("/schedules/:id/run", handlers.RunReportSchedule(svc.Attendance.GetDB()))
			reports.Get("/schedules/:id/logs", handlers.ListReportDeliveryLogs(svc.Attendance.GetDB()))
			reports.Get("/export/payroll", handlers.ExportPayrollReport(svc.Reporting))
			reports.Get("/export/compliance", handlers.ExportComplianceReport(svc.Reporting))
		}

		attendanceOps := api.Group("/attendance-ops")
		attendanceOps.Use(middleware.RequireAccess(services.PermissionReviewsManage, "org_admin", "hr", "dept_manager"))
		{
			attendanceOps.Get("/settings", handlers.GetAttendanceOperationsSettings(svc.Attendance.GetDB()))
			attendanceOps.Put("/settings", handlers.UpdateAttendanceOperationsSettings(svc.Attendance.GetDB()))
			attendanceOps.Get("/leave-requests", handlers.ListLeaveRequests(svc.Attendance.GetDB(), false))
			attendanceOps.Post("/leave-requests", handlers.CreateLeaveRequest(svc.Attendance.GetDB(), false))
			attendanceOps.Patch("/leave-requests/:id/review", handlers.ReviewLeaveRequest(svc.Attendance.GetDB()))
			attendanceOps.Get("/regularizations", handlers.ListRegularizationRequests(svc.Attendance.GetDB(), false))
			attendanceOps.Post("/regularizations", handlers.CreateRegularizationRequest(svc.Attendance.GetDB(), false))
			attendanceOps.Patch("/regularizations/:id/review", handlers.ReviewRegularizationRequest(svc.Attendance.GetDB()))
			attendanceOps.Get("/overtime-requests", handlers.ListOvertimeRequests(svc.Attendance.GetDB(), false))
			attendanceOps.Post("/overtime-requests", handlers.CreateOvertimeRequest(svc.Attendance.GetDB(), false))
			attendanceOps.Patch("/overtime-requests/:id/review", handlers.ReviewOvertimeRequest(svc.Attendance.GetDB()))
			attendanceOps.Get("/shifts", handlers.ListShiftAssignments(svc.Attendance.GetDB(), false))
			attendanceOps.Post("/shifts", handlers.CreateShiftAssignment(svc.Attendance.GetDB()))
			attendanceOps.Put("/shifts/:id", handlers.UpdateShiftAssignment(svc.Attendance.GetDB()))
			attendanceOps.Delete("/shifts/:id", handlers.DeleteShiftAssignment(svc.Attendance.GetDB()))
			attendanceOps.Get("/exceptions", handlers.ListAttendanceExceptions(svc.Attendance.GetDB()))
			attendanceOps.Post("/exceptions", handlers.AssignAttendanceException(svc.Attendance.GetDB()))
			attendanceOps.Patch("/exceptions/:id", handlers.ResolveAttendanceExceptionAssignment(svc.Attendance.GetDB()))
		}

		settings := api.Group("/org/settings")
		settings.Use(middleware.RequireAccess(services.PermissionSettingsManage, "org_admin"))
		{
			settings.Get("/", handlers.GetOrganizationSettings(svc.Admin))
			settings.Put("/", handlers.UpdateOrganizationSettings(svc.Admin, svc.Audit))
			settings.Get("/shifts", handlers.ListShiftTemplates(svc.Admin))
			settings.Post("/shifts", handlers.UpsertShiftTemplate(svc.Admin, svc.Audit))
			settings.Put("/shifts/:id", handlers.UpsertShiftTemplate(svc.Admin, svc.Audit))
			settings.Delete("/shifts/:id", handlers.DeleteShiftTemplate(svc.Admin, svc.Audit))
		}

		security := api.Group("/org/security")
		security.Use(middleware.RequireAccess(services.PermissionSecurityManage, "org_admin"))
		{
			security.Get("/", handlers.GetSecuritySettings(svc.Admin))
			security.Put("/", handlers.UpdateSecuritySettings(svc.Admin, svc.Audit))
			security.Get("/sso", handlers.GetSSOConfiguration(svc.Admin))
			security.Put("/sso", handlers.UpdateSSOConfiguration(svc.Admin, svc.Audit))
		}

		access := api.Group("/org/access")
		access.Use(middleware.RequireAccess(services.PermissionRolesManage, "org_admin"))
		{
			access.Get("/roles", handlers.ListCustomRoles(svc.Admin))
			access.Post("/roles", handlers.CreateCustomRole(svc.Admin, svc.Audit))
			access.Put("/roles/:id", handlers.UpdateCustomRole(svc.Admin, svc.Audit))
			access.Delete("/roles/:id", handlers.DeleteCustomRole(svc.Admin, svc.Audit))
			access.Post("/assignments", handlers.AssignCustomRole(svc.Admin, svc.Audit))
		}

		sessions := api.Group("/org/sessions")
		sessions.Use(middleware.RequireAccess(services.PermissionSessionsManage, "org_admin"))
		{
			sessions.Get("/", handlers.ListActiveSessions(svc.Admin))
			sessions.Post("/:id/revoke", handlers.RevokeSession(svc.Admin, svc.Audit))
			sessions.Post("/revoke-user", handlers.RevokeUserSessions(svc.Admin, svc.Audit))
		}

		finance := api.Group("/org/finance")
		finance.Use(middleware.RequireRole("org_admin", "hr"))
		{
			finance.Get("/overview", handlers.GetOrgBillingOverview(svc.Attendance.GetDB()))
			finance.Get("/invoices", handlers.ListOrgBillingInvoices(svc.Attendance.GetDB()))
		}

		support := api.Group("/org/support")
		support.Use(middleware.RequireRole("org_admin", "hr"))
		{
			support.Get("/tickets", handlers.ListSupportTickets(svc.Attendance.GetDB()))
			support.Post("/tickets", handlers.CreateSupportTicket(svc.Attendance.GetDB()))
		}

		notifications := api.Group("/org/notifications")
		notifications.Use(middleware.RequireRole("org_admin", "hr", "dept_manager"))
		{
			notifications.Get("/", handlers.ListOrgNotifications(svc.Attendance.GetDB()))
			notifications.Post("/:id/read", handlers.MarkOrgNotificationRead(svc.Attendance.GetDB()))
		}
		// Employee Dashboard
		employee := api.Group("/employee")
		{
			employee.Get("/dashboard", handlers.GetEmployeeDashboard(svc.Attendance.GetDB()))
			employee.Get("/leave-requests", handlers.ListLeaveRequests(svc.Attendance.GetDB(), true))
			employee.Post("/leave-requests", handlers.CreateLeaveRequest(svc.Attendance.GetDB(), true))
			employee.Get("/regularizations", handlers.ListRegularizationRequests(svc.Attendance.GetDB(), true))
			employee.Post("/regularizations", handlers.CreateRegularizationRequest(svc.Attendance.GetDB(), true))
			employee.Get("/overtime-requests", handlers.ListOvertimeRequests(svc.Attendance.GetDB(), true))
			employee.Post("/overtime-requests", handlers.CreateOvertimeRequest(svc.Attendance.GetDB(), true))
			employee.Get("/shifts", handlers.ListShiftAssignments(svc.Attendance.GetDB(), true))
		}
	}

	// HRMS Webhook endpoints (public, signature verified)
	webhooks := app.Group("/webhooks/hrms")
	{
		webhooks.Post("/:provider", handlers.HRMSWebhook(svc.HRMS))
	}
}
