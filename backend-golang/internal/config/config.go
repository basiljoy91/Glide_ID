package config

import (
	"os"
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
	JWTSecret  string
	JWTExpiry  time.Duration
	JWTIssuer  string

	// SSO Configuration
	SSOProvider     string // "saml", "oidc", "google", "azure"
	SSOClientID     string
	SSOClientSecret string
	SSOIssuerURL    string
	SSORedirectURL  string

	// AI Service
	AIServiceURL    string
	AIServiceAPIKey string

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
	KioskHMACSecret string

	// HRMS Webhooks
	HRMSWebhookSecret string
}

func Load() *Config {
	// Load .env file if it exists
	godotenv.Load()

	config := &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		JWTSecret:  getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiry:  parseDuration(getEnv("JWT_EXPIRY", "24h")),
		JWTIssuer:  getEnv("JWT_ISSUER", "enterprise-attendance-api"),

		SSOProvider:     getEnv("SSO_PROVIDER", ""),
		SSOClientID:     getEnv("SSO_CLIENT_ID", ""),
		SSOClientSecret: getEnv("SSO_CLIENT_SECRET", ""),
		SSOIssuerURL:    getEnv("SSO_ISSUER_URL", ""),
		SSORedirectURL:  getEnv("SSO_REDIRECT_URL", ""),

		AIServiceURL:    getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		AIServiceAPIKey:  getEnv("AI_SERVICE_API_KEY", ""),

		MQTTBrokerURL: getEnv("MQTT_BROKER_URL", ""),
		MQTTClientID:  getEnv("MQTT_CLIENT_ID", "attendance-api"),
		MQTTUsername:  getEnv("MQTT_USERNAME", ""),
		MQTTPassword:  getEnv("MQTT_PASSWORD", ""),

		CORSOrigins: parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000")),

		EncryptionKey:   getEnv("ENCRYPTION_KEY", ""),
		KioskHMACSecret: getEnv("KIOSK_HMAC_SECRET", ""),

		HRMSWebhookSecret: getEnv("HRMS_WEBHOOK_SECRET", ""),
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

