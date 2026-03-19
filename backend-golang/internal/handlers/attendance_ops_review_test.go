package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestReviewLeaveRequestTx_ApprovesRequest(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-approval"
	actorUserID := "reviewer-1"
	requestID := "leave-1"
	note := "approved"

	tx := &fakeTx{
		t: t,
		execQueue: []execExpectation{
			{
				sqlContains: "UPDATE leave_requests",
				args:        []any{"approved", actorUserID, &note, requestID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := reviewLeaveRequestTx(context.Background(), tx, tenantID, actorUserID, requestID, "approved", &note); err != nil {
		t.Fatalf("reviewLeaveRequestTx returned error: %v", err)
	}
	tx.assertDone()
}

func TestReviewRegularizationRequestTx_ApprovesAndUpdatesAttendanceLog(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-approval"
	actorUserID := "reviewer-2"
	requestID := "regularization-1"
	userID := "user-1"
	attendanceLogID := uuid.New()
	requestedPunchTime := time.Date(2026, 3, 19, 9, 30, 0, 0, time.UTC)
	note := "fixed punch"

	tx := &fakeTx{
		t: t,
		queryRowQueue: []queryRowExpectation{
			{
				sqlContains: "FROM attendance_regularization_requests rr",
				args:        []any{requestID, tenantID},
				scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = userID
					*(dest[1].(**uuid.UUID)) = &attendanceLogID
					*(dest[2].(*string)) = "check_in"
					*(dest[3].(*time.Time)) = requestedPunchTime
					return nil
				},
			},
		},
		execQueue: []execExpectation{
			{
				sqlContains: "UPDATE attendance_logs",
				args:        []any{"check_in", requestedPunchTime, attendanceLogID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
			{
				sqlContains: "UPDATE attendance_regularization_requests",
				args:        []any{"approved", actorUserID, &note, requestID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := reviewRegularizationRequestTx(context.Background(), tx, tenantID, actorUserID, requestID, "approved", &note); err != nil {
		t.Fatalf("reviewRegularizationRequestTx returned error: %v", err)
	}
	tx.assertDone()
}

func TestReviewOvertimeRequestTx_ApprovesMinutes(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-approval"
	actorUserID := "reviewer-3"
	requestID := "overtime-1"
	note := "approved for payroll"

	tx := &fakeTx{
		t: t,
		execQueue: []execExpectation{
			{
				sqlContains: "UPDATE overtime_requests",
				args:        []any{"approved", 120, actorUserID, &note, requestID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := reviewOvertimeRequestTx(context.Background(), tx, tenantID, actorUserID, requestID, "approved", 120, &note); err != nil {
		t.Fatalf("reviewOvertimeRequestTx returned error: %v", err)
	}
	tx.assertDone()
}

func TestMutateBulkEditBatchStatusTx_RollsBackAppliedState(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-bulk"
	batchID := uuid.New().String()
	userID := uuid.New()
	beforeState, _ := json.Marshal(map[string]any{
		"department_id":   "",
		"manager_id":      "",
		"employment_type": "contract",
		"work_location":   "HQ",
		"cost_center":     "CC-100",
		"designation":     "Analyst",
		"is_active":       false,
	})
	afterState, _ := json.Marshal(map[string]any{
		"department_id":   "",
		"manager_id":      "",
		"employment_type": "full_time",
		"work_location":   "Remote",
		"cost_center":     "CC-200",
		"designation":     "Lead Analyst",
		"is_active":       true,
	})

	tx := &fakeTx{
		t: t,
		queryRowQueue: []queryRowExpectation{
			{
				sqlContains: "SELECT status FROM bulk_change_batches",
				args:        []any{batchID, tenantID},
				scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = "applied"
					return nil
				},
			},
		},
		queryQueue: []queryExpectation{
			{
				sqlContains: "SELECT user_id, before_state, after_state FROM bulk_change_batch_items",
				args:        []any{batchID},
				rows: [][]any{
					{userID, beforeState, afterState},
				},
			},
		},
		execQueue: []execExpectation{
			{
				sqlContains: "UPDATE users",
				args:        []any{(*uuid.UUID)(nil), (*uuid.UUID)(nil), "contract", "HQ", "CC-100", "Analyst", false, tenantID, userID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
			{
				sqlContains: "UPDATE bulk_change_batches SET status = 'rolled_back'",
				args:        []any{batchID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := mutateBulkEditBatchStatusTx(context.Background(), tx, tenantID, batchID, "applied", false); err != nil {
		t.Fatalf("mutateBulkEditBatchStatusTx returned error: %v", err)
	}
	tx.assertDone()
}

func TestMutateBulkEditBatchStatusTx_ReturnsNotFound(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-bulk"
	batchID := uuid.New().String()
	tx := &fakeTx{
		t: t,
		queryRowQueue: []queryRowExpectation{
			{
				sqlContains: "SELECT status FROM bulk_change_batches",
				args:        []any{batchID, tenantID},
				scanFn:      errRow(pgx.ErrNoRows),
			},
		},
	}

	if err := mutateBulkEditBatchStatusTx(context.Background(), tx, tenantID, batchID, "applied", false); err != errBulkEditBatchNotFound {
		t.Fatalf("expected errBulkEditBatchNotFound, got %v", err)
	}
	tx.assertDone()
}
