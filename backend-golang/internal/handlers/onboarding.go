package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"enterprise-attendance-api/internal/config"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const onboardingAuthMethodPassword = "password"

const (
	onboardingVerificationCooldown   = 60 * time.Second
	onboardingVerificationWindow     = time.Hour
	onboardingVerificationMaxPerHour = 5
)

type onboardingProvisionRequest struct {
	Organization struct {
		Name               string `json:"name"`
		Industry           string `json:"industry"`
		EstimatedEmployees int    `json:"estimated_employees"`
		PlanTier           string `json:"plan_tier"`
	} `json:"organization"`
	Admin struct {
		Email       string `json:"email"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Phone       string `json:"phone"`
		AuthMethod  string `json:"auth_method"`
		Password    string `json:"password,omitempty"`
		SSOEmail    string `json:"sso_email,omitempty"`
		SSOProvider string `json:"sso_provider,omitempty"`
	} `json:"admin"`
	TeamMembers         []onboardingTeamMemberRequest `json:"team_members"`
	EmailVerificationID string                        `json:"email_verification_id"`
}

type onboardingTeamMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type provisionedOnboardingTeamMember struct {
	Email             string
	Role              string
	TemporaryPassword string
}

type onboardingPlanConfig struct {
	MaxUsers           int
	MaxKiosks          int
	BaseAmountCents    int
	PerSeatAmountCents int
	BillingCycle       string
}

var onboardingPlanConfigs = map[string]onboardingPlanConfig{
	"starter": {
		MaxUsers:           25,
		MaxKiosks:          1,
		BaseAmountCents:    19900,
		PerSeatAmountCents: 0,
		BillingCycle:       "monthly",
	},
	"professional": {
		MaxUsers:           250,
		MaxKiosks:          10,
		BaseAmountCents:    49900,
		PerSeatAmountCents: 0,
		BillingCycle:       "monthly",
	},
	"enterprise": {
		MaxUsers:           1000000,
		MaxKiosks:          1000,
		BaseAmountCents:    99900,
		PerSeatAmountCents: 0,
		BillingCycle:       "monthly",
	},
}

// ProvisionOrganization handles the onboarding provisioning request
func ProvisionOrganization(db *pgxpool.Pool, authSvc *services.AuthService, emailSvc services.EmailService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req onboardingProvisionRequest

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		req.Organization.Name = strings.TrimSpace(req.Organization.Name)
		req.Organization.Industry = strings.TrimSpace(req.Organization.Industry)
		req.Organization.PlanTier = normalizeOnboardingPlanTier(req.Organization.PlanTier)
		req.Admin.Email = strings.TrimSpace(strings.ToLower(req.Admin.Email))
		req.Admin.FirstName = strings.TrimSpace(req.Admin.FirstName)
		req.Admin.LastName = strings.TrimSpace(req.Admin.LastName)
		req.Admin.Phone = strings.TrimSpace(req.Admin.Phone)
		req.Admin.AuthMethod = normalizeOnboardingAuthMethod(req.Admin.AuthMethod)
		req.Admin.Password = strings.TrimSpace(req.Admin.Password)

		// Validate required fields
		if req.Organization.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Organization name is required",
			})
		}
		if req.Organization.Industry == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Industry is required",
			})
		}
		planConfig, err := validateOnboardingPlan(req.Organization.PlanTier, req.Organization.EstimatedEmployees)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if req.Admin.Email == "" || req.Admin.FirstName == "" || req.Admin.LastName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Admin details are required",
			})
		}
		if req.EmailVerificationID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Admin email must be verified before provisioning",
			})
		}
		if err := validateOnboardingAuthMethod(req.Admin.AuthMethod); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if req.Admin.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Password is required for onboarding",
			})
		}
		teamMembers, err := normalizeOnboardingTeamMembers(req.TeamMembers, req.Admin.Email)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
		defer cancel()

		tenantID := uuid.New()
		adminUserID := uuid.New()

		// slug + settings
		slugBase := slugify(req.Organization.Name)
		if slugBase == "" {
			slugBase = "org"
		}

		adminDomain := emailDomain(req.Admin.Email)

		settings := map[string]any{
			"industry":           req.Organization.Industry,
			"estimatedEmployees": req.Organization.EstimatedEmployees,
			"adminEmailDomain":   adminDomain,
		}
		settingsJSON, _ := json.Marshal(settings)

		// password hash
		var passwordHash *string
		if err := services.ValidatePasswordWithPolicy(services.DefaultPasswordPolicy(), req.Admin.Password); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Admin.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to hash password",
			})
		}
		h := string(hash)
		passwordHash = &h

		// Generate unique kiosk_code (10 digits) and slug
		var kioskCode string
		var slug string

		tx, err := db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to begin transaction"})
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		// Insert tenant with retries for uniqueness
		const maxAttempts = 10
		for i := 0; i < maxAttempts; i++ {
			slug = slugBase
			if i > 0 {
				slug = fmt.Sprintf("%s-%d", slugBase, i+1)
			}

			kioskCode, err = generateKioskCode10()
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate kiosk code"})
			}

			_, err = tx.Exec(ctx, `
					INSERT INTO tenants (id, name, slug, subscription_tier, max_users, max_kiosks, kiosk_code, settings, sso_provider)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, NULLIF($9, ''))
				`, tenantID, req.Organization.Name, slug, req.Organization.PlanTier, req.Organization.EstimatedEmployees, planConfig.MaxKiosks, kioskCode, string(settingsJSON), "")

			if err == nil {
				break
			}
			if !isUniqueViolation(err) || i == maxAttempts-1 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Failed to create tenant: %v", err)})
			}
			// retry on unique violation
		}

		employeeID := "ADMIN001"
		now := time.Now()
		// Insert admin user
		_, err = tx.Exec(ctx, `
			INSERT INTO users (
				id, tenant_id, employee_id, email, phone, first_name, last_name,
				date_of_joining, role, password_hash,
				is_active, data_privacy_consent, created_at, updated_at
			)
			VALUES ($1,$2,$3,$4, NULLIF($5,''),$6,$7, CURRENT_DATE,$8,$9,true,false,$10,$10)
		`, adminUserID, tenantID, employeeID, req.Admin.Email, req.Admin.Phone, req.Admin.FirstName, req.Admin.LastName, "org_admin", passwordHash, now)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Failed to create admin user: %v", err)})
		}

		_, err = tx.Exec(ctx, `
				INSERT INTO billing_subscriptions (
					tenant_id, plan_tier, status, billing_cycle, seat_count, base_amount_cents,
					per_seat_amount_cents, current_period_start, current_period_end, next_invoice_at, created_at, updated_at
				)
				VALUES (
					$1, $2::subscription_tier, 'active'::billing_subscription_status, $3, $4, $5,
					$6, NOW(), NOW() + INTERVAL '30 days', NOW() + INTERVAL '30 days', $7, $7
				)
			`, tenantID, req.Organization.PlanTier, planConfig.BillingCycle, req.Organization.EstimatedEmployees, planConfig.BaseAmountCents, planConfig.PerSeatAmountCents, now)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to initialize billing subscription"})
		}

		provisionedTeamMembers := make([]provisionedOnboardingTeamMember, 0, len(teamMembers))
		inviteStatus := "created"
		var inviteSentAt *time.Time
		if emailSvc != nil {
			inviteStatus = "sent"
			inviteSentAt = &now
		}
		for index, member := range teamMembers {
			tempPassword, err := generateOnboardingTemporaryPassword(16)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate team member credentials"})
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to secure team member credentials"})
			}
			teamUserID := uuid.New()
			teamEmployeeID := fmt.Sprintf("ADMIN%03d", index+2)
			_, err = tx.Exec(ctx, `
				INSERT INTO users (
					id, tenant_id, employee_id, email, first_name, last_name, date_of_joining, role, password_hash,
					is_active, data_privacy_consent, invite_status, invite_sent_at, created_at, updated_at
				)
				VALUES ($1,$2,$3,$4,$5,$6,CURRENT_DATE,$7,$8,true,false,$9,$10,$11,$11)
			`, teamUserID, tenantID, teamEmployeeID, member.Email, "Invited", "Admin", member.Role, string(hash), inviteStatus, inviteSentAt, now)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Failed to create team member: %v", err)})
			}
			provisionedTeamMembers = append(provisionedTeamMembers, provisionedOnboardingTeamMember{
				Email:             member.Email,
				Role:              member.Role,
				TemporaryPassword: tempPassword,
			})
		}

		// Create initial kiosk record tied to tenant kiosk_code
		kioskID := uuid.New()
		hmacSecret, err := generateSecretHex(32)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate kiosk secret"})
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO kiosks (id, tenant_id, name, code, hmac_secret, status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,'active',$6,$6)
		`, kioskID, tenantID, "Primary Kiosk", kioskCode, hmacSecret, now)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Failed to create kiosk: %v", err)})
		}

		if err := authSvc.ConsumeEmailVerificationChallengeTx(ctx, tx, req.EmailVerificationID, req.Admin.Email, "onboarding_admin"); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Admin email verification is missing or expired"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize admin email verification"})
		}

		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit transaction"})
		}

		if emailSvc != nil {
			_ = emailSvc.SendEmail(ctx, services.EmailMessage{
				To:      []string{req.Admin.Email},
				Subject: "Your Glide ID workspace is ready",
				HTMLContent: fmt.Sprintf(
					"<p>Hi %s,</p><p>Your Glide ID organization <strong>%s</strong> has been provisioned successfully.</p><p>Your kiosk code is <strong>%s</strong>.</p><p>You can now sign in to your admin workspace.</p>",
					req.Admin.FirstName,
					req.Organization.Name,
					kioskCode,
				),
			})
		}
		warnings := []string{}
		if len(provisionedTeamMembers) > 0 {
			loginURL := buildOnboardingAdminLoginURL()
			if emailSvc != nil {
				for _, member := range provisionedTeamMembers {
					if err := emailSvc.SendEmail(ctx, services.EmailMessage{
						To:      []string{member.Email},
						Subject: fmt.Sprintf("You've been added to %s on Glide ID", req.Organization.Name),
						HTMLContent: fmt.Sprintf(
							"<p>You have been added to <strong>%s</strong> as <strong>%s</strong>.</p><p>Sign in at <a href=\"%s\">%s</a> with this temporary password:</p><p><strong>%s</strong></p><p>For security, coordinate with your organization admin to rotate this password after your first sign-in.</p>",
							req.Organization.Name,
							member.Role,
							loginURL,
							loginURL,
							member.TemporaryPassword,
						),
					}); err != nil {
						warnings = append(warnings, fmt.Sprintf("Failed to send invite email to %s", member.Email))
					}
				}
			} else {
				warnings = append(warnings, "Team member accounts were created, but invite emails were not sent because email delivery is not configured.")
			}
		}
		return c.JSON(fiber.Map{
			"success":                true,
			"tenantId":               tenantID.String(),
			"kioskCode":              kioskCode,
			"adminUserId":            adminUserID.String(),
			"teamMembersProvisioned": len(provisionedTeamMembers),
			"teamMemberInvitesSent":  emailSvc != nil,
			"message":                "Organization provisioned successfully",
			"warnings":               warnings,
		})
	}
}

func StartOnboardingEmailVerification(authSvc *services.AuthService, emailSvc services.EmailService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if emailSvc == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Email delivery is not configured"})
		}

		var req struct {
			Email string `json:"email"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		if _, err := mail.ParseAddress(req.Email); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "A valid email address is required"})
		}
		retryAfter, capped, err := authSvc.CheckEmailVerificationRateLimit(c.Context(), req.Email, "onboarding_admin", onboardingVerificationCooldown, onboardingVerificationWindow, onboardingVerificationMaxPerHour)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to validate verification rate limit"})
		}
		if capped {
			c.Set("Retry-After", strconv.Itoa(int(onboardingVerificationWindow.Seconds())))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Too many verification codes requested. Please try again later."})
		}
		if retryAfter > 0 {
			waitSeconds := int(retryAfter.Seconds())
			if waitSeconds < 1 {
				waitSeconds = 1
			}
			c.Set("Retry-After", strconv.Itoa(waitSeconds))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": fmt.Sprintf("Please wait %d seconds before requesting another verification code.", waitSeconds)})
		}

		challengeID, code, expiresAt, err := authSvc.CreateEmailVerificationChallenge(c.Context(), req.Email, "onboarding_admin", 10*time.Minute, c.IP())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create verification code"})
		}
		if err := emailSvc.SendEmail(c.Context(), services.EmailMessage{
			To:      []string{req.Email},
			Subject: "Verify your Glide ID onboarding email",
			HTMLContent: fmt.Sprintf(
				"<p>Your Glide ID verification code is <strong>%s</strong>.</p><p>Enter this code to continue onboarding. It expires at %s.</p>",
				code,
				expiresAt.Format(time.RFC1123),
			),
		}); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Failed to deliver verification code"})
		}

		return c.JSON(fiber.Map{
			"challenge_id": challengeID,
			"expires_at":   expiresAt,
		})
	}
}

func VerifyOnboardingEmailVerification(authSvc *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			ChallengeID string `json:"challenge_id"`
			Email       string `json:"email"`
			Code        string `json:"code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		if req.ChallengeID == "" || req.Email == "" || strings.TrimSpace(req.Code) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "challenge_id, email and code are required"})
		}
		if err := authSvc.VerifyEmailVerificationChallenge(c.Context(), req.ChallengeID, req.Email, strings.TrimSpace(req.Code), "onboarding_admin"); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired verification code"})
		}
		return c.JSON(fiber.Map{
			"verified":     true,
			"challenge_id": req.ChallengeID,
			"email":        req.Email,
		})
	}
}

func generateKioskCode10() (string, error) {
	// 10-digit numeric code with leading zeros
	max := big.NewInt(10000000000) // 1e10
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%010s", n.String()), nil
}

func generateSecretHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// hex encoding length = 2*nBytes
	const hextable = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hextable[v>>4]
		out[i*2+1] = hextable[v&0x0f]
	}
	return string(out), nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func emailDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at == -1 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[at+1:])
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}

func normalizeOnboardingAuthMethod(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func validateOnboardingAuthMethod(authMethod string) error {
	switch authMethod {
	case onboardingAuthMethodPassword:
		return nil
	case "sso":
		return fmt.Errorf("SSO onboarding is not available yet. Use password authentication for now")
	default:
		return fmt.Errorf("auth_method must be 'password'")
	}
}

func normalizeOnboardingPlanTier(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func validateOnboardingPlan(planTier string, estimatedEmployees int) (onboardingPlanConfig, error) {
	config, ok := onboardingPlanConfigs[planTier]
	if !ok {
		return onboardingPlanConfig{}, fmt.Errorf("plan_tier must be 'starter', 'professional', or 'enterprise'")
	}
	if estimatedEmployees <= 0 {
		return onboardingPlanConfig{}, fmt.Errorf("estimated_employees must be greater than zero")
	}
	if planTier != "enterprise" && estimatedEmployees > config.MaxUsers {
		return onboardingPlanConfig{}, fmt.Errorf("%s supports up to %d employees. Choose a higher plan or lower the estimated employee count", planTier, config.MaxUsers)
	}
	return config, nil
}

func normalizeOnboardingTeamMembers(teamMembers []onboardingTeamMemberRequest, adminEmail string) ([]onboardingTeamMemberRequest, error) {
	if len(teamMembers) == 0 {
		return nil, nil
	}
	normalized := make([]onboardingTeamMemberRequest, 0, len(teamMembers))
	seen := map[string]struct{}{
		strings.ToLower(strings.TrimSpace(adminEmail)): {},
	}
	for _, member := range teamMembers {
		member.Email = strings.TrimSpace(strings.ToLower(member.Email))
		member.Role = strings.ToLower(strings.TrimSpace(member.Role))
		if member.Email == "" {
			return nil, fmt.Errorf("team member email is required")
		}
		if _, err := mail.ParseAddress(member.Email); err != nil {
			return nil, fmt.Errorf("team member email must be valid")
		}
		switch member.Role {
		case "org_admin", "hr":
		case "dept_manager":
			return nil, fmt.Errorf("department managers can be assigned after departments are created")
		default:
			return nil, fmt.Errorf("team member role must be 'org_admin' or 'hr'")
		}
		if _, exists := seen[member.Email]; exists {
			return nil, fmt.Errorf("team member emails must be unique and cannot match the primary admin email")
		}
		seen[member.Email] = struct{}{}
		normalized = append(normalized, member)
	}
	return normalized, nil
}

func generateOnboardingTemporaryPassword(length int) (string, error) {
	for attempt := 0; attempt < 10; attempt++ {
		password, err := generateTempPassword(length)
		if err != nil {
			return "", err
		}
		if err := services.ValidatePasswordWithPolicy(services.DefaultPasswordPolicy(), password); err == nil {
			return password, nil
		}
	}
	return "", errors.New("unable to generate onboarding password that satisfies policy")
}

func buildOnboardingAdminLoginURL() string {
	cfg := config.Load()
	baseURL := ""
	if len(cfg.CORSOrigins) > 0 {
		baseURL = strings.TrimRight(strings.TrimSpace(cfg.CORSOrigins[0]), "/")
	}
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	return baseURL + "/admin/login"
}
