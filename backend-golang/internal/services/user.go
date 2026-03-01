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
			shift_length_hours, role, is_active, data_privacy_consent, consent_date, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, user.ID, user.TenantID, user.EmployeeID, user.Email, user.Phone,
		user.FirstName, user.LastName, user.DepartmentID, user.Designation,
		user.DateOfJoining, user.ShiftStartTime, user.ShiftEndTime,
		user.ShiftLengthHours, user.Role, user.IsActive, user.DataPrivacyConsent,
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
			SELECT id, tenant_id, employee_id, email, phone, first_name, last_name,
				department_id, designation, date_of_joining, shift_start_time, shift_end_time,
				shift_length_hours, role, is_active, data_privacy_consent, consent_date,
				last_login_at, created_at, updated_at, deleted_at
			FROM users
			WHERE email = $1 AND deleted_at IS NULL
			LIMIT 1
		`, email).Scan(
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
	
	// Otherwise, search within specific tenant
	user := &models.User{}
	err := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, employee_id, email, phone, first_name, last_name,
			department_id, designation, date_of_joining, shift_start_time, shift_end_time,
			shift_length_hours, role, is_active, data_privacy_consent, consent_date,
			last_login_at, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, email, tenantID).Scan(
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

