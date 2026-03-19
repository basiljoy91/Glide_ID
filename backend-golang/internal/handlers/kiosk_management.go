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
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Code               string     `json:"code"`
	Status             string     `json:"status"`
	Location           *string    `json:"location"`
	LastHeartbeatAt    *time.Time `json:"last_heartbeat_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	HealthStatus       string     `json:"health_status"`
	AppVersion         *string    `json:"app_version,omitempty"`
	OSVersion          *string    `json:"os_version,omitempty"`
	BatteryPercent     *int       `json:"battery_percent,omitempty"`
	NetworkStrength    *int       `json:"network_strength,omitempty"`
	MemoryUsagePercent *int       `json:"memory_usage_percent,omitempty"`
	StorageFreeMB      *int       `json:"storage_free_mb,omitempty"`
	OpenIncidents      int        `json:"open_incidents"`
	PendingCommands    int        `json:"pending_commands"`
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
			WITH latest AS (
				SELECT DISTINCT ON (kiosk_id)
					kiosk_id, app_version, os_version, battery_percent, network_strength, memory_usage_percent, storage_free_mb
				FROM kiosk_telemetry_samples
				WHERE tenant_id = $1
				ORDER BY kiosk_id, recorded_at DESC
			),
			open_incidents AS (
				SELECT kiosk_id, COUNT(*) AS open_incidents
				FROM kiosk_incidents
				WHERE tenant_id = $1 AND status IN ('open', 'acknowledged')
				GROUP BY kiosk_id
			),
			pending_commands AS (
				SELECT kiosk_id, COUNT(*) AS pending_commands
				FROM kiosk_commands
				WHERE tenant_id = $1 AND status IN ('queued', 'delivered')
				GROUP BY kiosk_id
			)
			SELECT
				k.id, k.name, k.code, k.status, k.location, k.last_heartbeat_at, k.created_at, k.updated_at,
				l.app_version, l.os_version, l.battery_percent, l.network_strength, l.memory_usage_percent, l.storage_free_mb,
				COALESCE(oi.open_incidents, 0),
				COALESCE(pc.pending_commands, 0)
			FROM kiosks k
			LEFT JOIN latest l ON l.kiosk_id = k.id
			LEFT JOIN open_incidents oi ON oi.kiosk_id = k.id
			LEFT JOIN pending_commands pc ON pc.kiosk_id = k.id
			WHERE k.tenant_id = $1
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
			if err := rows.Scan(&k.ID, &k.Name, &k.Code, &k.Status, &k.Location, &k.LastHeartbeatAt, &k.CreatedAt, &k.UpdatedAt, &k.AppVersion, &k.OSVersion, &k.BatteryPercent, &k.NetworkStrength, &k.MemoryUsagePercent, &k.StorageFreeMB, &k.OpenIncidents, &k.PendingCommands); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read kiosks"})
			}
			k.HealthStatus = computeKioskHealthStatus(k.Status, k.LastHeartbeatAt, k.BatteryPercent)
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
			Name     string  `json:"name"`
			Code     string  `json:"code"`
			Location *string `json:"location"`
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
			INSERT INTO kiosks (tenant_id, name, code, hmac_secret, location, status, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,'active',NOW(),NOW())
			RETURNING id, name, code, status, location, last_heartbeat_at, created_at, updated_at
		`, tenantID, body.Name, body.Code, hmacSecret, body.Location).Scan(
			&k.ID, &k.Name, &k.Code, &k.Status, &k.Location, &k.LastHeartbeatAt, &k.CreatedAt, &k.UpdatedAt,
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
			Name     *string `json:"name"`
			Status   *string `json:"status"`
			Location *string `json:"location"`
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
				location = COALESCE($3, location),
				updated_at = NOW()
			WHERE id = $4 AND tenant_id = $5
			RETURNING id, name, code, status, location, last_heartbeat_at, created_at, updated_at
		`, body.Name, body.Status, body.Location, kioskID, tenantID).Scan(
			&k.ID, &k.Name, &k.Code, &k.Status, &k.Location, &k.LastHeartbeatAt, &k.CreatedAt, &k.UpdatedAt,
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

		payload := map[string]interface{}{
			"manual_secret_rotation": true,
			"kiosk_code":             kioskCode,
		}
		_, _ = tx.Exec(ctx, `
			INSERT INTO kiosk_commands (id, tenant_id, kiosk_id, command_type, payload, requested_by, status, completed_at)
			VALUES ($1, $2, $3, 'reprovision', $4, NULL, 'completed', NOW())
		`, uuid.New(), tenantID, kioskID, payload)

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

// GetKioskHistory returns real kiosk telemetry history for the last 7 days.
func GetKioskHistory(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant ID not found"})
		}

		kioskID := c.Params("id")
		if kioskID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Kiosk ID is required"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var exists bool
		if err := db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM kiosks
				WHERE id = $1 AND tenant_id = $2
			)
		`, kioskID, tenantID).Scan(&exists); err != nil || !exists {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kiosk not found"})
		}

		rows, err := db.Query(ctx, `
			WITH days AS (
				SELECT generate_series(CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, INTERVAL '1 day')::date AS day
			),
			samples AS (
				SELECT
					recorded_at::date AS day,
					COUNT(*) AS sample_count,
					COUNT(*) FILTER (WHERE status = 'active') AS active_samples,
					MAX(recorded_at) AS last_seen_at
				FROM kiosk_telemetry_samples
				WHERE tenant_id = $1
				  AND kiosk_id = $2
				  AND recorded_at::date >= CURRENT_DATE - INTERVAL '6 days'
				GROUP BY recorded_at::date
			),
			incidents AS (
				SELECT
					detected_at::date AS day,
					COUNT(*) AS incident_count
				FROM kiosk_incidents
				WHERE tenant_id = $1
				  AND kiosk_id = $2
				  AND detected_at::date >= CURRENT_DATE - INTERVAL '6 days'
				GROUP BY detected_at::date
			),
			activity AS (
				SELECT
					al.punch_time::date AS day,
					COUNT(*) AS activity_count,
					COUNT(*) FILTER (WHERE al.anomaly_detected = true) AS anomalies,
					MAX(al.punch_time) AS last_activity_at
				FROM attendance_logs al
				WHERE al.tenant_id = $1
				  AND al.kiosk_id = $2
				  AND al.punch_time::date >= CURRENT_DATE - INTERVAL '6 days'
				GROUP BY al.punch_time::date
			)
			SELECT
				d.day,
				COALESCE(a.activity_count, 0),
				COALESCE(a.anomalies, 0),
				a.last_activity_at,
				COALESCE(i.incident_count, 0),
				COALESCE(
					ROUND(
						CASE WHEN s.sample_count = 0 THEN 0
						ELSE s.active_samples * 100.0 / s.sample_count
						END, 2
					), 0
				) AS uptime_percent,
				s.last_seen_at
			FROM days d
			LEFT JOIN activity a ON a.day = d.day
			LEFT JOIN incidents i ON i.day = d.day
			LEFT JOIN samples s ON s.day = d.day
			ORDER BY d.day ASC
		`, tenantID, kioskID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load kiosk history"})
		}
		defer rows.Close()

		history := make([]fiber.Map, 0, 7)
		for rows.Next() {
			var day time.Time
			var activityCount int
			var anomalies int
			var lastActivityAt *time.Time
			var incidentCount int
			var uptimePercent float64
			var lastSeenAt *time.Time
			if err := rows.Scan(&day, &activityCount, &anomalies, &lastActivityAt, &incidentCount, &uptimePercent, &lastSeenAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read kiosk history"})
			}

			var lastActivityStr *string
			if lastActivityAt != nil {
				value := lastActivityAt.UTC().Format(time.RFC3339)
				lastActivityStr = &value
			}
			var lastSeenStr *string
			if lastSeenAt != nil {
				value := lastSeenAt.UTC().Format(time.RFC3339)
				lastSeenStr = &value
			}

			history = append(history, fiber.Map{
				"date":             day.Format("2006-01-02"),
				"activity_count":   activityCount,
				"anomalies":        anomalies,
				"incident_count":   incidentCount,
				"uptime_percent":   uptimePercent,
				"last_seen_at":     lastSeenStr,
				"last_activity_at": lastActivityStr,
			})
		}
		if err := rows.Err(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize kiosk history"})
		}

		return c.JSON(history)
	}
}
