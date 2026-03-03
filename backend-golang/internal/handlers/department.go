package handlers

import (
	"context"
	"time"

	"enterprise-attendance-api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DepartmentDTO struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Code        *string    `json:"code,omitempty"`
	Description *string    `json:"description,omitempty"`
	ManagerID   *uuid.UUID `json:"manager_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListDepartments lists departments for the current tenant
func ListDepartments(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, `
			SELECT id, name, code, description, manager_id, created_at, updated_at
			FROM departments
			WHERE tenant_id = $1 AND deleted_at IS NULL
			ORDER BY name ASC
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
			if err := rows.Scan(&d.ID, &d.Name, &d.Code, &d.Description, &d.ManagerID, &d.CreatedAt, &d.UpdatedAt); err != nil {
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

		var d DepartmentDTO
		err := db.QueryRow(ctx, `
			INSERT INTO departments (
				tenant_id, name, code, description, manager_id, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			RETURNING id, name, code, description, manager_id, created_at, updated_at
		`, tenantID, body.Name, body.Code, body.Description, body.ManagerID).Scan(
			&d.ID, &d.Name, &d.Code, &d.Description, &d.ManagerID, &d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create department",
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

		var body struct {
			Name        *string    `json:"name"`
			Code        *string    `json:"code"`
			Description *string    `json:"description"`
			ManagerID   *uuid.UUID `json:"manager_id"`
		}

		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		var d DepartmentDTO
		err := db.QueryRow(ctx, `
			UPDATE departments
			SET
				name = COALESCE($1, name),
				code = COALESCE($2, code),
				description = COALESCE($3, description),
				manager_id = COALESCE($4, manager_id),
				updated_at = NOW()
			WHERE id = $5 AND tenant_id = $6 AND deleted_at IS NULL
			RETURNING id, name, code, description, manager_id, created_at, updated_at
		`, body.Name, body.Code, body.Description, body.ManagerID, deptID, tenantID).Scan(
			&d.ID, &d.Name, &d.Code, &d.Description, &d.ManagerID, &d.CreatedAt, &d.UpdatedAt,
		)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update department",
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

		_, err := db.Exec(ctx, `
			UPDATE departments
			SET deleted_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		`, deptID, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete department",
			})
		}

		return c.JSON(fiber.Map{"success": true})
	}
}

