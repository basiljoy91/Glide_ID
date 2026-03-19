package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"enterprise-attendance-api/internal/models"
	"enterprise-attendance-api/internal/services"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errRoleMutationForbidden   = errors.New("forbidden role mutation")
	errDepartmentScopeRequired = errors.New("department scope required")
	errDepartmentScopeMismatch = errors.New("department scope mismatch")
)

func canAssignRole(actorRole, targetRole string) bool {
	role := strings.TrimSpace(targetRole)
	if role == "" {
		role = "employee"
	}

	switch actorRole {
	case "org_admin":
		return role != "super_admin"
	case "hr":
		return role != "org_admin" && role != "super_admin"
	default:
		return false
	}
}

func canManageRole(actorRole, targetRole string) bool {
	switch actorRole {
	case "org_admin":
		return targetRole != "super_admin"
	case "hr":
		return targetRole != "org_admin" && targetRole != "super_admin"
	default:
		return false
	}
}

func authorizeUserMutation(ctx context.Context, userSvc *services.UserService, tenantID, actorRole, targetUserID string, requestedRole *string) (*models.User, error) {
	target, err := userSvc.GetUser(ctx, tenantID, targetUserID)
	if err != nil {
		return nil, err
	}
	if !canManageRole(actorRole, target.Role) {
		return nil, errRoleMutationForbidden
	}
	if requestedRole != nil && !canAssignRole(actorRole, *requestedRole) {
		return nil, errRoleMutationForbidden
	}
	return target, nil
}

func authorizeBulkUserMutation(ctx context.Context, userSvc *services.UserService, tenantID, actorRole string, targetUserIDs []string) error {
	targets, err := userSvc.GetUsersByIDs(ctx, tenantID, targetUserIDs)
	if err != nil {
		return err
	}
	if len(targets) != len(targetUserIDs) {
		return services.ErrUserNotFound
	}
	for _, target := range targets {
		if !canManageRole(actorRole, target.Role) {
			return errRoleMutationForbidden
		}
	}
	return nil
}

func resolveManagedDepartmentID(ctx context.Context, db *pgxpool.Pool, tenantID, userID, role string) (string, error) {
	if role != "dept_manager" {
		return "", nil
	}

	var departmentID *uuid.UUID
	err := db.QueryRow(ctx, `
		SELECT department_id
		FROM users
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, userID, tenantID).Scan(&departmentID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve manager department: %w", err)
	}
	if departmentID == nil {
		return "", errDepartmentScopeRequired
	}
	return departmentID.String(), nil
}

func enforceDepartmentScope(requestedDepartmentID, scopedDepartmentID string) (string, error) {
	requested := strings.TrimSpace(requestedDepartmentID)
	if scopedDepartmentID == "" {
		return requested, nil
	}
	if requested != "" && requested != scopedDepartmentID {
		return "", errDepartmentScopeMismatch
	}
	return scopedDepartmentID, nil
}

func scheduleDepartmentID(filters map[string]interface{}) string {
	if len(filters) == 0 {
		return ""
	}
	raw, ok := filters["department_id"]
	if !ok || raw == nil {
		return ""
	}
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	case []byte:
		return strings.TrimSpace(string(value))
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func scopeScheduleFilters(filters map[string]interface{}, scopedDepartmentID string) map[string]interface{} {
	if filters == nil {
		filters = map[string]interface{}{}
	}
	if scopedDepartmentID == "" {
		return filters
	}
	filters["department_id"] = scopedDepartmentID
	return filters
}

func scheduleVisibleToDepartment(filters map[string]interface{}, scopedDepartmentID string) bool {
	if scopedDepartmentID == "" {
		return true
	}
	return scheduleDepartmentID(filters) == scopedDepartmentID
}
