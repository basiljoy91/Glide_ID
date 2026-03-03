package handlers

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

type superAdminOrganizationRow struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Slug               string     `json:"slug"`
	SubscriptionTier   string     `json:"subscription_tier"`
	BillingStatus      string     `json:"billing_status"`
	SeatCount          int        `json:"seat_count"`
	BaseAmountCents    int        `json:"base_amount_cents"`
	PerSeatAmountCents int        `json:"per_seat_amount_cents"`
	EstimatedMRRCents  int        `json:"estimated_mrr_cents"`
	UsersCount         int        `json:"users_count"`
	CurrentPeriodStart *time.Time `json:"current_period_start"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end"`
	IsActive           bool       `json:"is_active"`
	CreatedAt          time.Time  `json:"created_at"`
}

type billingOverviewResponse struct {
	ActiveSubscriptions int `json:"active_subscriptions"`
	MonthlyRecurringRev int `json:"monthly_recurring_revenue_cents"`
	PaidThisMonth       int `json:"paid_this_month_cents"`
	OutstandingAmount   int `json:"outstanding_amount_cents"`
	OverdueInvoices     int `json:"overdue_invoices"`
	OpenInvoices        int `json:"open_invoices"`
}

type billingInvoiceRow struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	TenantName       string     `json:"tenant_name"`
	SubscriptionID   *uuid.UUID `json:"subscription_id"`
	InvoiceNumber    string     `json:"invoice_number"`
	Status           string     `json:"status"`
	PeriodStart      time.Time  `json:"period_start"`
	PeriodEnd        time.Time  `json:"period_end"`
	SubtotalCents    int        `json:"subtotal_cents"`
	TaxCents         int        `json:"tax_cents"`
	TotalCents       int        `json:"total_cents"`
	DueAt            *time.Time `json:"due_at"`
	PaidAt           *time.Time `json:"paid_at"`
	PaymentReference *string    `json:"payment_reference"`
	CreatedAt        time.Time  `json:"created_at"`
}

// GetSuperAdminMetrics returns global platform metrics (no PII).
func GetSuperAdminMetrics(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 6*time.Second)
		defer cancel()

		var totalOrgs int
		var activeOrgs int
		var totalUsers int
		var totalCheckIns int
		var monthlyRevenue int

		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL`).Scan(&totalOrgs); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count organizations"})
		}

		// Active organizations are mapped from active subscriptions where tenant is not deleted.
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM tenants t
			LEFT JOIN billing_subscriptions bs ON bs.tenant_id = t.id
			WHERE t.deleted_at IS NULL
			  AND COALESCE(bs.status::text, 'active') IN ('active', 'trialing', 'past_due')
		`).Scan(&activeOrgs); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count active organizations"})
		}

		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&totalUsers); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count users"})
		}

		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM attendance_logs`).Scan(&totalCheckIns); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count check-ins"})
		}

		// Realized revenue = paid invoices in current month.
		_ = db.QueryRow(ctx, `
			SELECT COALESCE(SUM(total_cents), 0)
			FROM billing_invoices
			WHERE status = 'paid'
			  AND paid_at >= date_trunc('month', NOW())
			  AND paid_at < (date_trunc('month', NOW()) + INTERVAL '1 month')
		`).Scan(&monthlyRevenue)

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
		}

		return c.JSON(superAdminMetricsResponse{
			TotalOrganizations:  totalOrgs,
			ActiveOrganizations: activeOrgs,
			TotalUsers:          totalUsers,
			TotalCheckIns:       totalCheckIns,
			MonthlyRevenue:      monthlyRevenue / 100,
			GrowthRate:          growthRate,
		})
	}
}

// ListSuperAdminOrganizations returns tenant list with billing projection and usage.
func ListSuperAdminOrganizations(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 100)
		offset := c.QueryInt("offset", 0)
		q := strings.TrimSpace(c.Query("q"))
		if limit <= 0 || limit > 500 {
			limit = 100
		}
		if offset < 0 {
			offset = 0
		}

		ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
		defer cancel()

		args := []interface{}{limit, offset}
		where := "1=1"
		if q != "" {
			where += fmt.Sprintf(" AND (t.name ILIKE $%d OR t.slug ILIKE $%d)", len(args)+1, len(args)+1)
			args = append(args, "%"+q+"%")
		}

		rows, err := db.Query(ctx, fmt.Sprintf(`
			SELECT
				t.id, t.name, t.slug, t.subscription_tier::text,
				COALESCE(bs.status::text, 'inactive') AS billing_status,
				COALESCE(bs.seat_count, GREATEST(t.max_users, 1)) AS seat_count,
				COALESCE(bs.base_amount_cents, 0) AS base_amount_cents,
				COALESCE(bs.per_seat_amount_cents, 0) AS per_seat_amount_cents,
				COALESCE(bs.current_period_start, NULL) AS current_period_start,
				COALESCE(bs.current_period_end, NULL) AS current_period_end,
				COALESCE(uc.users_count, 0) AS users_count,
				(t.deleted_at IS NULL) AS is_active,
				t.created_at
			FROM tenants t
			LEFT JOIN billing_subscriptions bs ON bs.tenant_id = t.id
			LEFT JOIN (
				SELECT tenant_id, COUNT(*)::int AS users_count
				FROM users
				WHERE deleted_at IS NULL
				GROUP BY tenant_id
			) uc ON uc.tenant_id = t.id
			WHERE %s
			ORDER BY t.created_at DESC
			LIMIT $1 OFFSET $2
		`, where), args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list organizations"})
		}
		defer rows.Close()

		out := make([]superAdminOrganizationRow, 0, limit)
		for rows.Next() {
			var r superAdminOrganizationRow
			if err := rows.Scan(
				&r.ID, &r.Name, &r.Slug, &r.SubscriptionTier, &r.BillingStatus,
				&r.SeatCount, &r.BaseAmountCents, &r.PerSeatAmountCents,
				&r.CurrentPeriodStart, &r.CurrentPeriodEnd,
				&r.UsersCount, &r.IsActive, &r.CreatedAt,
			); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read organizations"})
			}
			r.EstimatedMRRCents = r.BaseAmountCents + (r.PerSeatAmountCents * r.SeatCount)
			out = append(out, r)
		}
		return c.JSON(out)
	}
}

// UpdateOrganizationSubscription updates tenant plan and billing subscription settings.
func UpdateOrganizationSubscription(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")
		if _, err := uuid.Parse(tenantID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tenant ID"})
		}

		var body struct {
			PlanTier           *string `json:"plan_tier"`
			Status             *string `json:"status"`
			SeatCount          *int    `json:"seat_count"`
			BaseAmountCents    *int    `json:"base_amount_cents"`
			PerSeatAmountCents *int    `json:"per_seat_amount_cents"`
			BillingCycle       *string `json:"billing_cycle"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
		defer cancel()

		tx, err := db.Begin(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to start transaction"})
		}
		defer tx.Rollback(ctx)

		var existingPlan string
		err = tx.QueryRow(ctx, `SELECT subscription_tier::text FROM tenants WHERE id = $1 AND deleted_at IS NULL`, tenantID).Scan(&existingPlan)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Organization not found"})
		}

		var currentStatus string
		var seatCount int
		var baseAmount int
		var perSeatAmount int
		var billingCycle string
		err = tx.QueryRow(ctx, `
			SELECT status::text, seat_count, base_amount_cents, per_seat_amount_cents, billing_cycle
			FROM billing_subscriptions
			WHERE tenant_id = $1
		`, tenantID).Scan(&currentStatus, &seatCount, &baseAmount, &perSeatAmount, &billingCycle)
		if err == pgx.ErrNoRows {
			currentStatus = "active"
			seatCount = 1
			billingCycle = "monthly"
			baseAmount = 0
			perSeatAmount = 0
		} else if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load subscription"})
		}

		plan := existingPlan
		if body.PlanTier != nil && *body.PlanTier != "" {
			plan = *body.PlanTier
		}
		if body.Status != nil && *body.Status != "" {
			currentStatus = *body.Status
		}
		if body.SeatCount != nil && *body.SeatCount > 0 {
			seatCount = *body.SeatCount
		}
		if body.BaseAmountCents != nil && *body.BaseAmountCents >= 0 {
			baseAmount = *body.BaseAmountCents
		}
		if body.PerSeatAmountCents != nil && *body.PerSeatAmountCents >= 0 {
			perSeatAmount = *body.PerSeatAmountCents
		}
		if body.BillingCycle != nil && *body.BillingCycle != "" {
			billingCycle = *body.BillingCycle
		}

		if _, err := tx.Exec(ctx, `
			UPDATE tenants
			SET subscription_tier = $1::subscription_tier, updated_at = NOW()
			WHERE id = $2
		`, plan, tenantID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid subscription tier"})
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO billing_subscriptions (
				tenant_id, plan_tier, status, seat_count, base_amount_cents, per_seat_amount_cents,
				billing_cycle, current_period_start, current_period_end, next_invoice_at, updated_at
			)
			VALUES (
				$1, $2::subscription_tier, $3::billing_subscription_status, $4, $5, $6,
				$7, NOW() - INTERVAL '30 days', NOW(), NOW() + INTERVAL '30 days', NOW()
			)
			ON CONFLICT (tenant_id)
			DO UPDATE SET
				plan_tier = EXCLUDED.plan_tier,
				status = EXCLUDED.status,
				seat_count = EXCLUDED.seat_count,
				base_amount_cents = EXCLUDED.base_amount_cents,
				per_seat_amount_cents = EXCLUDED.per_seat_amount_cents,
				billing_cycle = EXCLUDED.billing_cycle,
				updated_at = NOW()
		`, tenantID, plan, currentStatus, seatCount, baseAmount, perSeatAmount, billingCycle); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to update subscription details"})
		}

		if err := tx.Commit(ctx); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to commit subscription update"})
		}

		return c.JSON(fiber.Map{
			"success":               true,
			"tenant_id":             tenantID,
			"plan_tier":             plan,
			"status":                currentStatus,
			"seat_count":            seatCount,
			"base_amount_cents":     baseAmount,
			"per_seat_amount_cents": perSeatAmount,
			"billing_cycle":         billingCycle,
		})
	}
}

// SetOrganizationStatus activates/deactivates an organization.
func SetOrganizationStatus(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := c.Params("id")
		if _, err := uuid.Parse(tenantID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tenant ID"})
		}
		var body struct {
			IsActive bool `json:"is_active"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		if body.IsActive {
			_, err := db.Exec(ctx, `UPDATE tenants SET deleted_at = NULL, updated_at = NOW() WHERE id = $1`, tenantID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to activate organization"})
			}
			return c.JSON(fiber.Map{"success": true, "is_active": true})
		}

		_, err := db.Exec(ctx, `UPDATE tenants SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to deactivate organization"})
		}
		return c.JSON(fiber.Map{"success": true, "is_active": false})
	}
}

// GetBillingOverview returns high-level billing KPIs.
func GetBillingOverview(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 6*time.Second)
		defer cancel()

		var resp billingOverviewResponse
		_ = db.QueryRow(ctx, `
			SELECT COALESCE(COUNT(*), 0)
			FROM billing_subscriptions
			WHERE status IN ('active', 'trialing', 'past_due')
		`).Scan(&resp.ActiveSubscriptions)
		_ = db.QueryRow(ctx, `
			SELECT COALESCE(SUM(base_amount_cents + (per_seat_amount_cents * seat_count)), 0)
			FROM billing_subscriptions
			WHERE status IN ('active', 'trialing', 'past_due')
		`).Scan(&resp.MonthlyRecurringRev)
		_ = db.QueryRow(ctx, `
			SELECT COALESCE(SUM(total_cents), 0)
			FROM billing_invoices
			WHERE status = 'paid'
			  AND paid_at >= date_trunc('month', NOW())
			  AND paid_at < (date_trunc('month', NOW()) + INTERVAL '1 month')
		`).Scan(&resp.PaidThisMonth)
		_ = db.QueryRow(ctx, `
			SELECT COALESCE(SUM(total_cents), 0)
			FROM billing_invoices
			WHERE status IN ('open', 'uncollectible')
		`).Scan(&resp.OutstandingAmount)
		_ = db.QueryRow(ctx, `
			SELECT COALESCE(COUNT(*), 0)
			FROM billing_invoices
			WHERE status = 'open' AND due_at IS NOT NULL AND due_at < NOW()
		`).Scan(&resp.OverdueInvoices)
		_ = db.QueryRow(ctx, `
			SELECT COALESCE(COUNT(*), 0)
			FROM billing_invoices
			WHERE status = 'open'
		`).Scan(&resp.OpenInvoices)

		return c.JSON(resp)
	}
}

// ListBillingInvoices returns invoice list with tenant details.
func ListBillingInvoices(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 100)
		offset := c.QueryInt("offset", 0)
		status := strings.TrimSpace(c.Query("status"))
		tenantID := strings.TrimSpace(c.Query("tenant_id"))
		if limit <= 0 || limit > 500 {
			limit = 100
		}
		if offset < 0 {
			offset = 0
		}

		ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
		defer cancel()

		args := []interface{}{limit, offset}
		where := "1=1"
		if status != "" {
			where += fmt.Sprintf(" AND bi.status::text = $%d", len(args)+1)
			args = append(args, status)
		}
		if tenantID != "" {
			where += fmt.Sprintf(" AND bi.tenant_id = $%d", len(args)+1)
			args = append(args, tenantID)
		}

		rows, err := db.Query(ctx, fmt.Sprintf(`
			SELECT
				bi.id, bi.tenant_id, t.name, bi.subscription_id, bi.invoice_number, bi.status::text,
				bi.period_start, bi.period_end, bi.subtotal_cents, bi.tax_cents, bi.total_cents,
				bi.due_at, bi.paid_at, bi.payment_reference, bi.created_at
			FROM billing_invoices bi
			JOIN tenants t ON t.id = bi.tenant_id
			WHERE %s
			ORDER BY bi.created_at DESC
			LIMIT $1 OFFSET $2
		`, where), args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list invoices"})
		}
		defer rows.Close()

		out := make([]billingInvoiceRow, 0, limit)
		for rows.Next() {
			var r billingInvoiceRow
			if err := rows.Scan(
				&r.ID, &r.TenantID, &r.TenantName, &r.SubscriptionID, &r.InvoiceNumber, &r.Status,
				&r.PeriodStart, &r.PeriodEnd, &r.SubtotalCents, &r.TaxCents, &r.TotalCents,
				&r.DueAt, &r.PaidAt, &r.PaymentReference, &r.CreatedAt,
			); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read invoices"})
			}
			out = append(out, r)
		}
		return c.JSON(out)
	}
}

// CreateBillingInvoice creates a manual invoice from current subscription terms.
func CreateBillingInvoice(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			TenantID    string  `json:"tenant_id"`
			PeriodStart string  `json:"period_start"`
			PeriodEnd   string  `json:"period_end"`
			TaxCents    *int    `json:"tax_cents"`
			DueAt       *string `json:"due_at"`
			Notes       *string `json:"notes"`
			Status      *string `json:"status"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if _, err := uuid.Parse(body.TenantID); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Valid tenant_id is required"})
		}
		ps, errPS := time.Parse("2006-01-02", body.PeriodStart)
		pe, errPE := time.Parse("2006-01-02", body.PeriodEnd)
		if errPS != nil || errPE != nil || pe.Before(ps) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Valid period_start and period_end are required"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
		defer cancel()

		var subscriptionID *uuid.UUID
		var seatCount int
		var baseAmount int
		var perSeatAmount int
		err := db.QueryRow(ctx, `
			SELECT id, seat_count, base_amount_cents, per_seat_amount_cents
			FROM billing_subscriptions
			WHERE tenant_id = $1
		`, body.TenantID).Scan(&subscriptionID, &seatCount, &baseAmount, &perSeatAmount)
		if err != nil {
			if err == pgx.ErrNoRows {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Subscription not configured for this tenant"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load subscription"})
		}

		subtotal := baseAmount + (perSeatAmount * seatCount)
		tax := 0
		if body.TaxCents != nil && *body.TaxCents >= 0 {
			tax = *body.TaxCents
		}
		total := subtotal + tax
		status := "open"
		if body.Status != nil && *body.Status != "" {
			status = *body.Status
		}

		var dueAt *time.Time
		if body.DueAt != nil && *body.DueAt != "" {
			parsed, err := time.Parse(time.RFC3339, *body.DueAt)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "due_at must be RFC3339"})
			}
			dueAt = &parsed
		}

		invoiceNum, err := generateInvoiceNumber()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate invoice number"})
		}

		var out billingInvoiceRow
		err = db.QueryRow(ctx, `
			INSERT INTO billing_invoices (
				tenant_id, subscription_id, invoice_number, status, period_start, period_end,
				subtotal_cents, tax_cents, total_cents, due_at, notes, created_at, updated_at
			)
			VALUES (
				$1, $2, $3, $4::billing_invoice_status, $5, $6,
				$7, $8, $9, $10, $11, NOW(), NOW()
			)
			RETURNING id, tenant_id, subscription_id, invoice_number, status::text, period_start, period_end,
				subtotal_cents, tax_cents, total_cents, due_at, paid_at, payment_reference, created_at
		`, body.TenantID, subscriptionID, invoiceNum, status, ps, pe, subtotal, tax, total, dueAt, body.Notes).Scan(
			&out.ID, &out.TenantID, &out.SubscriptionID, &out.InvoiceNumber, &out.Status, &out.PeriodStart, &out.PeriodEnd,
			&out.SubtotalCents, &out.TaxCents, &out.TotalCents, &out.DueAt, &out.PaidAt, &out.PaymentReference, &out.CreatedAt,
		)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to create invoice"})
		}

		_ = db.QueryRow(ctx, `SELECT name FROM tenants WHERE id = $1`, out.TenantID).Scan(&out.TenantName)
		return c.Status(fiber.StatusCreated).JSON(out)
	}
}

// MarkBillingInvoicePaid marks an invoice as paid with optional payment reference.
func MarkBillingInvoicePaid(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid invoice ID"})
		}

		var body struct {
			PaymentReference *string `json:"payment_reference"`
		}
		_ = c.BodyParser(&body)

		ctx, cancel := context.WithTimeout(c.Context(), 6*time.Second)
		defer cancel()

		tag, err := db.Exec(ctx, `
			UPDATE billing_invoices
			SET status = 'paid', paid_at = NOW(), payment_reference = COALESCE($2, payment_reference), updated_at = NOW()
			WHERE id = $1
		`, id, body.PaymentReference)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to mark invoice paid"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invoice not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func generateInvoiceNumber() (string, error) {
	max := big.NewInt(90000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("INV-%s-%05d", time.Now().UTC().Format("20060102"), n.Int64()+10000), nil
}
