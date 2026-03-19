package handlers

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	return r.scanFn(dest...)
}

type queryRowExpectation struct {
	sqlContains string
	args        []any
	scanFn      func(dest ...any) error
}

type execExpectation struct {
	sqlContains string
	args        []any
	tag         pgconn.CommandTag
	err         error
}

type fakeTx struct {
	t                 *testing.T
	queryRowQueue     []queryRowExpectation
	execQueue         []execExpectation
	queryRowCallCount int
	execCallCount     int
}

func (f *fakeTx) Begin(context.Context) (pgx.Tx, error) { panic("unexpected Begin") }
func (f *fakeTx) Commit(context.Context) error          { return nil }
func (f *fakeTx) Rollback(context.Context) error        { return nil }
func (f *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	panic("unexpected CopyFrom")
}
func (f *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	panic("unexpected SendBatch")
}
func (f *fakeTx) LargeObjects() pgx.LargeObjects { panic("unexpected LargeObjects") }
func (f *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	panic("unexpected Prepare")
}
func (f *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) { panic("unexpected Query") }
func (f *fakeTx) Conn() *pgx.Conn                                         { return nil }

func (f *fakeTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if len(f.queryRowQueue) == 0 {
		f.t.Fatalf("unexpected QueryRow: %s", sql)
	}
	exp := f.queryRowQueue[0]
	f.queryRowQueue = f.queryRowQueue[1:]
	f.queryRowCallCount++
	if !strings.Contains(sql, exp.sqlContains) {
		f.t.Fatalf("QueryRow SQL mismatch\nwant contains: %s\ngot: %s", exp.sqlContains, sql)
	}
	if !reflect.DeepEqual(args, exp.args) {
		f.t.Fatalf("QueryRow args mismatch\nwant: %#v\ngot: %#v", exp.args, args)
	}
	return fakeRow{scanFn: exp.scanFn}
}

func (f *fakeTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if len(f.execQueue) == 0 {
		f.t.Fatalf("unexpected Exec: %s", sql)
	}
	exp := f.execQueue[0]
	f.execQueue = f.execQueue[1:]
	f.execCallCount++
	if !strings.Contains(sql, exp.sqlContains) {
		f.t.Fatalf("Exec SQL mismatch\nwant contains: %s\ngot: %s", exp.sqlContains, sql)
	}
	if !reflect.DeepEqual(args, exp.args) {
		f.t.Fatalf("Exec args mismatch\nwant: %#v\ngot: %#v", exp.args, args)
	}
	return exp.tag, exp.err
}

func (f *fakeTx) assertDone() {
	f.t.Helper()
	if len(f.queryRowQueue) != 0 {
		f.t.Fatalf("unused QueryRow expectations: %d", len(f.queryRowQueue))
	}
	if len(f.execQueue) != 0 {
		f.t.Fatalf("unused Exec expectations: %d", len(f.execQueue))
	}
}

func boolRow(value bool) func(dest ...any) error {
	return func(dest ...any) error {
		ptr := dest[0].(*bool)
		*ptr = value
		return nil
	}
}

func managerCandidateRow(role string, isActive bool) func(dest ...any) error {
	return func(dest ...any) error {
		*(dest[0].(*string)) = role
		*(dest[1].(*bool)) = isActive
		return nil
	}
}

func uuidPtrRow(value *uuid.UUID) func(dest ...any) error {
	return func(dest ...any) error {
		ptr := dest[0].(**uuid.UUID)
		*ptr = value
		return nil
	}
}

func errRow(err error) func(dest ...any) error {
	return func(dest ...any) error {
		return err
	}
}

func TestSyncDepartmentManagerAssignmentTx_ReassignsAndDemotesPreviousManager(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-1"
	deptID := uuid.New()
	oldManagerID := uuid.New()
	newManagerID := uuid.New()
	previousDeptID := uuid.New()

	tx := &fakeTx{
		t: t,
		queryRowQueue: []queryRowExpectation{
			{sqlContains: "SELECT EXISTS", args: []any{deptID, tenantID}, scanFn: boolRow(true)},
			{sqlContains: "SELECT manager_id", args: []any{deptID, tenantID}, scanFn: uuidPtrRow(&oldManagerID)},
			{sqlContains: "SELECT role, is_active", args: []any{newManagerID, tenantID}, scanFn: managerCandidateRow("employee", true)},
			{sqlContains: "WHERE tenant_id = $1 AND manager_id = $2", args: []any{tenantID, newManagerID}, scanFn: uuidPtrRow(&previousDeptID)},
			{sqlContains: "SELECT EXISTS", args: []any{tenantID, oldManagerID}, scanFn: boolRow(false)},
		},
		execQueue: []execExpectation{
			{
				sqlContains: "SET manager_id = NULL, updated_at = NOW()",
				args:        []any{previousDeptID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
			{
				sqlContains: "SET manager_id = $1",
				args:        []any{&newManagerID, deptID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
			{
				sqlContains: "SET department_id = $1",
				args:        []any{deptID, newManagerID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
			{
				sqlContains: "SET role = 'employee'",
				args:        []any{oldManagerID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := syncDepartmentManagerAssignmentTx(context.Background(), tx, tenantID, deptID, &newManagerID); err != nil {
		t.Fatalf("syncDepartmentManagerAssignmentTx returned error: %v", err)
	}
	tx.assertDone()
}

func TestSyncDepartmentManagerAssignmentTx_UnassignsAndDemotesCurrentManager(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-2"
	deptID := uuid.New()
	currentManagerID := uuid.New()

	tx := &fakeTx{
		t: t,
		queryRowQueue: []queryRowExpectation{
			{sqlContains: "SELECT EXISTS", args: []any{deptID, tenantID}, scanFn: boolRow(true)},
			{sqlContains: "SELECT manager_id", args: []any{deptID, tenantID}, scanFn: uuidPtrRow(&currentManagerID)},
			{sqlContains: "SELECT EXISTS", args: []any{tenantID, currentManagerID}, scanFn: boolRow(false)},
		},
		execQueue: []execExpectation{
			{
				sqlContains: "SET manager_id = $1",
				args:        []any{(*uuid.UUID)(nil), deptID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
			{
				sqlContains: "SET role = 'employee'",
				args:        []any{currentManagerID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := syncDepartmentManagerAssignmentTx(context.Background(), tx, tenantID, deptID, nil); err != nil {
		t.Fatalf("syncDepartmentManagerAssignmentTx returned error: %v", err)
	}
	tx.assertDone()
}

func TestCleanupDepartmentDeleteTx_UnlinksUsersAndSoftDeletesDepartment(t *testing.T) {
	t.Parallel()

	tenantID := "tenant-3"
	deptID := uuid.New()

	tx := &fakeTx{
		t: t,
		queryRowQueue: []queryRowExpectation{
			{sqlContains: "SELECT EXISTS", args: []any{deptID, tenantID}, scanFn: boolRow(true)},
		},
		execQueue: []execExpectation{
			{
				sqlContains: "UPDATE users",
				args:        []any{tenantID, deptID},
				tag:         pgconn.NewCommandTag("UPDATE 8"),
			},
			{
				sqlContains: "SET manager_id = NULL, deleted_at = NOW(), updated_at = NOW()",
				args:        []any{deptID, tenantID},
				tag:         pgconn.NewCommandTag("UPDATE 1"),
			},
		},
	}

	if err := cleanupDepartmentDeleteTx(context.Background(), tx, tenantID, deptID); err != nil {
		t.Fatalf("cleanupDepartmentDeleteTx returned error: %v", err)
	}
	tx.assertDone()
}
