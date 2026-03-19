package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditService struct {
	db *pgxpool.Pool
}

func NewAuditService(db *pgxpool.Pool) *AuditService {
	return &AuditService{db: db}
}

type AuditLogEntry struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     *uuid.UUID             `json:"tenant_id"`
	UserID       *uuid.UUID             `json:"user_id"`
	ActorName    *string                `json:"actor_name,omitempty"`
	ActorEmail   *string                `json:"actor_email,omitempty"`
	TargetUserID *uuid.UUID             `json:"target_user_id"`
	TargetName   *string                `json:"target_name,omitempty"`
	TargetEmail  *string                `json:"target_email,omitempty"`
	Action       string                 `json:"action"`
	ResourceType *string                `json:"resource_type"`
	ResourceID   *uuid.UUID             `json:"resource_id"`
	Details      map[string]interface{} `json:"details"`
	IPAddress    *string                `json:"ip_address"`
	UserAgent    *string                `json:"user_agent"`
	CreatedAt    time.Time              `json:"created_at"`
}

type AuditLogFilters struct {
	Action string
	Query  string
	Limit  int
	Offset int
}

func (s *AuditService) ListLogs(ctx context.Context, tenantID string, filters AuditLogFilters) ([]AuditLogEntry, int, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	where := []string{"al.tenant_id = $1"}
	args := []interface{}{tenantID}

	if strings.TrimSpace(filters.Action) != "" {
		args = append(args, filters.Action)
		where = append(where, fmt.Sprintf("al.action = $%d", len(args)))
	}
	if q := strings.TrimSpace(filters.Query); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf(`(
			LOWER(COALESCE(actor.first_name || ' ' || actor.last_name, '')) LIKE $%d OR
			LOWER(COALESCE(actor.email, '')) LIKE $%d OR
			LOWER(COALESCE(target.first_name || ' ' || target.last_name, '')) LIKE $%d OR
			LOWER(COALESCE(target.email, '')) LIKE $%d OR
			LOWER(al.action::text) LIKE $%d OR
			LOWER(COALESCE(al.resource_type, '')) LIKE $%d
		)`, len(args), len(args), len(args), len(args), len(args), len(args)))
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM audit_logs al %s`, whereClause)
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(`
		SELECT
			al.id, al.tenant_id, al.user_id,
			CASE WHEN actor.id IS NULL THEN NULL ELSE actor.first_name || ' ' || actor.last_name END AS actor_name,
			actor.email,
			al.target_user_id,
			CASE WHEN target.id IS NULL THEN NULL ELSE target.first_name || ' ' || target.last_name END AS target_name,
			target.email,
			al.action, al.resource_type, al.resource_id, al.details, al.ip_address::text, al.user_agent, al.created_at
		FROM audit_logs al
		LEFT JOIN users actor ON actor.id = al.user_id
		LEFT JOIN users target ON target.id = al.target_user_id
		%s
		ORDER BY al.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)

	rows, err := s.db.Query(ctx, listQuery, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]AuditLogEntry, 0, limit)
	for rows.Next() {
		var entry AuditLogEntry
		var detailsRaw []byte
		if err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.UserID, &entry.ActorName, &entry.ActorEmail,
			&entry.TargetUserID, &entry.TargetName, &entry.TargetEmail,
			&entry.Action, &entry.ResourceType, &entry.ResourceID, &detailsRaw,
			&entry.IPAddress, &entry.UserAgent, &entry.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(detailsRaw) > 0 {
			_ = json.Unmarshal(detailsRaw, &entry.Details)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

// LogAction logs an audit action
func (s *AuditService) LogAction(ctx context.Context, log *models.AuditLog) error {
	log.ID = uuid.New()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	detailsJSON, _ := json.Marshal(log.Details)

	_, err := s.db.Exec(ctx, `
		INSERT INTO audit_logs (
			id, tenant_id, user_id, target_user_id, action, resource_type,
			resource_id, details, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, log.ID, log.TenantID, log.UserID, log.TargetUserID, log.Action,
		log.ResourceType, log.ResourceID, detailsJSON, log.IPAddress, log.UserAgent, log.CreatedAt)

	return err
}
