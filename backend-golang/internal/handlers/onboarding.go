package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// ProvisionOrganization handles the onboarding provisioning request
func ProvisionOrganization(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Organization struct {
				Name               string `json:"name"`
				Industry           string `json:"industry"`
				EstimatedEmployees int    `json:"estimated_employees"`
			} `json:"organization"`
			Admin struct {
				Email       string `json:"email"`
				FirstName   string `json:"first_name"`
				LastName    string `json:"last_name"`
				Phone       string `json:"phone"`
				AuthMethod  string `json:"auth_method"` // "sso" or "password"
				Password    string `json:"password,omitempty"`
				SSOEmail    string `json:"sso_email,omitempty"`
				SSOProvider string `json:"sso_provider,omitempty"`
			} `json:"admin"`
			TeamMembers []struct {
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"team_members"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

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
		if req.Admin.Email == "" || req.Admin.FirstName == "" || req.Admin.LastName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Admin details are required",
			})
		}
		if req.Admin.AuthMethod == "password" && req.Admin.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Password is required for password authentication",
			})
		}
		if req.Admin.AuthMethod == "sso" && req.Admin.SSOEmail == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "SSO email is required for SSO authentication",
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
		ssoDomain := ""
		if req.Admin.AuthMethod == "sso" {
			ssoDomain = emailDomain(req.Admin.SSOEmail)
		}

		settings := map[string]any{
			"industry":           req.Organization.Industry,
			"estimatedEmployees": req.Organization.EstimatedEmployees,
			"adminEmailDomain":   adminDomain,
		}
		if ssoDomain != "" {
			settings["sso_domain"] = ssoDomain
		}
		settingsJSON, _ := json.Marshal(settings)

		// password hash
		var passwordHash *string
		if req.Admin.AuthMethod == "password" {
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
		}

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

			ssoProvider := req.Admin.SSOProvider
			if req.Admin.AuthMethod != "sso" {
				ssoProvider = ""
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO tenants (id, name, slug, subscription_tier, kiosk_code, settings, sso_provider)
				VALUES ($1, $2, $3, $4, $5, $6::jsonb, NULLIF($7, ''))
			`, tenantID, req.Organization.Name, slug, "starter", kioskCode, string(settingsJSON), ssoProvider)

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

		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit transaction"})
		}

		// Note: team member invites not persisted yet (no invites table in schema)
		return c.JSON(fiber.Map{
			"success":     true,
			"tenantId":    tenantID.String(),
			"kioskCode":   kioskCode,
			"adminUserId": adminUserID.String(),
			"message":     "Organization provisioned successfully",
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
