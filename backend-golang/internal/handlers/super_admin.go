package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type superAdminMetricsResponse struct {
	TotalOrganizations  int     `json:"totalOrganizations"`
	ActiveOrganizations int     `json:"activeOrganizations"`
	TotalUsers          int     `json:"totalUsers"`
	TotalCheckIns       int     `json:"totalCheckIns"`
	MonthlyRevenue      int     `json:"monthlyRevenue"`
	GrowthRate          float64 `json:"growthRate"`
}

// GetSuperAdminMetrics returns global platform metrics (no PII).
func GetSuperAdminMetrics(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var totalOrgs int
		var activeOrgs int
		var totalUsers int
		var totalCheckIns int

		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL`).Scan(&totalOrgs); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count organizations"})
		}
		activeOrgs = totalOrgs

		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&totalUsers); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count users"})
		}

		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM attendance_logs`).Scan(&totalCheckIns); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count check-ins"})
		}

		// Rough revenue model by tier (placeholder until billing tables exist)
		var monthlyRevenue int
		if err := db.QueryRow(ctx, `
			SELECT COALESCE(SUM(
				CASE subscription_tier
					WHEN 'free' THEN 0
					WHEN 'starter' THEN 199
					WHEN 'professional' THEN 499
					WHEN 'enterprise' THEN 999
					ELSE 0
				END
			), 0)
			FROM tenants
			WHERE deleted_at IS NULL
		`).Scan(&monthlyRevenue); err != nil {
			monthlyRevenue = 0
		}

		// Growth rate based on tenant signups (last 30 days vs previous 30 days)
		var current30 int
		var prev30 int
		_ = db.QueryRow(ctx, `
			SELECT COUNT(*) FROM tenants
			WHERE deleted_at IS NULL AND created_at >= NOW() - INTERVAL '30 days'
		`).Scan(&current30)
		_ = db.QueryRow(ctx, `
			SELECT COUNT(*) FROM tenants
			WHERE deleted_at IS NULL
			  AND created_at < NOW() - INTERVAL '30 days'
			  AND created_at >= NOW() - INTERVAL '60 days'
		`).Scan(&prev30)

		var growthRate float64
		if prev30 > 0 {
			growthRate = float64(current30-prev30) / float64(prev30) * 100
		} else if current30 > 0 {
			growthRate = 100
		} else {
			growthRate = 0
		}

		return c.JSON(superAdminMetricsResponse{
			TotalOrganizations:  totalOrgs,
			ActiveOrganizations: activeOrgs,
			TotalUsers:          totalUsers,
			TotalCheckIns:       totalCheckIns,
			MonthlyRevenue:      monthlyRevenue,
			GrowthRate:          growthRate,
		})
	}
}


