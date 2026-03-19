package handlers

import (
	"context"
	"time"

	"enterprise-attendance-api/internal/middleware"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetOrgBillingOverview(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		var planTier, status, billingCycle string
		var seatCount, activeUsers int
		var baseAmount, perSeatAmount int
		var nextInvoiceAt, currentPeriodEnd *time.Time
		err := db.QueryRow(ctx, `
			SELECT
				bs.plan_tier::text,
				bs.status::text,
				bs.billing_cycle,
				bs.seat_count,
				bs.base_amount_cents,
				bs.per_seat_amount_cents,
				bs.next_invoice_at,
				bs.current_period_end,
				COALESCE((SELECT COUNT(*) FROM users u WHERE u.tenant_id = bs.tenant_id AND u.deleted_at IS NULL), 0) AS active_users
			FROM billing_subscriptions bs
			WHERE bs.tenant_id = $1
		`, tenantID).Scan(&planTier, &status, &billingCycle, &seatCount, &baseAmount, &perSeatAmount, &nextInvoiceAt, &currentPeriodEnd, &activeUsers)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load billing overview"})
		}

		resp := fiber.Map{
			"plan_tier":               planTier,
			"status":                  status,
			"billing_cycle":           billingCycle,
			"seat_count":              seatCount,
			"active_users":            activeUsers,
			"overage":                 maxInt(activeUsers-seatCount, 0),
			"base_amount_cents":       baseAmount,
			"per_seat_amount_cents":   perSeatAmount,
			"projected_total_cents":   baseAmount + (perSeatAmount * seatCount),
			"projected_overage_cents": perSeatAmount * maxInt(activeUsers-seatCount, 0),
		}
		if nextInvoiceAt != nil {
			resp["next_invoice_at"] = nextInvoiceAt.UTC().Format(time.RFC3339)
		}
		if currentPeriodEnd != nil {
			resp["current_period_end"] = currentPeriodEnd.UTC().Format(time.RFC3339)
		}
		return c.JSON(resp)
	}
}

func ListOrgBillingInvoices(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		status := c.Query("status")

		query := `
			SELECT id, invoice_number, status::text, period_start, period_end, subtotal_cents, tax_cents, total_cents, due_at, paid_at, payment_reference, created_at
			FROM billing_invoices
			WHERE tenant_id = $1
		`
		args := []interface{}{tenantID}
		if status != "" && status != "all" {
			query += " AND status = $2::billing_invoice_status"
			args = append(args, status)
		}
		query += " ORDER BY created_at DESC LIMIT 100"

		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load invoices"})
		}
		defer rows.Close()
		out := []fiber.Map{}
		for rows.Next() {
			var id uuid.UUID
			var invoiceNumber, invoiceStatus string
			var periodStart, periodEnd time.Time
			var subtotal, tax, total int
			var dueAt, paidAt *time.Time
			var paymentReference *string
			var createdAt time.Time
			if err := rows.Scan(&id, &invoiceNumber, &invoiceStatus, &periodStart, &periodEnd, &subtotal, &tax, &total, &dueAt, &paidAt, &paymentReference, &createdAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read invoices"})
			}
			item := fiber.Map{
				"id":                id,
				"invoice_number":    invoiceNumber,
				"status":            invoiceStatus,
				"period_start":      periodStart.Format("2006-01-02"),
				"period_end":        periodEnd.Format("2006-01-02"),
				"subtotal_cents":    subtotal,
				"tax_cents":         tax,
				"total_cents":       total,
				"created_at":        createdAt.UTC().Format(time.RFC3339),
				"payment_reference": paymentReference,
			}
			if dueAt != nil {
				item["due_at"] = dueAt.UTC().Format(time.RFC3339)
			}
			if paidAt != nil {
				item["paid_at"] = paidAt.UTC().Format(time.RFC3339)
			}
			out = append(out, item)
		}
		return c.JSON(out)
	}
}

func ListSupportTickets(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `
			SELECT id, category, priority, subject, description, status, created_at, resolved_at
			FROM support_tickets
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT 100
		`, tenantID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load support tickets"})
		}
		defer rows.Close()
		out := []fiber.Map{}
		for rows.Next() {
			var id uuid.UUID
			var category, priority, subject, description, status string
			var createdAt time.Time
			var resolvedAt *time.Time
			if err := rows.Scan(&id, &category, &priority, &subject, &description, &status, &createdAt, &resolvedAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read support tickets"})
			}
			item := fiber.Map{
				"id":          id,
				"category":    category,
				"priority":    priority,
				"subject":     subject,
				"description": description,
				"status":      status,
				"created_at":  createdAt.UTC().Format(time.RFC3339),
			}
			if resolvedAt != nil {
				item["resolved_at"] = resolvedAt.UTC().Format(time.RFC3339)
			}
			out = append(out, item)
		}
		return c.JSON(out)
	}
}

func CreateSupportTicket(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		actorUserID := middleware.GetUserID(c)
		var body struct {
			Category    string `json:"category"`
			Priority    string `json:"priority"`
			Subject     string `json:"subject"`
			Description string `json:"description"`
		}
		if err := c.BodyParser(&body); err != nil || body.Subject == "" || body.Description == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "subject and description are required"})
		}
		if body.Category == "" {
			body.Category = "general"
		}
		if body.Priority == "" {
			body.Priority = "normal"
		}
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		var ticketID uuid.UUID
		if err := db.QueryRow(ctx, `
			INSERT INTO support_tickets (tenant_id, submitted_by, category, priority, subject, description, status)
			VALUES ($1, $2, $3, $4, $5, $6, 'open')
			RETURNING id
		`, tenantID, actorUserID, body.Category, body.Priority, body.Subject, body.Description).Scan(&ticketID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create support ticket"})
		}
		actionURL := "/admin/org/support"
		_ = services.InsertNotification(ctx, db, tenantID, nil, "support.ticket_created", "Support ticket opened", body.Subject, "info", &actionURL)
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": ticketID, "success": true})
	}
}

func ListOrgNotifications(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := middleware.GetUserID(c)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `
			SELECT id, notification_type, title, body, severity, is_read, action_url, created_at, read_at
			FROM org_notifications
			WHERE tenant_id = $1 AND (user_id IS NULL OR user_id::text = $2)
			ORDER BY created_at DESC
			LIMIT 100
		`, tenantID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load notifications"})
		}
		defer rows.Close()
		out := []fiber.Map{}
		for rows.Next() {
			var id uuid.UUID
			var notificationType, title, body, severity string
			var isRead bool
			var actionURL *string
			var createdAt time.Time
			var readAt *time.Time
			if err := rows.Scan(&id, &notificationType, &title, &body, &severity, &isRead, &actionURL, &createdAt, &readAt); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read notifications"})
			}
			item := fiber.Map{
				"id":                id,
				"notification_type": notificationType,
				"title":             title,
				"body":              body,
				"severity":          severity,
				"is_read":           isRead,
				"action_url":        actionURL,
				"created_at":        createdAt.UTC().Format(time.RFC3339),
			}
			if readAt != nil {
				item["read_at"] = readAt.UTC().Format(time.RFC3339)
			}
			out = append(out, item)
		}
		return c.JSON(out)
	}
}

func MarkOrgNotificationRead(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		userID := middleware.GetUserID(c)
		notificationID := c.Params("id")
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		tag, err := db.Exec(ctx, `
			UPDATE org_notifications
			SET is_read = true, read_at = NOW()
			WHERE tenant_id = $1 AND id = $2 AND (user_id IS NULL OR user_id::text = $3)
		`, tenantID, notificationID, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update notification"})
		}
		if tag.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Notification not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
