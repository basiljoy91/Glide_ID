package services

import (
	"context"
	"fmt"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	db *pgxpool.Pool
}

func NewUserService(db *pgxpool.Pool) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, tenantID string, user *models.User) error {
	user.ID = uuid.New()
	user.TenantID = uuid.MustParse(tenantID)
	user.CreatedAt = user.CreatedAt
	user.UpdatedAt = user.UpdatedAt

	_, err := s.db.Exec(ctx, `
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
	err := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, employee_id, email, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, userID, tenantID).Scan(
		&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.Phone,
		&user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation,
		&user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
		&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent,
		&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// ListUsers lists all users for a tenant
func (s *UserService) ListUsers(ctx context.Context, tenantID string, limit, offset int) ([]*models.User, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, tenant_id, employee_id, email, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)

	if err != nil {
		return nil, err
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
			&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUserByEmail retrieves a user by email (searches across tenants for login)
func (s *UserService) GetUserByEmail(ctx context.Context, tenantID, email string) (*models.User, error) {
	// If tenantID is empty, search across all tenants (for login)
	if tenantID == "" {
		user := &models.User{}
		err := s.db.QueryRow(ctx, `
			SELECT id, tenant_id, employee_id, email, password_hash, phone, first_name, last_name,
				department_id, designation, date_of_joining, shift_start_time, shift_end_time,
				shift_length_hours, role, is_active, data_privacy_consent, consent_date,
				last_login_at, created_at, updated_at, deleted_at
			FROM users
			WHERE email = $1 AND deleted_at IS NULL
			LIMIT 1
		`, email).Scan(
			&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.PasswordHash, &user.Phone,
			&user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation,
			&user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
			&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent,
			&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

		if err != nil {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return user, nil
	}

	// Otherwise, search within specific tenant
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
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// UpdateUserBasic updates core user fields within a tenant and returns the updated user.
func (s *UserService) UpdateUserBasic(ctx context.Context, tenantID, userID string, u *models.User) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(ctx, `
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
		u.Role, u.IsActive, userID, tenantID).Scan(
		&user.ID, &user.TenantID, &user.EmployeeID, &user.Email, &user.Phone,
		&user.FirstName, &user.LastName, &user.DepartmentID, &user.Designation,
		&user.DateOfJoining, &user.ShiftStartTime, &user.ShiftEndTime,
		&user.ShiftLengthHours, &user.Role, &user.IsActive, &user.DataPrivacyConsent,
		&user.ConsentDate, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

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
