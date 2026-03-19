package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	db *pgxpool.Pool
}

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrAmbiguousLoginUser = errors.New("multiple users found for email")
)

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetDB() *pgxpool.Pool {
	return s.db
}

func scanUserRow(row pgx.Row, user *models.User) error {
	return row.Scan(
		&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.Phone,
		&user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation,
		&user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
		&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent,
		&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, tenantID string, user *models.User) error {
	return s.CreateUserTx(ctx, s.db, tenantID, user)
}

// CreateUserTx creates a new user within an existing transaction.
func (s *UserService) CreateUserTx(ctx context.Context, tx interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
}, tenantID string, user *models.User) error {
	user.ID = uuid.New()
	user.TenantID = uuid.MustParse(tenantID)
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	_, err := tx.Exec(ctx, `
		INSERT INTO users (
			id, tenant_id, employee_id, email, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, password_hash, is_active, data_privacy_consent, consent_date, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, user.ID, user.TenantID, user.EmployeeID, user.Email, user.Phone,
		user.FirstName, user.LastName, user.DepartmentID, user.Designation,
		user.DateOfJoining, user.ShiftStartTime, user.ShiftEndTime,
		user.ShiftLengthHours, user.Role, user.PasswordHash, user.IsActive, user.DataPrivacyConsent,
		user.ConsentDate, user.CreatedAt, user.UpdatedAt)

	return err
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, tenantID, userID string) (*models.User, error) {
	user := &models.User{}
	err := scanUserRow(s.db.QueryRow(ctx, `
		SELECT id, tenant_id, employee_id, email, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, userID, tenantID), user)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// GetUsersByIDs retrieves multiple users by ID within a tenant.
func (s *UserService) GetUsersByIDs(ctx context.Context, tenantID string, userIDs []string) ([]*models.User, error) {
	ids := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("invalid user id: %s", id)
		}
		ids = append(ids, parsed)
	}
	if len(ids) == 0 {
		return []*models.User{}, nil
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, employee_id, email, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE tenant_id = $1 AND id = ANY($2::uuid[]) AND deleted_at IS NULL
	`, tenantID, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list users by id: %w", err)
	}
	defer rows.Close()

	users := make([]*models.User, 0, len(ids))
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(
			&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.Phone,
			&user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation,
			&user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
			&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent,
			&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to read user rows: %w", err)
		}
		users = append(users, user)
	}
	return users, nil
}

// ListUsers lists all users for a tenant
func (s *UserService) ListUsers(
	ctx context.Context,
	tenantID string,
	limit, offset int,
	query, role, status, sortBy, sortDir string,
) ([]*models.User, int, error) {
	where := []string{"u.tenant_id = $1", "u.deleted_at IS NULL"}
	args := []interface{}{tenantID}

	if strings.TrimSpace(query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(query))+"%")
		where = append(where, fmt.Sprintf("(LOWER(u.first_name || ' ' || u.last_name) LIKE $%d OR LOWER(u.email) LIKE $%d OR LOWER(u.employee_id) LIKE $%d)", len(args), len(args), len(args)))
	}
	if role != "" && role != "all" {
		args = append(args, role)
		where = append(where, fmt.Sprintf("u.role = $%d", len(args)))
	}
	if status == "active" {
		where = append(where, "u.is_active = true")
	} else if status == "inactive" {
		where = append(where, "u.is_active = false")
	}

	sortSQL := buildUserSort(sortBy, sortDir)
	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s`, whereClause)
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf(`
		SELECT u.id, u.tenant_id, u.employee_id, u.email, u.phone, u.first_name, u.last_name,
			u.department_id, u.designation, u.date_of_joining, u.shift_start_time, u.shift_end_time,
			u.shift_length_hours, u.role, u.is_active, u.data_privacy_consent, u.consent_date,
			u.last_login_at, u.created_at, u.updated_at, u.deleted_at,
			al.last_check_in_at
		FROM users u
		LEFT JOIN (
			SELECT user_id, MAX(punch_time) AS last_check_in_at
			FROM attendance_logs
			WHERE tenant_id = $1
			GROUP BY user_id
		) al ON al.user_id = u.id
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortSQL, len(args)+1, len(args)+2)

	args = append(args, limit, offset)
	rows, err := s.db.Query(ctx, listQuery, args...)

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(
			&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.Phone,
			&user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation,
			&user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
			&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent,
			&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
			&user.LastCheckInAt); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, nil
}

func buildUserSort(sortBy, sortDir string) string {
	dir := "DESC"
	if strings.EqualFold(sortDir, "asc") {
		dir = "ASC"
	}

	switch sortBy {
	case "employee_id":
		return fmt.Sprintf("ORDER BY u.employee_id %s", dir)
	case "name":
		return fmt.Sprintf("ORDER BY LOWER(u.first_name) %s, LOWER(u.last_name) %s", dir, dir)
	case "email":
		return fmt.Sprintf("ORDER BY u.email %s", dir)
	case "role":
		return fmt.Sprintf("ORDER BY u.role %s", dir)
	case "status":
		return fmt.Sprintf("ORDER BY u.is_active %s", dir)
	case "last_login":
		return fmt.Sprintf("ORDER BY u.last_login_at %s NULLS LAST", dir)
	case "last_check_in":
		return fmt.Sprintf("ORDER BY al.last_check_in_at %s NULLS LAST", dir)
	case "created_at":
		return fmt.Sprintf("ORDER BY u.created_at %s", dir)
	default:
		return "ORDER BY u.created_at DESC"
	}
}

// GetUserByEmail retrieves a user by email within a tenant.
func (s *UserService) GetUserByEmail(ctx context.Context, tenantID, email string) (*models.User, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant id is required: %w", ErrUserNotFound)
	}

	user := &models.User{}
	err := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, employee_id, email, password_hash, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, email, tenantID).Scan(
		&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.PasswordHash, &user.Phone,
		&user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation,
		&user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
		&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent,
		&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUserNotFound, email)
	}

	return user, nil
}

// FindLoginUserByEmail resolves a password-login user safely across tenants.
func (s *UserService) FindLoginUserByEmail(ctx context.Context, email string) (*models.User, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, employee_id, email, password_hash, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT 2
	`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(
			&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.PasswordHash, &user.Phone,
			&user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation,
			&user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
			&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent,
			&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	switch len(users) {
	case 0:
		return nil, fmt.Errorf("%w: %s", ErrUserNotFound, email)
	case 1:
		return users[0], nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrAmbiguousLoginUser, email)
	}
}

// UpdateLastLogin records a successful sign-in timestamp for a user.
func (s *UserService) UpdateLastLogin(ctx context.Context, tenantID, userID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// UpdateUserBasic updates core user fields within a tenant and returns the updated user.
func (s *UserService) UpdateUserBasic(ctx context.Context, tenantID, userID string, u *models.User) (*models.User, error) {
	return s.UpdateUserBasicTx(ctx, s.db, tenantID, userID, u)
}

// UpdateUserBasicTx updates core user fields within a tenant inside a transaction.
func (s *UserService) UpdateUserBasicTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}, tenantID, userID string, u *models.User) (*models.User, error) {
	user := &models.User{}
	err := scanUserRow(tx.QueryRow(ctx, `
		UPDATE users
		SET
			email = $1,
			phone = $2,
			first_name = $3,
			last_name = $4,
			department_id = $5,
			designation = $6,
			role = $7,
			is_active = $8,
			updated_at = NOW()
		WHERE id = $9 AND tenant_id = $10 AND deleted_at IS NULL
		RETURNING id, tenant_id, employee_id, email, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, created_at, updated_at, deleted_at
	`, u.Email, u.Phone, u.FirstName, u.LastName, u.DepartmentID, u.Designation,
		u.Role, u.IsActive, userID, tenantID), user)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return user, nil
}

// SoftDeleteUser marks a user as deleted within a tenant.
func (s *UserService) SoftDeleteUser(ctx context.Context, tenantID, userID string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// SetUserPasswordHash updates a user's password hash.
func (s *UserService) SetUserPasswordHash(ctx context.Context, tenantID, userID, passwordHash string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`, passwordHash, userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to reset user password: %w", err)
	}
	return nil
}

// BulkSetActive updates is_active for multiple users in one operation.
func (s *UserService) BulkSetActive(ctx context.Context, tenantID string, userIDs []string, active bool) (int64, error) {
	ids := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return 0, fmt.Errorf("invalid user id: %s", id)
		}
		ids = append(ids, parsed)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE users
		SET is_active = $1, updated_at = NOW()
		WHERE tenant_id = $2
		  AND id = ANY($3::uuid[])
		  AND deleted_at IS NULL
	`, active, tenantID, ids)
	if err != nil {
		return 0, fmt.Errorf("failed to update user status: %w", err)
	}
	return tag.RowsAffected(), nil
}

// BulkSoftDelete soft-deletes multiple users.
func (s *UserService) BulkSoftDelete(ctx context.Context, tenantID string, userIDs []string) (int64, error) {
	ids := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return 0, fmt.Errorf("invalid user id: %s", id)
		}
		ids = append(ids, parsed)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE users
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE tenant_id = $1
		  AND id = ANY($2::uuid[])
		  AND deleted_at IS NULL
	`, tenantID, ids)
	if err != nil {
		return 0, fmt.Errorf("failed to delete users: %w", err)
	}
	return tag.RowsAffected(), nil
}
