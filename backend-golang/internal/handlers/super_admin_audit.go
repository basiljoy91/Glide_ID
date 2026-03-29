package handlers

import (
	"log"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/models"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func logSuperAdminAudit(c *fiber.Ctx, auditSvc *services.AuditService, targetTenantID string, action, resourceType string, resourceID *uuid.UUID, details map[string]any) {
	if auditSvc == nil {
		return
	}
	actorID := middleware.GetUserID(c)
	if actorID == "" {
		return
	}
	actorUUID, err := uuid.Parse(actorID)
	if err != nil {
		return
	}

	payload := map[string]any{
		"scope":      "super_admin",
		"actor_role": "super_admin",
	}
	for key, value := range details {
		payload[key] = value
	}

	var tenantUUID *uuid.UUID
	if targetTenantID != "" {
		parsedTenantID, err := uuid.Parse(targetTenantID)
		if err == nil {
			tenantUUID = &parsedTenantID
		}
	}

	if err := auditSvc.LogAction(c.Context(), &models.AuditLog{
		TenantID:     tenantUUID,
		UserID:       &actorUUID,
		Action:       action,
		ResourceType: stringPtr(resourceType),
		ResourceID:   resourceID,
		Details:      payload,
		IPAddress:    stringPtr(c.IP()),
		UserAgent:    stringPtr(c.Get("User-Agent")),
	}); err != nil {
		log.Printf("super admin audit log failed: action=%s tenant=%s resource=%s err=%v", action, targetTenantID, resourceType, err)
	}
}
