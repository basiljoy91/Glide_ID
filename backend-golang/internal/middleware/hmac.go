package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HMACAuth middleware verifies HMAC signatures for kiosk requests
func HMACAuth(db *pgxpool.Pool, maxSkewSeconds int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		if path != "/api/v1/kiosk" && !strings.HasPrefix(path, "/api/v1/kiosk/") {
			return c.Next()
		}

		// Get kiosk code from header
		kioskCode := c.Get("X-Kiosk-Code")
		if kioskCode == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Kiosk code required",
			})
		}

		// Get HMAC signature from header
		signature := c.Get("X-HMAC-Signature")
		if signature == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "HMAC signature required",
			})
		}

		// Get timestamp from header (for replay attack prevention)
		timestampStr := c.Get("X-Timestamp")
		if timestampStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Timestamp required",
			})
		}

		// Verify timestamp (prevent replay attacks)
		var timestamp int64
		if _, err := fmt.Sscanf(timestampStr, "%d", &timestamp); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid timestamp format",
			})
		}

		// Check if timestamp is within acceptable window
		now := time.Now().Unix()
		if maxSkewSeconds <= 0 {
			maxSkewSeconds = 300
		}
		if abs(now-timestamp) > int64(maxSkewSeconds) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Request timestamp expired",
			})
		}

		// Get kiosk HMAC secret from database
		var kioskID uuid.UUID
		var tenantID uuid.UUID
		var hmacSecret string
		var status string

		err := db.QueryRow(c.Context(), `
			SELECT id, tenant_id, hmac_secret, status
			FROM kiosks
			WHERE code = $1 AND status = 'active'
		`, kioskCode).Scan(&kioskID, &tenantID, &hmacSecret, &status)

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid kiosk code",
			})
		}

		// Verify HMAC signature
		// Signature format: HMAC-SHA256(body + timestamp + kiosk_code)
		body := string(c.Body())
		message := fmt.Sprintf("%s%s%s", body, timestampStr, kioskCode)

		mac := hmac.New(sha256.New, []byte(hmacSecret))
		mac.Write([]byte(message))
		expectedSignature := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid HMAC signature",
			})
		}

		// Store kiosk info in context
		c.Locals("kiosk_code", kioskCode)
		c.Locals("tenant_id", tenantID.String())
		c.Locals("kiosk_id", kioskID.String())

		return c.Next()
	}
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
