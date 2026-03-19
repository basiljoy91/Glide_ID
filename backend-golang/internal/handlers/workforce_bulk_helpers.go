package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	errBulkEditBatchNotFound    = errors.New("bulk edit batch not found")
	errBulkEditBatchInvalidFlow = errors.New("bulk edit batch invalid state transition")
)

func mutateBulkEditBatchStatusTx(ctx context.Context, tx pgx.Tx, tenantID, batchID, fromStatus string, apply bool) error {
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM bulk_change_batches WHERE id = $1 AND tenant_id = $2`, batchID, tenantID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errBulkEditBatchNotFound
		}
		return fmt.Errorf("failed to load bulk edit batch: %w", err)
	}
	if status != fromStatus {
		return errBulkEditBatchInvalidFlow
	}

	rows, err := tx.Query(ctx, `SELECT user_id, before_state, after_state FROM bulk_change_batch_items WHERE batch_id = $1 ORDER BY id`, batchID)
	if err != nil {
		return fmt.Errorf("failed to load bulk edit items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		var beforeRaw []byte
		var afterRaw []byte
		if err := rows.Scan(&userID, &beforeRaw, &afterRaw); err != nil {
			return fmt.Errorf("failed to read bulk edit items: %w", err)
		}

		state := map[string]any{}
		if apply {
			_ = json.Unmarshal(afterRaw, &state)
		} else {
			_ = json.Unmarshal(beforeRaw, &state)
		}
		if err := applyBulkChangeState(ctx, tx, tenantID, userID, state); err != nil {
			return fmt.Errorf("failed to apply bulk edit state: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to finalize bulk edit item scan: %w", err)
	}

	if apply {
		_, err = tx.Exec(ctx, `UPDATE bulk_change_batches SET status = 'applied', applied_at = NOW() WHERE id = $1 AND tenant_id = $2`, batchID, tenantID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE bulk_change_batches SET status = 'rolled_back', rolled_back_at = NOW() WHERE id = $1 AND tenant_id = $2`, batchID, tenantID)
	}
	if err != nil {
		return fmt.Errorf("failed to update bulk edit batch status: %w", err)
	}

	return nil
}
