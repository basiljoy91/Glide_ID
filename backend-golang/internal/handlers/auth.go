package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/models"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// Login handles user login
func Login(authSvc *services.AuthService, userSvc *services.UserService, adminSvc *services.AdminService, auditSvc *services.AuditService, emailSvc services.EmailService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		req.Email = strings.TrimSpace(req.Email)

		if req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Email and password are required",
			})
		}

		user, err := userSvc.FindLoginUserByEmail(c.Context(), req.Email)
		if err != nil {
			if errors.Is(err, services.ErrAmbiguousLoginUser) {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "Multiple organizations use this email. Contact your administrator to consolidate access before using password login.",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}
		if !user.IsActive || user.DeletedAt != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "This account is inactive",
			})
		}
		allowed, _, err := adminSvc.IsIPAllowed(c.Context(), user.TenantID.String(), c.IP())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to validate trusted network rules",
			})
		}
		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access from this network is not allowed",
			})
		}

		// Verify password if user has password auth enabled
		if user.PasswordHash == nil || *user.PasswordHash == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Password login not enabled for this user",
			})
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}
		mfaRequired, err := adminSvc.MFARequiredForRole(c.Context(), user.TenantID.String(), user.Role)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to resolve MFA policy",
			})
		}
		if mfaRequired {
			if emailSvc == nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error": "MFA is enabled but email delivery is not configured",
				})
			}
			challengeID, code, expiresAt, err := authSvc.CreateMFAChallenge(c.Context(), user.TenantID.String(), user.ID.String(), user.Email, c.IP(), 10*time.Minute)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to create MFA challenge",
				})
			}
			if err := emailSvc.SendEmail(c.Context(), services.EmailMessage{
				To:          []string{user.Email},
				Subject:     "Your Glide ID verification code",
				HTMLContent: fmt.Sprintf("<p>Your verification code is <strong>%s</strong>.</p><p>It expires at %s.</p>", code, expiresAt.Format(time.RFC1123)),
			}); err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error": "Failed to deliver MFA code",
				})
			}
			return c.JSON(fiber.Map{
				"mfa_required": true,
				"challenge_id": challengeID,
				"expires_at":   expiresAt,
				"user": fiber.Map{
					"email": user.Email,
					"role":  user.Role,
				},
			})
		}

		return finalizePasswordLogin(c, authSvc, userSvc, adminSvc, auditSvc, user, c.IP(), c.Get("User-Agent"))
	}
}

func VerifyMFALogin(authSvc *services.AuthService, userSvc *services.UserService, adminSvc *services.AdminService, auditSvc *services.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			ChallengeID string `json:"challenge_id"`
			Code        string `json:"code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		userID, tenantID, role, _, err := authSvc.VerifyMFAChallenge(c.Context(), req.ChallengeID, strings.TrimSpace(req.Code))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid verification code"})
		}
		user, err := userSvc.GetUser(c.Context(), tenantID, userID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
		}
		if user.Role != role {
			user.Role = role
		}
		return finalizePasswordLogin(c, authSvc, userSvc, adminSvc, auditSvc, user, c.IP(), c.Get("User-Agent"))
	}
}

func finalizePasswordLogin(c *fiber.Ctx, authSvc *services.AuthService, userSvc *services.UserService, adminSvc *services.AdminService, auditSvc *services.AuditService, user *models.User, ipAddress, userAgent string) error {
	if err := userSvc.UpdateLastLogin(c.Context(), user.TenantID.String(), user.ID.String()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to finalize login",
		})
	}
	sessionExpiry, err := adminSvc.SessionTimeout(c.Context(), user.TenantID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve session policy"})
	}
	token, claims, err := authSvc.GenerateTokenWithMetadata(
		user.ID.String(),
		user.TenantID.String(),
		user.Role,
		user.Email,
		sessionExpiry,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}
	sessionID, err := authSvc.CreateSession(c.Context(), user.TenantID.String(), user.ID.String(), claims.ID, ipAddress, userAgent, claims.ExpiresAt.Time)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create session"})
	}
	permissions, customRole, err := adminSvc.GetEffectivePermissions(c.Context(), user.TenantID.String(), user.ID.String(), user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to resolve permissions"})
	}

	tenantID := user.TenantID
	userID := user.ID
	action := "admin_login"
	resourceType := "user"
	now := time.Now().UTC()
	_ = auditSvc.LogAction(c.Context(), &models.AuditLog{
		TenantID:     &tenantID,
		UserID:       &userID,
		TargetUserID: &userID,
		Action:       action,
		ResourceType: &resourceType,
		ResourceID:   &userID,
		Details: map[string]interface{}{
			"role":       user.Role,
			"email":      user.Email,
			"session_id": sessionID,
		},
		IPAddress: func(v string) *string { return &v }(ipAddress),
		UserAgent: func(v string) *string { return &v }(userAgent),
		CreatedAt: now,
	})

	respUser := fiber.Map{
		"id":          user.ID,
		"tenant_id":   user.TenantID,
		"email":       user.Email,
		"first_name":  user.FirstName,
		"last_name":   user.LastName,
		"role":        user.Role,
		"permissions": permissions,
	}
	if customRole != nil {
		respUser["custom_role"] = customRole
	}

	return c.JSON(fiber.Map{
		"token":      token,
		"session_id": sessionID,
		"user":       respUser,
	})
}
