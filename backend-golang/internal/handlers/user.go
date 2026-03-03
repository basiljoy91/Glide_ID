package handlers

import (
	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/models"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CreateUser creates a new user
func CreateUser(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := middleware.GetUserID(c)

		var user models.User
		if err := c.BodyParser(&user); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if err := userSvc.CreateUser(c.Context(), tenantID, &user); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Log audit
		tenantUUID := uuid.MustParse(tenantID)
		userUUID := uuid.MustParse(userID)
		auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &userUUID,
			TargetUserID: &user.ID,
			Action:       "user_created",
			ResourceType:  stringPtr("user"),
			ResourceID:   &user.ID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		})

		return c.Status(fiber.StatusCreated).JSON(user)
	}
}

// GetUser gets a user by ID
func GetUser(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := c.Params("id")

		user, err := userSvc.GetUser(c.Context(), tenantID, userID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		return c.JSON(user)
	}
}

// ListUsers lists all users
func ListUsers(userSvc *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		limit := c.QueryInt("limit", 50)
		offset := c.QueryInt("offset", 0)

		users, err := userSvc.ListUsers(c.Context(), tenantID, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(users)
	}
}

// UpdateUser updates a user
func UpdateUser(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		targetUserID := c.Params("id")

		var body models.User
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		updated, err := userSvc.UpdateUserBasic(c.Context(), tenantID, targetUserID, &body)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		tenantUUID := uuid.MustParse(tenantID)
		actorUUID := uuid.MustParse(actorUserID)
		targetUUID := uuid.MustParse(targetUserID)
		auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &actorUUID,
			TargetUserID: &targetUUID,
			Action:       "user_updated",
			ResourceType: stringPtr("user"),
			ResourceID:   &targetUUID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		})

		return c.JSON(updated)
	}
}

// DeleteUser deletes a user
func DeleteUser(userSvc *services.UserService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := middleware.GetUserID(c)
		targetUserID := c.Params("id")

		// Soft delete user
		if err := userSvc.SoftDeleteUser(c.Context(), tenantID, targetUserID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Log audit
		tenantUUID := uuid.MustParse(tenantID)
		userUUID := uuid.MustParse(userID)
		targetUUID := uuid.MustParse(targetUserID)
		auditSvc.LogAction(c.Context(), &models.AuditLog{
			TenantID:     &tenantUUID,
			UserID:       &userUUID,
			TargetUserID: &targetUUID,
			Action:       "user_deleted",
			ResourceType:  stringPtr("user"),
			ResourceID:   &targetUUID,
			IPAddress:    stringPtr(c.IP()),
			UserAgent:    stringPtr(c.Get("User-Agent")),
		})

		return c.JSON(fiber.Map{"message": "User deleted"})
	}
}

func stringPtr(s string) *string {
	return &s
}

