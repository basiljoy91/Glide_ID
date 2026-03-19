package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	errInvalidDepartmentManagerCandidate = errors.New("invalid department manager candidate")
	errDepartmentManagerNeedsDepartment  = errors.New("department manager needs department")
	errDepartmentNotFound                = errors.New("department not found")
)

func validateDepartmentExistsTx(ctx context.Context, tx pgx.Tx, tenantID string, deptID uuid.UUID) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM departments
			WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		)
	`, deptID, tenantID).Scan(&exists); err != nil {
		return fmt.Errorf("failed to validate department: %w", err)
	}
	if !exists {
		return errDepartmentNotFound
	}
	return nil
}

func validateDepartmentManagerCandidateTx(ctx context.Context, tx pgx.Tx, tenantID string, userID uuid.UUID) error {
	var role string
	var isActive bool
	err := tx.QueryRow(ctx, `
		SELECT role, is_active
		FROM users
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, userID, tenantID).Scan(&role, &isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errInvalidDepartmentManagerCandidate
		}
		return fmt.Errorf("failed to validate department manager candidate: %w", err)
	}
	if !isActive {
		return errInvalidDepartmentManagerCandidate
	}
	if role != "employee" && role != "dept_manager" {
		return errInvalidDepartmentManagerCandidate
	}
	return nil
}

func loadManagedDepartmentIDTx(ctx context.Context, tx pgx.Tx, tenantID string, userID uuid.UUID) (*uuid.UUID, error) {
	var departmentID *uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM departments
		WHERE tenant_id = $1 AND manager_id = $2 AND deleted_at IS NULL
		LIMIT 1
	`, tenantID, userID).Scan(&departmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load managed department: %w", err)
	}
	return departmentID, nil
}

func loadDepartmentManagerIDTx(ctx context.Context, tx pgx.Tx, tenantID string, deptID uuid.UUID) (*uuid.UUID, error) {
	var managerID *uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT manager_id
		FROM departments
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`, deptID, tenantID).Scan(&managerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errDepartmentNotFound
		}
		return nil, fmt.Errorf("failed to load department manager: %w", err)
	}
	return managerID, nil
}

func demoteManagerIfOrphanedTx(ctx context.Context, tx pgx.Tx, tenantID string, userID uuid.UUID) error {
	var stillManaging bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM departments
			WHERE tenant_id = $1 AND manager_id = $2 AND deleted_at IS NULL
		)
	`, tenantID, userID).Scan(&stillManaging); err != nil {
		return fmt.Errorf("failed to verify manager linkage: %w", err)
	}
	if stillManaging {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET role = 'employee',
			updated_at = NOW()
		WHERE id = $1
		  AND tenant_id = $2
		  AND deleted_at IS NULL
		  AND role = 'dept_manager'
	`, userID, tenantID); err != nil {
		return fmt.Errorf("failed to demote previous manager: %w", err)
	}
	return nil
}

func syncDepartmentManagerAssignmentTx(ctx context.Context, tx pgx.Tx, tenantID string, deptID uuid.UUID, newManagerID *uuid.UUID) error {
	if err := validateDepartmentExistsTx(ctx, tx, tenantID, deptID); err != nil {
		return err
	}
	currentManagerID, err := loadDepartmentManagerIDTx(ctx, tx, tenantID, deptID)
	if err != nil {
		return err
	}

	if newManagerID != nil {
		if err := validateDepartmentManagerCandidateTx(ctx, tx, tenantID, *newManagerID); err != nil {
			return err
		}

		currentManagedDeptID, err := loadManagedDepartmentIDTx(ctx, tx, tenantID, *newManagerID)
		if err != nil {
			return err
		}
		if currentManagedDeptID != nil && *currentManagedDeptID != deptID {
			if _, err := tx.Exec(ctx, `
				UPDATE departments
				SET manager_id = NULL, updated_at = NOW()
				WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
			`, *currentManagedDeptID, tenantID); err != nil {
				return fmt.Errorf("failed to clear previous managed department: %w", err)
			}
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE departments
		SET manager_id = $1,
			updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`, newManagerID, deptID, tenantID); err != nil {
		return fmt.Errorf("failed to update department manager: %w", err)
	}

	if newManagerID != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET department_id = $1,
				role = 'dept_manager',
				updated_at = NOW()
			WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
		`, deptID, *newManagerID, tenantID); err != nil {
			return fmt.Errorf("failed to assign department manager user: %w", err)
		}
	}

	if currentManagerID != nil && (newManagerID == nil || *currentManagerID != *newManagerID) {
		if err := demoteManagerIfOrphanedTx(ctx, tx, tenantID, *currentManagerID); err != nil {
			return err
		}
	}

	return nil
}

func syncUserDepartmentManagerRoleTx(ctx context.Context, tx pgx.Tx, tenantID string, userID uuid.UUID, role string, departmentID *uuid.UUID) error {
	if role == "dept_manager" {
		if departmentID == nil {
			return errDepartmentManagerNeedsDepartment
		}
		return syncDepartmentManagerAssignmentTx(ctx, tx, tenantID, *departmentID, &userID)
	}

	managedDeptID, err := loadManagedDepartmentIDTx(ctx, tx, tenantID, userID)
	if err != nil {
		return err
	}
	if managedDeptID == nil {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE departments
		SET manager_id = NULL, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, *managedDeptID, tenantID); err != nil {
		return fmt.Errorf("failed to clear managed department: %w", err)
	}
	return nil
}

func cleanupDepartmentDeleteTx(ctx context.Context, tx pgx.Tx, tenantID string, deptID uuid.UUID) error {
	if err := validateDepartmentExistsTx(ctx, tx, tenantID, deptID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET department_id = NULL,
			role = CASE WHEN role = 'dept_manager' THEN 'employee' ELSE role END,
			updated_at = NOW()
		WHERE tenant_id = $1 AND department_id = $2 AND deleted_at IS NULL
	`, tenantID, deptID); err != nil {
		return fmt.Errorf("failed to unlink department users: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE departments
		SET manager_id = NULL, deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, deptID, tenantID); err != nil {
		return fmt.Errorf("failed to soft delete department: %w", err)
	}
	return nil
}
