package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewConnection(databaseURL string) (*DB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30
	config.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

// SetTenantContext sets the tenant context for RLS policies
func (db *DB) SetTenantContext(ctx context.Context, tenantID string) error {
	_, err := db.Pool.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	return err
}

// SetUserContext sets the user context for RLS policies
func (db *DB) SetUserContext(ctx context.Context, userID string) error {
	_, err := db.Pool.Exec(ctx, "SET LOCAL app.current_user_id = $1", userID)
	return err
}

// SetAIServiceContext sets the AI service context to bypass RLS
func (db *DB) SetAIServiceContext(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, "SET LOCAL app.is_ai_service = 'true'")
	return err
}

