package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port        string
	Environment string

	// Database
	DatabaseURL string

	// JWT Authentication
	JWTSecret string
	JWTExpiry time.Duration
	JWTIssuer string

	// SSO Configuration
	SSOProvider     string // "saml", "oidc", "google", "azure"
	SSOClientID     string
	SSOClientSecret string
	SSOIssuerURL    string
	SSORedirectURL  string

	// AI Service
	AIServiceURL       string
	AIServiceAPIKey    string
	FaceMatchThreshold float64
	AIVectorizeTimeout time.Duration
	AILivenessTimeout  time.Duration
	AICompareTimeout   time.Duration
	AIPinTimeout       time.Duration

	// MQTT for IoT Door Relays
	MQTTBrokerURL string
	MQTTClientID  string
	MQTTUsername  string
	MQTTPassword  string

	// CORS
	CORSOrigins []string

	// Encryption
	EncryptionKey string

	// HMAC for Kiosk
	KioskHMACSecret    string
	HMACMaxSkewSeconds int

	// Offline encryption (private key used server-side to decrypt kiosk offline payloads)
	OfflinePrivateKeyPEM string

	// HRMS Webhooks
	HRMSWebhookSecret string

	// Email (Brevo)
	EmailProvider           string
	BrevoAPIKey             string
	EmailFrom               string
	EmailFromName           string
	ReportSchedulerInterval time.Duration
}

func Load() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		JWTSecret: getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiry: parseDuration(getEnv("JWT_EXPIRY", "24h")),
		JWTIssuer: getEnv("JWT_ISSUER", "enterprise-attendance-api"),

		SSOProvider:     getEnv("SSO_PROVIDER", ""),
		SSOClientID:     getEnv("SSO_CLIENT_ID", ""),
		SSOClientSecret: getEnv("SSO_CLIENT_SECRET", ""),
		SSOIssuerURL:    getEnv("SSO_ISSUER_URL", ""),
		SSORedirectURL:  getEnv("SSO_REDIRECT_URL", ""),

		AIServiceURL:       getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		AIServiceAPIKey:    getEnv("AI_SERVICE_API_KEY", ""),
		FaceMatchThreshold: parseFloat(getEnv("FACE_MATCH_THRESHOLD", "0.62"), 0.62),
		AIVectorizeTimeout: parseDuration(getEnv("AI_VECTORIZE_TIMEOUT", "90s")),
		AILivenessTimeout:  parseDuration(getEnv("AI_LIVENESS_TIMEOUT", "15s")),
		AICompareTimeout:   parseDuration(getEnv("AI_COMPARE_TIMEOUT", "20s")),
		AIPinTimeout:       parseDuration(getEnv("AI_PIN_TIMEOUT", "8s")),

		MQTTBrokerURL: getEnv("MQTT_BROKER_URL", ""),
		MQTTClientID:  getEnv("MQTT_CLIENT_ID", "attendance-api"),
		MQTTUsername:  getEnv("MQTT_USERNAME", ""),
		MQTTPassword:  getEnv("MQTT_PASSWORD", ""),

		CORSOrigins: parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000")),

		EncryptionKey:      getEnv("ENCRYPTION_KEY", ""),
		KioskHMACSecret:    getEnv("KIOSK_HMAC_SECRET", ""),
		HMACMaxSkewSeconds: parseInt(getEnv("HMAC_MAX_SKEW_SECONDS", "300"), 300),

		OfflinePrivateKeyPEM: loadOfflinePrivateKeyPEM(),

		HRMSWebhookSecret: getEnv("HRMS_WEBHOOK_SECRET", ""),

		EmailProvider:           strings.ToLower(getEnv("EMAIL_PROVIDER", "")),
		BrevoAPIKey:             getEnv("BREVO_API_KEY", ""),
		EmailFrom:               getEnv("EMAIL_FROM", ""),
		EmailFromName:           getEnv("EMAIL_FROM_NAME", "Glide ID Reports"),
		ReportSchedulerInterval: parseDuration(getEnv("REPORT_SCHEDULER_INTERVAL", "60s")),
	}

	// Validate required config
	if config.DatabaseURL == "" {
		panic("DATABASE_URL is required")
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour // Default to 24 hours
	}
	return d
}

func parseCORSOrigins(s string) []string {
	if s == "" {
		return []string{"*"}
	}
	return strings.Split(s, ",")
}

func parseInt(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}

func parseFloat(s string, def float64) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return n
}

func loadOfflinePrivateKeyPEM() string {
	// 1) Prefer direct PEM in env (current behavior).
	if pem := strings.TrimSpace(getEnv("OFFLINE_PRIVATE_KEY_PEM", "")); pem != "" {
		return strings.ReplaceAll(pem, `\\n`, "\n")
	}

	// 2) Optional explicit file path env for easier local/dev setup.
	// Example: OFFLINE_PRIVATE_KEY_PATH=../keys/kiosk_offline_private.pem
	pathCandidates := []string{}
	if p := strings.TrimSpace(getEnv("OFFLINE_PRIVATE_KEY_PATH", "")); p != "" {
		pathCandidates = append(pathCandidates, p)
	}

	// 3) Sensible defaults for this repository layout.
	pathCandidates = append(pathCandidates,
		"../keys/kiosk_offline_private.pem",
		"./keys/kiosk_offline_private.pem",
	)

	for _, p := range pathCandidates {
		clean := filepath.Clean(p)
		b, err := os.ReadFile(clean)
		if err != nil {
			continue
		}
		if len(b) == 0 {
			continue
		}
		return string(b)
	}

	return ""
}
