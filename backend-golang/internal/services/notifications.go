package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InsertNotification(ctx context.Context, db *pgxpool.Pool, tenantID string, userID *string, notificationType, title, body, severity string, actionURL *string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO org_notifications (tenant_id, user_id, notification_type, title, body, severity, action_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, userID, notificationType, title, body, severity, actionURL)
	return err
}
