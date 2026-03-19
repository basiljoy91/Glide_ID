package handlers

import (
	"context"
	"errors"
	"fmt"
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

type queryExpectation struct {
	sqlContains string
	args        []any
	rows        [][]any
	err         error
}

type fakeRows struct {
	t      *testing.T
	rows   [][]any
	idx    int
	closed bool
	err    error
}

type fakeTx struct {
	t                 *testing.T
	queryRowQueue     []queryRowExpectation
	queryQueue        []queryExpectation
	execQueue         []execExpectation
	queryRowCallCount int
	queryCallCount    int
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
func (f *fakeTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	if len(f.queryQueue) == 0 {
		f.t.Fatalf("unexpected Query: %s", sql)
	}
	exp := f.queryQueue[0]
	f.queryQueue = f.queryQueue[1:]
	f.queryCallCount++
	if !strings.Contains(sql, exp.sqlContains) {
		f.t.Fatalf("Query SQL mismatch\nwant contains: %s\ngot: %s", exp.sqlContains, sql)
	}
	if !reflect.DeepEqual(args, exp.args) {
		f.t.Fatalf("Query args mismatch\nwant: %#v\ngot: %#v", exp.args, args)
	}
	if exp.err != nil {
		return nil, exp.err
	}
	return &fakeRows{t: f.t, rows: exp.rows, idx: -1}, nil
}
func (f *fakeTx) Conn() *pgx.Conn { return nil }

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
	if len(f.queryQueue) != 0 {
		f.t.Fatalf("unused Query expectations: %d", len(f.queryQueue))
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

func (r *fakeRows) Close() { r.closed = true }
func (r *fakeRows) Err() error {
	return r.err
}
func (r *fakeRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (r *fakeRows) Next() bool {
	if r.idx+1 >= len(r.rows) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.idx < 0 || r.idx >= len(r.rows) {
		return errors.New("scan called with no current row")
	}
	values := r.rows[r.idx]
	if len(values) != len(dest) {
		return fmt.Errorf("row length mismatch: got %d values for %d destinations", len(values), len(dest))
	}
	for i, value := range values {
		target := reflect.ValueOf(dest[i])
		if target.Kind() != reflect.Ptr || target.IsNil() {
			return fmt.Errorf("destination %d is not a non-nil pointer", i)
		}
		elem := target.Elem()
		if value == nil {
			elem.Set(reflect.Zero(elem.Type()))
			continue
		}
		v := reflect.ValueOf(value)
		if v.Type().AssignableTo(elem.Type()) {
			elem.Set(v)
			continue
		}
		if v.Type().ConvertibleTo(elem.Type()) {
			elem.Set(v.Convert(elem.Type()))
			continue
		}
		return fmt.Errorf("cannot assign %T to %s", value, elem.Type())
	}
	return nil
}
func (r *fakeRows) Values() ([]any, error) {
	if r.idx < 0 || r.idx >= len(r.rows) {
		return nil, errors.New("no current row")
	}
	return r.rows[r.idx], nil
}
func (r *fakeRows) RawValues() [][]byte { return nil }
func (r *fakeRows) Conn() *pgx.Conn     { return nil }

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
