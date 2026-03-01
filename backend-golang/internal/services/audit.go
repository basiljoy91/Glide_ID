package services

import (
	"context"
	"encoding/json"

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

// LogAction logs an audit action
func (s *AuditService) LogAction(ctx context.Context, log *models.AuditLog) error {
	log.ID = uuid.New()

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

