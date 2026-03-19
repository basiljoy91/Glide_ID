package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	errWorkflowRequestNotFound = errors.New("workflow request not found")
)

func reviewLeaveRequestTx(ctx context.Context, tx pgx.Tx, tenantID, actorUserID, requestID, status string, reviewNote *string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE leave_requests
		SET status = $1, reviewed_by = $2, reviewed_at = NOW(), review_note = $3, updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5
	`, status, actorUserID, nullableString(reviewNote), requestID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to review leave request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errWorkflowRequestNotFound
	}
	return nil
}

func reviewRegularizationRequestTx(ctx context.Context, tx pgx.Tx, tenantID, actorUserID, requestID, status string, reviewNote *string) error {
	var userID string
	var attendanceLogID *uuid.UUID
	var requestedStatus string
	var requestedPunchTime time.Time
	if err := tx.QueryRow(ctx, `
		SELECT rr.user_id, rr.attendance_log_id, rr.requested_status::text, rr.requested_punch_time
		FROM attendance_regularization_requests rr
		WHERE rr.id = $1 AND rr.tenant_id = $2
	`, requestID, tenantID).Scan(&userID, &attendanceLogID, &requestedStatus, &requestedPunchTime); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errWorkflowRequestNotFound
		}
		return fmt.Errorf("failed to load regularization request: %w", err)
	}

	if status == "approved" {
		if attendanceLogID != nil {
			if _, err := tx.Exec(ctx, `
				UPDATE attendance_logs
				SET status = $1, punch_time = $2, verification_method = 'manual', updated_at = NOW()
				WHERE id = $3 AND tenant_id = $4
			`, requestedStatus, requestedPunchTime, *attendanceLogID, tenantID); err != nil {
				return fmt.Errorf("failed to apply regularization to attendance log: %w", err)
			}
		} else {
			if _, err := tx.Exec(ctx, `
				INSERT INTO attendance_logs (id, tenant_id, user_id, status, punch_time, verification_method, notes)
				VALUES ($1, $2, $3, $4, $5, 'manual', $6)
			`, uuid.New(), tenantID, userID, requestedStatus, requestedPunchTime, "Created from approved regularization request"); err != nil {
				return fmt.Errorf("failed to create corrected attendance log: %w", err)
			}
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE attendance_regularization_requests
		SET status = $1, reviewed_by = $2, reviewed_at = NOW(), review_note = $3, updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5
	`, status, actorUserID, nullableString(reviewNote), requestID, tenantID); err != nil {
		return fmt.Errorf("failed to review regularization request: %w", err)
	}

	return nil
}

func reviewOvertimeRequestTx(ctx context.Context, tx pgx.Tx, tenantID, actorUserID, requestID, status string, approvedMinutes int, reviewNote *string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE overtime_requests
		SET status = $1, approved_minutes = $2, reviewed_by = $3, reviewed_at = NOW(), review_note = $4, updated_at = NOW()
		WHERE id = $5 AND tenant_id = $6
	`, status, approvedMinutes, actorUserID, nullableString(reviewNote), requestID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to review overtime request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errWorkflowRequestNotFound
	}
	return nil
}
