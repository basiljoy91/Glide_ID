package handlers

import (
	"context"
	"errors"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type kioskDTO struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Code            string     `json:"code"`
	Status          string     `json:"status"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ListKiosks lists kiosks for the tenant
func ListKiosks(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		statusFilter := c.Query("status", "")
		if statusFilter != "" && statusFilter != "active" && statusFilter != "inactive" && statusFilter != "revoked" && statusFilter != "maintenance" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid status filter"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open transaction"})
		}
		defer func() { _ = tx.Rollback(ctx) }()

		_, _ = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)

		baseSQL := `
			SELECT id, name, code, status, last_heartbeat_at, created_at, updated_at
			FROM kiosks
			WHERE tenant_id = $1
		`
		args := []any{tenantID}
		if statusFilter != "" {
			baseSQL += " AND status = $2"
			args = append(args, statusFilter)
		}
		baseSQL += " ORDER BY created_at DESC"

		rows, err := tx.Query(ctx, baseSQL, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list kiosks"})
		}
		defer rows.Close()

		out := make([]kioskDTO, 0)
		for rows.Next() {
			var k kioskDTO
			if err := rows.Scan(&k.ID, &k.Name, &k.Code, &k.Status, &k.LastHeartbeatAt, &k.CreatedAt, &k.UpdatedAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read kiosks"})
			}
			out = append(out, k)
		}
		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize list kiosks"})
		}
		return c.JSON(out)
	}
}

// CreateKiosk creates a kiosk
func CreateKiosk(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var body struct {
			Name string `json:"name"`
			Code string `json:"code"`
		}
		if err := c.BodyParser(&body); err != nil || body.Name == "" || body.Code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and code are required"})
		}

		hmacSecret, err := generateSecretHex(32)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate kiosk secret"})
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open transaction"})
		}
		defer func() { _ = tx.Rollback(ctx) }()
		_, _ = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)

		var k kioskDTO
		err = tx.QueryRow(ctx, `
			INSERT INTO kiosks (tenant_id, name, code, hmac_secret, status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,'active',NOW(),NOW())
			RETURNING id, name, code, status, last_heartbeat_at, created_at, updated_at
		`, tenantID, body.Name, body.Code, hmacSecret).Scan(
			&k.ID, &k.Name, &k.Code, &k.Status, &k.LastHeartbeatAt, &k.CreatedAt, &k.UpdatedAt,
		)
		if err != nil {
			// Surface uniqueness violations more nicely.
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Kiosk code already exists for this tenant"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create kiosk"})
		}

		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize create kiosk"})
		}
		return c.Status(fiber.StatusCreated).JSON(k)
	}
}

// UpdateKiosk updates a kiosk
func UpdateKiosk(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		kioskID := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var body struct {
			Name   *string `json:"name"`
			Status *string `json:"status"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if body.Status != nil {
			switch *body.Status {
			case "active", "inactive", "maintenance", "revoked":
			default:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid status"})
			}
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open transaction"})
		}
		defer func() { _ = tx.Rollback(ctx) }()
		_, _ = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)

		var k kioskDTO
		err = tx.QueryRow(ctx, `
			UPDATE kiosks
			SET
				name = COALESCE($1, name),
				status = COALESCE($2, status),
				updated_at = NOW()
			WHERE id = $3 AND tenant_id = $4
			RETURNING id, name, code, status, last_heartbeat_at, created_at, updated_at
		`, body.Name, body.Status, kioskID, tenantID).Scan(
			&k.ID, &k.Name, &k.Code, &k.Status, &k.LastHeartbeatAt, &k.CreatedAt, &k.UpdatedAt,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update kiosk"})
		}

		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize update kiosk"})
		}
		return c.JSON(k)
	}
}

// RevokeKiosk revokes a kiosk
func RevokeKiosk(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		kioskID := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open transaction"})
		}
		defer func() { _ = tx.Rollback(ctx) }()
		_, _ = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)

		_, err = tx.Exec(ctx, `
			UPDATE kiosks
			SET status = 'revoked', revoked_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2
		`, kioskID, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to revoke kiosk"})
		}

		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize revoke kiosk"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

// RotateKioskSecret rotates the kiosk HMAC secret and returns the new value once.
func RotateKioskSecret(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}
		kioskID := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		newSecret, err := generateSecretHex(32)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate kiosk secret"})
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open transaction"})
		}
		defer func() { _ = tx.Rollback(ctx) }()
		_, _ = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)

		var kioskCode string
		err = tx.QueryRow(ctx, `
			UPDATE kiosks
			SET hmac_secret = $1, updated_at = NOW()
			WHERE id = $2 AND tenant_id = $3
			RETURNING code
		`, newSecret, kioskID, tenantID).Scan(&kioskCode)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kiosk not found"})
		}

		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize secret rotation"})
		}

		return c.JSON(fiber.Map{
			"success":     true,
			"kiosk_id":    kioskID,
			"kiosk_code":  kioskCode,
			"hmac_secret": newSecret,
		})
	}
}
