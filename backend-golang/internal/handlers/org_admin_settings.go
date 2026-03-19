package handlers

import (
	"errors"
	"log"
	"strings"
	"time"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/models"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetOrganizationSettings(adminSvc *services.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		settings, err := adminSvc.GetOrganizationSettings(c.Context(), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load organization settings"})
		}
		return c.JSON(settings)
	}
}

func UpdateOrganizationSettings(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload models.OrganizationSettings
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		settings, err := adminSvc.UpdateOrganizationSettings(c.Context(), middleware.GetTenantID(c), payload)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update organization settings"})
		}
		logAdminAudit(c, auditSvc, "settings_updated", "tenant_settings", nil, map[string]any{"section": "organization"})
		return c.JSON(settings)
	}
}

func GetSecuritySettings(adminSvc *services.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		settings, err := adminSvc.GetSecuritySettings(c.Context(), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load security settings"})
		}
		return c.JSON(settings)
	}
}

func UpdateSecuritySettings(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload models.SecuritySettings
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		settings, err := adminSvc.UpdateSecuritySettings(c.Context(), middleware.GetTenantID(c), payload)
		if err != nil {
			status := fiber.StatusInternalServerError
			message := "Failed to update security settings"
			if errors.Is(err, services.ErrInvalidTrustedIPRange) {
				status = fiber.StatusBadRequest
				message = "One or more trusted IP ranges are invalid"
			}
			return c.Status(status).JSON(fiber.Map{"error": message})
		}
		logAdminAudit(c, auditSvc, "settings_updated", "tenant_settings", nil, map[string]any{"section": "security"})
		return c.JSON(settings)
	}
}

func GetSSOConfiguration(adminSvc *services.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cfg, err := adminSvc.GetSSOConfiguration(c.Context(), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load SSO configuration"})
		}
		return c.JSON(cfg)
	}
}

func UpdateSSOConfiguration(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload models.SSOConfiguration
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		cfg, err := adminSvc.UpdateSSOConfiguration(c.Context(), middleware.GetTenantID(c), payload)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update SSO configuration"})
		}
		logAdminAudit(c, auditSvc, "settings_updated", "sso_config", nil, map[string]any{"provider": cfg.Provider, "enabled": cfg.Enabled})
		return c.JSON(cfg)
	}
}

func ListShiftTemplates(adminSvc *services.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		templates, err := adminSvc.ListShiftTemplates(c.Context(), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load shift templates"})
		}
		return c.JSON(templates)
	}
}

func UpsertShiftTemplate(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload models.ShiftTemplate
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		template, err := adminSvc.UpsertShiftTemplate(c.Context(), middleware.GetTenantID(c), payload)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		logAdminAudit(c, auditSvc, "settings_updated", "shift_template", nil, map[string]any{"template_id": template.ID, "name": template.Name})
		return c.JSON(template)
	}
}

func DeleteShiftTemplate(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := adminSvc.DeleteShiftTemplate(c.Context(), middleware.GetTenantID(c), c.Params("id")); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete shift template"})
		}
		logAdminAudit(c, auditSvc, "settings_updated", "shift_template", nil, map[string]any{"template_id": c.Params("id"), "deleted": true})
		return c.JSON(fiber.Map{"message": "Shift template deleted"})
	}
}

func ListCustomRoles(adminSvc *services.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roles, err := adminSvc.ListCustomRoles(c.Context(), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load custom roles"})
		}
		assignments, err := adminSvc.ListRoleAssignments(c.Context(), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load role assignments"})
		}
		return c.JSON(fiber.Map{
			"roles":       roles,
			"assignments": assignments,
			"permissions": adminSvc.GetPermissionsCatalog(),
		})
	}
}

func CreateCustomRole(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload models.CustomRoleUpsert
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		role, err := adminSvc.CreateCustomRole(c.Context(), middleware.GetTenantID(c), middleware.GetUserID(c), payload)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		logAdminAudit(c, auditSvc, "permission_granted", "custom_role", &role.ID, map[string]any{"name": role.Name, "permissions": role.Permissions})
		return c.Status(fiber.StatusCreated).JSON(role)
	}
}

func UpdateCustomRole(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload models.CustomRoleUpsert
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		role, err := adminSvc.UpdateCustomRole(c.Context(), middleware.GetTenantID(c), c.Params("id"), payload)
		if err != nil {
			status := fiber.StatusInternalServerError
			if errors.Is(err, services.ErrCustomRoleNotFound) {
				status = fiber.StatusNotFound
			}
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}
		logAdminAudit(c, auditSvc, "permission_granted", "custom_role", &role.ID, map[string]any{"name": role.Name, "permissions": role.Permissions, "updated": true})
		return c.JSON(role)
	}
}

func DeleteCustomRole(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleID := c.Params("id")
		if err := adminSvc.DeleteCustomRole(c.Context(), middleware.GetTenantID(c), roleID); err != nil {
			status := fiber.StatusInternalServerError
			if errors.Is(err, services.ErrCustomRoleNotFound) {
				status = fiber.StatusNotFound
			}
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}
		parsedRoleID, _ := uuid.Parse(roleID)
		logAdminAudit(c, auditSvc, "permission_revoked", "custom_role", &parsedRoleID, map[string]any{"deleted": true})
		return c.JSON(fiber.Map{"message": "Custom role deleted"})
	}
}

func AssignCustomRole(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload struct {
			UserID       string  `json:"user_id"`
			CustomRoleID *string `json:"custom_role_id"`
		}
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if strings.TrimSpace(payload.UserID) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id is required"})
		}
		if err := adminSvc.AssignCustomRole(c.Context(), middleware.GetTenantID(c), payload.UserID, payload.CustomRoleID, middleware.GetUserID(c)); err != nil {
			status := fiber.StatusInternalServerError
			switch {
			case errors.Is(err, services.ErrUserNotFound):
				status = fiber.StatusNotFound
			case errors.Is(err, services.ErrCustomRoleNotFound), errors.Is(err, services.ErrCustomRoleUserIneligible):
				status = fiber.StatusBadRequest
			}
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}
		logAdminAudit(c, auditSvc, "permission_granted", "custom_role_assignment", nil, map[string]any{"user_id": payload.UserID, "custom_role_id": payload.CustomRoleID})
		return c.JSON(fiber.Map{"message": "Custom role assignment updated"})
	}
}

func ListActiveSessions(adminSvc *services.AdminService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessions, err := adminSvc.ListSessions(c.Context(), middleware.GetTenantID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load sessions"})
		}
		currentJTI := c.Locals("token_jti")
		return c.JSON(fiber.Map{"sessions": sessions, "current_jti": currentJTI})
	}
}

func RevokeSession(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := adminSvc.RevokeSession(c.Context(), middleware.GetTenantID(c), c.Params("id"), middleware.GetUserID(c)); err != nil {
			status := fiber.StatusInternalServerError
			if errors.Is(err, services.ErrSessionNotFound) {
				status = fiber.StatusNotFound
			}
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}
		logAdminAudit(c, auditSvc, "permission_revoked", "auth_session", nil, map[string]any{"session_id": c.Params("id")})
		return c.JSON(fiber.Map{"message": "Session revoked"})
	}
}

func RevokeUserSessions(adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload struct {
			UserID string `json:"user_id"`
		}
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		count, err := adminSvc.RevokeUserSessions(c.Context(), middleware.GetTenantID(c), strings.TrimSpace(payload.UserID), middleware.GetUserID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to revoke user sessions"})
		}
		logAdminAudit(c, auditSvc, "permission_revoked", "auth_session", nil, map[string]any{"user_id": payload.UserID, "count": count, "scope": "user"})
		return c.JSON(fiber.Map{"revoked_count": count})
	}
}

func logAdminAudit(c *fiber.Ctx, auditSvc *services.AuditService, action, resourceType string, resourceID *uuid.UUID, details map[string]any) {
	tenantID := middleware.GetTenantID(c)
	actorID := middleware.GetUserID(c)
	if tenantID == "" || actorID == "" || auditSvc == nil {
		return
	}
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return
	}
	actorUUID, err := uuid.Parse(actorID)
	if err != nil {
		return
	}
	if err := auditSvc.LogAction(c.Context(), &models.AuditLog{
		TenantID:     &tenantUUID,
		UserID:       &actorUUID,
		TargetUserID: &actorUUID,
		Action:       action,
		ResourceType: &resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    stringPtr(c.IP()),
		UserAgent:    stringPtr(c.Get("User-Agent")),
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		log.Printf("audit log failed: action=%s resource=%s err=%v", action, resourceType, err)
	}
}
