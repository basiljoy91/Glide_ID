package handlers

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSetOrganizationStatusTx_DeactivateRevokesSessionsAndCancelsBilling(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New().String()
	tx := &fakeTx{
		t: t,
		execQueue: []execExpectation{
			{
				sqlContains: "UPDATE tenants SET deleted_at = NOW()",
				args:        []any{tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
			{
				sqlContains: "UPDATE auth_sessions",
				args:        []any{tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 2"),
			},
			{
				sqlContains: "UPDATE billing_subscriptions",
				args:        []any{tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := setOrganizationStatusTx(context.Background(), tx, tenantID, false); err != nil {
		t.Fatalf("setOrganizationStatusTx returned error: %v", err)
	}
	tx.assertDone()
}

func TestSetOrganizationStatusTx_ActivateRestoresTenantAndBilling(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New().String()
	tx := &fakeTx{
		t: t,
		execQueue: []execExpectation{
			{
				sqlContains: "UPDATE tenants SET deleted_at = NULL",
				args:        []any{tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
			{
				sqlContains: "UPDATE billing_subscriptions",
				args:        []any{tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := setOrganizationStatusTx(context.Background(), tx, tenantID, true); err != nil {
		t.Fatalf("setOrganizationStatusTx returned error: %v", err)
	}
	tx.assertDone()
}

func TestSetOrganizationStatusTx_ReturnsNotFoundWhenTenantMissing(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New().String()
	tx := &fakeTx{
		t: t,
		execQueue: []execExpectation{
			{
				sqlContains: "UPDATE tenants SET deleted_at = NOW()",
				args:        []any{tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 0"),
			},
		},
	}

	if err := setOrganizationStatusTx(context.Background(), tx, tenantID, false); err != errOrganizationNotFound {
		t.Fatalf("expected errOrganizationNotFound, got %v", err)
	}
	tx.assertDone()
}

func TestLoadBillableSubscription_ReturnsOrganizationNotFoundForInactiveTenant(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New().String()
	tx := &fakeTx{
		t: t,
		queryRowQueue: []queryRowExpectation{
			{
				sqlContains: "FROM tenants t",
				args:        []any{tenantID},
				scanFn:      errRow(pgx.ErrNoRows),
			},
			{
				sqlContains: "SELECT EXISTS(SELECT 1 FROM tenants",
				args:        []any{tenantID},
				scanFn:      boolRow(false),
			},
		},
	}

	var subscriptionID *uuid.UUID
	var seatCount, baseAmount, perSeatAmount int
	err := loadBillableSubscription(context.Background(), tx, tenantID, &subscriptionID, &seatCount, &baseAmount, &perSeatAmount)
	if err != errOrganizationNotFound {
		t.Fatalf("expected errOrganizationNotFound, got %v", err)
	}
	tx.assertDone()
}

func TestLoadBillableSubscription_ReturnsNoRowsWhenTenantHasNoActiveSubscription(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New().String()
	tx := &fakeTx{
		t: t,
		queryRowQueue: []queryRowExpectation{
			{
				sqlContains: "FROM tenants t",
				args:        []any{tenantID},
				scanFn:      errRow(pgx.ErrNoRows),
			},
			{
				sqlContains: "SELECT EXISTS(SELECT 1 FROM tenants",
				args:        []any{tenantID},
				scanFn:      boolRow(true),
			},
		},
	}

	var subscriptionID *uuid.UUID
	var seatCount, baseAmount, perSeatAmount int
	err := loadBillableSubscription(context.Background(), tx, tenantID, &subscriptionID, &seatCount, &baseAmount, &perSeatAmount)
	if err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
	tx.assertDone()
}
