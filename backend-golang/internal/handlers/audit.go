package handlers

import (
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ListAuditLogs lists audit logs
func ListAuditLogs(auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Tenant ID not found",
			})
		}

		limit := c.QueryInt("limit", 50)
		if limit <= 0 {
			limit = 50
		}
		if limit > 200 {
			limit = 200
		}
		page := c.QueryInt("page", 1)
		if page <= 0 {
			page = 1
		}
		offset := (page - 1) * limit

		action := strings.TrimSpace(c.Query("action"))
		query := strings.TrimSpace(c.Query("q"))
		logs, total, err := auditSvc.ListLogs(c.Context(), tenantID, services.AuditLogFilters{
			Action: action,
			Query:  query,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to list audit logs",
			})
		}

		return c.JSON(fiber.Map{
			"data": logs,
			"meta": fiber.Map{
				"total": total,
				"page":  page,
				"limit": limit,
			},
		})
	}
}
