package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type kioskTelemetryPayload struct {
	AppVersion         *string                `json:"app_version"`
	OSVersion          *string                `json:"os_version"`
	BatteryPercent     *int                   `json:"battery_percent"`
	NetworkStrength    *int                   `json:"network_strength"`
	StorageFreeMB      *int                   `json:"storage_free_mb"`
	StorageTotalMB     *int                   `json:"storage_total_mb"`
	MemoryUsagePercent *int                   `json:"memory_usage_percent"`
	Metadata           map[string]interface{} `json:"metadata"`
}

type kioskDashboardSummary struct {
	TotalKiosks      int `json:"total_kiosks"`
	HealthyKiosks    int `json:"healthy_kiosks"`
	StaleKiosks      int `json:"stale_kiosks"`
	OpenIncidents    int `json:"open_incidents"`
	PendingCommands  int `json:"pending_commands"`
	DeliveredCommand int `json:"delivered_commands"`
}

type kioskLocationMetric struct {
	Location         string  `json:"location"`
	Kiosks           int     `json:"kiosks"`
	HealthyKiosks    int     `json:"healthy_kiosks"`
	StaleKiosks      int     `json:"stale_kiosks"`
	ActivityCount    int     `json:"activity_count"`
	Anomalies        int     `json:"anomalies"`
	OpenIncidents    int     `json:"open_incidents"`
	AverageUptimePct float64 `json:"average_uptime_pct"`
	LastActivityAt   *string `json:"last_activity_at,omitempty"`
	LastHeartbeatAt  *string `json:"last_heartbeat_at,omitempty"`
}

type kioskIncidentRow struct {
	ID           uuid.UUID  `json:"id"`
	IncidentType string     `json:"incident_type"`
	Severity     string     `json:"severity"`
	Status       string     `json:"status"`
	Title        string     `json:"title"`
	Details      *string    `json:"details"`
	DetectedAt   time.Time  `json:"detected_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

type kioskCommandRow struct {
	ID          uuid.UUID              `json:"id"`
	CommandType string                 `json:"command_type"`
	Status      string                 `json:"status"`
	Payload     map[string]interface{} `json:"payload"`
	RequestedAt time.Time              `json:"requested_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	LastError   *string                `json:"last_error,omitempty"`
}

func computeKioskHealthStatus(status string, lastHeartbeatAt *time.Time, batteryPercent *int) string {
	if status != "active" {
		return status
	}
	if lastHeartbeatAt == nil || time.Since(lastHeartbeatAt.UTC()) > 10*time.Minute {
		return "stale"
	}
	if batteryPercent != nil && *batteryPercent <= 20 {
		return "warning"
	}
	return "healthy"
}

func upsertKioskIncident(ctx context.Context, db *pgxpool.Pool, tenantID, kioskID, incidentType, severity, title string, details *string, shouldBeOpen bool) error {
	if shouldBeOpen {
		var existingID uuid.UUID
		err := db.QueryRow(ctx, `
			SELECT id
			FROM kiosk_incidents
			WHERE tenant_id = $1 AND kiosk_id = $2 AND incident_type = $3 AND status IN ('open', 'acknowledged')
			ORDER BY detected_at DESC
			LIMIT 1
		`, tenantID, kioskID, incidentType).Scan(&existingID)
		switch {
		case err == nil:
			_, err = db.Exec(ctx, `
				UPDATE kiosk_incidents
				SET severity = $1, title = $2, details = $3, updated_at = NOW()
				WHERE id = $4
			`, severity, title, details, existingID)
			return err
		case errors.Is(err, pgx.ErrNoRows):
			_, err = db.Exec(ctx, `
				INSERT INTO kiosk_incidents (id, tenant_id, kiosk_id, incident_type, severity, status, title, details)
				VALUES ($1, $2, $3, $4, $5, 'open', $6, $7)
			`, uuid.New(), tenantID, kioskID, incidentType, severity, title, details)
			return err
		default:
			return err
		}
	}

	_, err := db.Exec(ctx, `
		UPDATE kiosk_incidents
		SET status = 'resolved', resolved_at = NOW(), updated_at = NOW()
		WHERE tenant_id = $1 AND kiosk_id = $2 AND incident_type = $3 AND status IN ('open', 'acknowledged')
	`, tenantID, kioskID, incidentType)
	return err
}

func persistKioskTelemetry(ctx context.Context, db *pgxpool.Pool, tenantID, kioskID, kioskStatus string, payload kioskTelemetryPayload) error {
	if payload.Metadata == nil {
		payload.Metadata = map[string]interface{}{}
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO kiosk_telemetry_samples (
			id, tenant_id, kiosk_id, status, app_version, os_version,
			battery_percent, network_strength, storage_free_mb, storage_total_mb,
			memory_usage_percent, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, uuid.New(), tenantID, kioskID, kioskStatus, nullableString(payload.AppVersion), nullableString(payload.OSVersion), payload.BatteryPercent, payload.NetworkStrength, payload.StorageFreeMB, payload.StorageTotalMB, payload.MemoryUsagePercent, payload.Metadata); err != nil {
		return err
	}

	lowBattery := payload.BatteryPercent != nil && *payload.BatteryPercent <= 20
	batteryDetails := ""
	if payload.BatteryPercent != nil {
		batteryDetails = fmt.Sprintf("Battery at %d%%", *payload.BatteryPercent)
	}
	if err := upsertKioskIncident(ctx, db, tenantID, kioskID, "low_battery", "warning", "Battery below threshold", nullableString(&batteryDetails), lowBattery); err != nil {
		return err
	}

	lowStorage := payload.StorageFreeMB != nil && *payload.StorageFreeMB < 1024
	storageDetails := ""
	if payload.StorageFreeMB != nil {
		storageDetails = fmt.Sprintf("%d MB free storage remaining", *payload.StorageFreeMB)
	}
	if err := upsertKioskIncident(ctx, db, tenantID, kioskID, "low_storage", "warning", "Available storage is low", nullableString(&storageDetails), lowStorage); err != nil {
		return err
	}

	highMemory := payload.MemoryUsagePercent != nil && *payload.MemoryUsagePercent >= 95
	memoryDetails := ""
	if payload.MemoryUsagePercent != nil {
		memoryDetails = fmt.Sprintf("Memory usage at %d%%", *payload.MemoryUsagePercent)
	}
	return upsertKioskIncident(ctx, db, tenantID, kioskID, "high_memory", "warning", "Memory pressure detected", nullableString(&memoryDetails), highMemory)
}

func deliverQueuedKioskCommands(ctx context.Context, db *pgxpool.Pool, tenantID, kioskID string) ([]kioskCommandRow, error) {
	rows, err := db.Query(ctx, `
		SELECT id, command_type, status, COALESCE(payload, '{}'::jsonb), requested_at, completed_at, last_error
		FROM kiosk_commands
		WHERE tenant_id = $1 AND kiosk_id = $2 AND status = 'queued'
		ORDER BY requested_at ASC
		LIMIT 5
	`, tenantID, kioskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []kioskCommandRow{}
	commandIDs := []uuid.UUID{}
	for rows.Next() {
		var row kioskCommandRow
		if err := rows.Scan(&row.ID, &row.CommandType, &row.Status, &row.Payload, &row.RequestedAt, &row.CompletedAt, &row.LastError); err != nil {
			return nil, err
		}
		result = append(result, row)
		commandIDs = append(commandIDs, row.ID)
	}
	if len(commandIDs) > 0 {
		if _, err := db.Exec(ctx, `
			UPDATE kiosk_commands
			SET status = 'delivered'
			WHERE tenant_id = $1 AND kiosk_id = $2 AND id = ANY($3::uuid[])
		`, tenantID, kioskID, commandIDs); err != nil {
			return nil, err
		}
		for i := range result {
			result[i].Status = "delivered"
		}
	}
	return result, rows.Err()
}

func GetKioskFleetDashboard(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var summary kioskDashboardSummary
		if err := db.QueryRow(ctx, `
			WITH latest AS (
				SELECT DISTINCT ON (kiosk_id) kiosk_id, battery_percent, recorded_at
				FROM kiosk_telemetry_samples
				WHERE tenant_id = $1
				ORDER BY kiosk_id, recorded_at DESC
			)
			SELECT
				COUNT(*) AS total_kiosks,
				COUNT(*) FILTER (
					WHERE k.status = 'active'
					  AND k.last_heartbeat_at IS NOT NULL
					  AND k.last_heartbeat_at >= NOW() - INTERVAL '10 minutes'
					  AND (l.battery_percent IS NULL OR l.battery_percent > 20)
				) AS healthy_kiosks,
				COUNT(*) FILTER (
					WHERE k.status = 'active'
					  AND (k.last_heartbeat_at IS NULL OR k.last_heartbeat_at < NOW() - INTERVAL '10 minutes')
				) AS stale_kiosks,
				COALESCE((SELECT COUNT(*) FROM kiosk_incidents WHERE tenant_id = $1 AND status IN ('open', 'acknowledged')), 0) AS open_incidents,
				COALESCE((SELECT COUNT(*) FROM kiosk_commands WHERE tenant_id = $1 AND status = 'queued'), 0) AS pending_commands,
				COALESCE((SELECT COUNT(*) FROM kiosk_commands WHERE tenant_id = $1 AND status = 'delivered'), 0) AS delivered_commands
			FROM kiosks k
			LEFT JOIN latest l ON l.kiosk_id = k.id
			WHERE k.tenant_id = $1
		`, tenantID).Scan(&summary.TotalKiosks, &summary.HealthyKiosks, &summary.StaleKiosks, &summary.OpenIncidents, &summary.PendingCommands, &summary.DeliveredCommand); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load kiosk fleet summary"})
		}

		locationRows, err := db.Query(ctx, `
			WITH latest AS (
				SELECT DISTINCT ON (kiosk_id) kiosk_id, battery_percent
				FROM kiosk_telemetry_samples
				WHERE tenant_id = $1
				ORDER BY kiosk_id, recorded_at DESC
			),
			uptime AS (
				SELECT
					kiosk_id,
					ROUND(
						CASE WHEN COUNT(*) = 0 THEN 0
						ELSE COUNT(*) FILTER (WHERE status = 'active') * 100.0 / COUNT(*)
						END, 2
					) AS uptime_pct,
					MAX(recorded_at) AS last_telemetry_at
				FROM kiosk_telemetry_samples
				WHERE tenant_id = $1
				  AND recorded_at >= NOW() - INTERVAL '7 days'
				GROUP BY kiosk_id
			),
			activity AS (
				SELECT kiosk_id, COUNT(*) AS activity_count, COUNT(*) FILTER (WHERE anomaly_detected = true) AS anomalies, MAX(punch_time) AS last_activity_at
				FROM attendance_logs
				WHERE tenant_id = $1
				  AND kiosk_id IS NOT NULL
				  AND punch_time >= NOW() - INTERVAL '30 days'
				GROUP BY kiosk_id
			),
			open_incidents AS (
				SELECT kiosk_id, COUNT(*) AS open_incidents
				FROM kiosk_incidents
				WHERE tenant_id = $1 AND status IN ('open', 'acknowledged')
				GROUP BY kiosk_id
			)
			SELECT
				COALESCE(k.location, 'Unassigned') AS location,
				COUNT(*) AS kiosks,
				COUNT(*) FILTER (
					WHERE k.status = 'active'
					  AND k.last_heartbeat_at IS NOT NULL
					  AND k.last_heartbeat_at >= NOW() - INTERVAL '10 minutes'
					  AND (l.battery_percent IS NULL OR l.battery_percent > 20)
				) AS healthy_kiosks,
				COUNT(*) FILTER (
					WHERE k.status = 'active'
					  AND (k.last_heartbeat_at IS NULL OR k.last_heartbeat_at < NOW() - INTERVAL '10 minutes')
				) AS stale_kiosks,
				COALESCE(SUM(a.activity_count), 0) AS activity_count,
				COALESCE(SUM(a.anomalies), 0) AS anomalies,
				COALESCE(SUM(oi.open_incidents), 0) AS open_incidents,
				COALESCE(AVG(u.uptime_pct), 0) AS average_uptime_pct,
				MAX(a.last_activity_at) AS last_activity_at,
				MAX(k.last_heartbeat_at) AS last_heartbeat_at
			FROM kiosks k
			LEFT JOIN latest l ON l.kiosk_id = k.id
			LEFT JOIN uptime u ON u.kiosk_id = k.id
			LEFT JOIN activity a ON a.kiosk_id = k.id
			LEFT JOIN open_incidents oi ON oi.kiosk_id = k.id
			WHERE k.tenant_id = $1
			GROUP BY COALESCE(k.location, 'Unassigned')
			ORDER BY location ASC
		`, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load location metrics"})
		}
		defer locationRows.Close()

		locations := []kioskLocationMetric{}
		for locationRows.Next() {
			var metric kioskLocationMetric
			var lastActivityAt *time.Time
			var lastHeartbeatAt *time.Time
			if err := locationRows.Scan(&metric.Location, &metric.Kiosks, &metric.HealthyKiosks, &metric.StaleKiosks, &metric.ActivityCount, &metric.Anomalies, &metric.OpenIncidents, &metric.AverageUptimePct, &lastActivityAt, &lastHeartbeatAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read location metrics"})
			}
			if lastActivityAt != nil {
				value := lastActivityAt.UTC().Format(time.RFC3339)
				metric.LastActivityAt = &value
			}
			if lastHeartbeatAt != nil {
				value := lastHeartbeatAt.UTC().Format(time.RFC3339)
				metric.LastHeartbeatAt = &value
			}
			locations = append(locations, metric)
		}

		return c.JSON(fiber.Map{
			"summary":   summary,
			"locations": locations,
		})
	}
}

func ListKioskIncidents(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		kioskID := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT id, incident_type, severity, status, title, details, detected_at, resolved_at
			FROM kiosk_incidents
			WHERE tenant_id = $1 AND kiosk_id = $2
			ORDER BY detected_at DESC
			LIMIT 25
		`, tenantID, kioskID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list kiosk incidents"})
		}
		defer rows.Close()

		out := []kioskIncidentRow{}
		for rows.Next() {
			var row kioskIncidentRow
			if err := rows.Scan(&row.ID, &row.IncidentType, &row.Severity, &row.Status, &row.Title, &row.Details, &row.DetectedAt, &row.ResolvedAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read kiosk incidents"})
			}
			out = append(out, row)
		}
		return c.JSON(out)
	}
}

func UpdateKioskIncident(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		kioskID := c.Params("id")
		incidentID := c.Params("incidentId")
		var body struct {
			Status string `json:"status"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		status := strings.ToLower(strings.TrimSpace(body.Status))
		if status != "acknowledged" && status != "resolved" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be acknowledged or resolved"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var query string
		var args []interface{}
		if status == "acknowledged" {
			query = `
				UPDATE kiosk_incidents
				SET status = 'acknowledged', acknowledged_at = NOW(), acknowledged_by = $1, updated_at = NOW()
				WHERE tenant_id = $2 AND kiosk_id = $3 AND id = $4
			`
			args = []interface{}{actorUserID, tenantID, kioskID, incidentID}
		} else {
			query = `
				UPDATE kiosk_incidents
				SET status = 'resolved', resolved_at = NOW(), resolved_by = $1, updated_at = NOW()
				WHERE tenant_id = $2 AND kiosk_id = $3 AND id = $4
			`
			args = []interface{}{actorUserID, tenantID, kioskID, incidentID}
		}
		tag, err := db.Exec(ctx, query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update kiosk incident"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kiosk incident not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func ListKioskCommands(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		kioskID := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT id, command_type, status, COALESCE(payload, '{}'::jsonb), requested_at, completed_at, last_error
			FROM kiosk_commands
			WHERE tenant_id = $1 AND kiosk_id = $2
			ORDER BY requested_at DESC
			LIMIT 25
		`, tenantID, kioskID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list kiosk commands"})
		}
		defer rows.Close()

		out := []kioskCommandRow{}
		for rows.Next() {
			var row kioskCommandRow
			if err := rows.Scan(&row.ID, &row.CommandType, &row.Status, &row.Payload, &row.RequestedAt, &row.CompletedAt, &row.LastError); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read kiosk commands"})
			}
			out = append(out, row)
		}
		return c.JSON(out)
	}
}

func QueueKioskCommand(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		kioskID := c.Params("id")
		var body struct {
			CommandType string                 `json:"command_type"`
			Note        *string                `json:"note"`
			Payload     map[string]interface{} `json:"payload"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		commandType := strings.ToLower(strings.TrimSpace(body.CommandType))
		switch commandType {
		case "lock", "disable", "enable", "reprovision":
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Unsupported kiosk command"})
		}
		if body.Payload == nil {
			body.Payload = map[string]interface{}{}
		}
		if body.Note != nil && strings.TrimSpace(*body.Note) != "" {
			body.Payload["note"] = strings.TrimSpace(*body.Note)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to queue kiosk command"})
		}
		defer tx.Rollback(ctx)

		switch commandType {
		case "disable":
			if _, err := tx.Exec(ctx, `UPDATE kiosks SET status = 'inactive', updated_at = NOW() WHERE tenant_id = $1 AND id = $2`, tenantID, kioskID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to disable kiosk"})
			}
		case "enable":
			if _, err := tx.Exec(ctx, `UPDATE kiosks SET status = 'active', updated_at = NOW() WHERE tenant_id = $1 AND id = $2`, tenantID, kioskID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to enable kiosk"})
			}
		}

		var command kioskCommandRow
		if err := tx.QueryRow(ctx, `
			INSERT INTO kiosk_commands (id, tenant_id, kiosk_id, command_type, payload, requested_by)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, command_type, status, payload, requested_at, completed_at, last_error
		`, uuid.New(), tenantID, kioskID, commandType, body.Payload, actorUserID).Scan(
			&command.ID, &command.CommandType, &command.Status, &command.Payload, &command.RequestedAt, &command.CompletedAt, &command.LastError,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save kiosk command"})
		}
		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to finalize kiosk command"})
		}
		return c.Status(fiber.StatusCreated).JSON(command)
	}
}
