package services

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeAuthRow struct {
	scanFn func(dest ...any) error
}

func (r fakeAuthRow) Scan(dest ...any) error {
	return r.scanFn(dest...)
}

type authQueryRowExpectation struct {
	sqlContains string
	args        []any
	scanFn      func(dest ...any) error
}

type fakeAuthSessionStore struct {
	t             *testing.T
	queryRowQueue []authQueryRowExpectation
}

func (f *fakeAuthSessionStore) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if len(f.queryRowQueue) == 0 {
		f.t.Fatalf("unexpected QueryRow: %s", sql)
	}
	exp := f.queryRowQueue[0]
	f.queryRowQueue = f.queryRowQueue[1:]
	if !strings.Contains(sql, exp.sqlContains) {
		f.t.Fatalf("QueryRow SQL mismatch\nwant contains: %s\ngot: %s", exp.sqlContains, sql)
	}
	if !reflect.DeepEqual(args, exp.args) {
		f.t.Fatalf("QueryRow args mismatch\nwant: %#v\ngot: %#v", exp.args, args)
	}
	return fakeAuthRow{scanFn: exp.scanFn}
}

func (f *fakeAuthSessionStore) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected Exec")
}

func authErrRow(err error) func(dest ...any) error {
	return func(dest ...any) error {
		return err
	}
}

func TestValidateActiveSession_ReturnsSessionIDForActiveTenantUser(t *testing.T) {
	t.Parallel()

	sessionID := uuid.New()
	store := &fakeAuthSessionStore{
		t: t,
		queryRowQueue: []authQueryRowExpectation{
			{
				sqlContains: "FROM auth_sessions s",
				args:        []any{"jti-1", "user-1", "tenant-1"},
				scanFn: func(dest ...any) error {
					*(dest[0].(*uuid.UUID)) = sessionID
					return nil
				},
			},
		},
	}

	got, err := validateActiveSession(context.Background(), store, "jti-1", "user-1", "tenant-1")
	if err != nil {
		t.Fatalf("validateActiveSession returned error: %v", err)
	}
	if got != sessionID {
		t.Fatalf("expected session %s, got %s", sessionID, got)
	}
}

func TestValidateActiveSession_ReturnsNoRowsForDeactivatedTenant(t *testing.T) {
	t.Parallel()

	store := &fakeAuthSessionStore{
		t: t,
		queryRowQueue: []authQueryRowExpectation{
			{
				sqlContains: "FROM auth_sessions s",
				args:        []any{"jti-2", "user-2", "tenant-2"},
				scanFn:      authErrRow(pgx.ErrNoRows),
			},
		},
	}

	_, err := validateActiveSession(context.Background(), store, "jti-2", "user-2", "tenant-2")
	if err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
}
