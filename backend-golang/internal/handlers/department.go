package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DepartmentDTO struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Code          *string    `json:"code,omitempty"`
	Description   *string    `json:"description,omitempty"`
	ManagerID     *uuid.UUID `json:"manager_id,omitempty"`
	ManagerName   *string    `json:"manager_name,omitempty"`
	EmployeeCount int        `json:"employee_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func loadDepartmentDTO(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}, tenantID string, deptID uuid.UUID) (*DepartmentDTO, error) {
	var d DepartmentDTO
	err := q.QueryRow(ctx, `
		SELECT
			d.id, d.name, d.code, d.description, d.manager_id, d.created_at, d.updated_at,
			(SELECT COUNT(*) FROM users u WHERE u.department_id = d.id AND u.is_active = true AND u.deleted_at IS NULL) as employee_count,
			m.first_name || ' ' || m.last_name as manager_name
		FROM departments d
		LEFT JOIN users m ON m.id = d.manager_id
		WHERE d.tenant_id = $1 AND d.id = $2 AND d.deleted_at IS NULL
	`, tenantID, deptID).Scan(
		&d.ID, &d.Name, &d.Code, &d.Description, &d.ManagerID, &d.CreatedAt, &d.UpdatedAt, &d.EmployeeCount, &d.ManagerName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errDepartmentNotFound
		}
		return nil, err
	}
	return &d, nil
}

// ListDepartments lists departments for the current tenant
func ListDepartments(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT 
				d.id, d.name, d.code, d.description, d.manager_id, d.created_at, d.updated_at,
				(SELECT COUNT(*) FROM users u WHERE u.department_id = d.id AND u.is_active = true) as employee_count,
				m.first_name || ' ' || m.last_name as manager_name
			FROM departments d
			LEFT JOIN users m ON m.id = d.manager_id
			WHERE d.tenant_id = $1 AND d.deleted_at IS NULL
			ORDER BY d.name ASC
		`, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to list departments",
			})
		}
		defer rows.Close()

		departments := make([]DepartmentDTO, 0)
		for rows.Next() {
			var d DepartmentDTO
			if err := rows.Scan(&d.ID, &d.Name, &d.Code, &d.Description, &d.ManagerID, &d.CreatedAt, &d.UpdatedAt, &d.EmployeeCount, &d.ManagerName); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to read departments",
				})
			}
			departments = append(departments, d)
		}

		return c.JSON(departments)
	}
}

// CreateDepartment creates a department for the current tenant
func CreateDepartment(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var body struct {
			Name        string     `json:"name"`
			Code        *string    `json:"code"`
			Description *string    `json:"description"`
			ManagerID   *uuid.UUID `json:"manager_id"`
		}

		if err := c.BodyParser(&body); err != nil || body.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		tx, err := db.Begin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to start department creation",
			})
		}
		defer tx.Rollback(c.Context())

		var deptID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO departments (
				tenant_id, name, code, description, manager_id, created_at, updated_at
			) VALUES ($1, $2, $3, $4, NULL, NOW(), NOW())
			RETURNING id
		`, tenantID, body.Name, body.Code, body.Description).Scan(&deptID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create department",
			})
		}

		if body.ManagerID != nil {
			if err := syncDepartmentManagerAssignmentTx(ctx, tx, tenantID, deptID, body.ManagerID); err != nil {
				switch {
				case errors.Is(err, errInvalidDepartmentManagerCandidate):
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Manager must be an active employee or department manager"})
				default:
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to assign department manager"})
				}
			}
		}

		if err := tx.Commit(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to finalize department creation",
			})
		}

		d, err := loadDepartmentDTO(ctx, db, tenantID, deptID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to load created department",
			})
		}
		return c.Status(fiber.StatusCreated).JSON(d)
	}
}

// UpdateDepartment updates a department
func UpdateDepartment(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		deptID := c.Params("id")

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(c.Body(), &raw); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		var body struct {
			Name        *string
			Code        *string
			Description *string
		}
		if rawName, ok := raw["name"]; ok {
			if err := json.Unmarshal(rawName, &body.Name); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid name"})
			}
		}
		if rawCode, ok := raw["code"]; ok {
			if err := json.Unmarshal(rawCode, &body.Code); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid code"})
			}
		}
		if rawDescription, ok := raw["description"]; ok {
			if err := json.Unmarshal(rawDescription, &body.Description); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid description"})
			}
		}
		managerSet := false
		var managerID *uuid.UUID
		if rawManagerID, ok := raw["manager_id"]; ok {
			managerSet = true
			if string(rawManagerID) != "null" {
				var parsed string
				if err := json.Unmarshal(rawManagerID, &parsed); err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid manager_id"})
				}
				if parsed != "" {
					id, err := uuid.Parse(parsed)
					if err != nil {
						return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid manager_id"})
					}
					managerID = &id
				}
			}
		}

		tx, err := db.Begin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to start department update",
			})
		}
		defer tx.Rollback(c.Context())

		deptUUID, err := uuid.Parse(deptID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid department id"})
		}
		if err := validateDepartmentExistsTx(ctx, tx, tenantID, deptUUID); err != nil {
			status := fiber.StatusInternalServerError
			message := "Failed to update department"
			if errors.Is(err, errDepartmentNotFound) {
				status = fiber.StatusNotFound
				message = "Department not found"
			}
			return c.Status(status).JSON(fiber.Map{"error": message})
		}

		tag, err := tx.Exec(ctx, `
			UPDATE departments
			SET
				name = COALESCE($1, name),
				code = COALESCE($2, code),
				description = COALESCE($3, description),
				updated_at = NOW()
			WHERE id = $4 AND tenant_id = $5 AND deleted_at IS NULL
		`, body.Name, body.Code, body.Description, deptUUID, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update department",
			})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Department not found",
			})
		}

		if managerSet {
			if err := syncDepartmentManagerAssignmentTx(ctx, tx, tenantID, deptUUID, managerID); err != nil {
				switch {
				case errors.Is(err, errInvalidDepartmentManagerCandidate):
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Manager must be an active employee or department manager"})
				case errors.Is(err, errDepartmentNotFound):
					return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Department not found"})
				default:
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update department manager"})
				}
			}
		}

		if err := tx.Commit(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to finalize department update",
			})
		}

		d, err := loadDepartmentDTO(ctx, db, tenantID, deptUUID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to load updated department",
			})
		}
		return c.JSON(d)
	}
}

// DeleteDepartment soft-deletes a department
func DeleteDepartment(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		deptID := c.Params("id")

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		deptUUID, err := uuid.Parse(deptID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid department id"})
		}

		tx, err := db.Begin(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to start department deletion",
			})
		}
		defer tx.Rollback(c.Context())

		if err := cleanupDepartmentDeleteTx(ctx, tx, tenantID, deptUUID); err != nil {
			switch {
			case errors.Is(err, errDepartmentNotFound):
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Department not found"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete department"})
			}
		}

		if err := tx.Commit(c.Context()); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to finalize department deletion",
			})
		}

		return c.JSON(fiber.Map{"success": true})
	}
}
