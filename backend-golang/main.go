package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"enterprise-attendance-api/internal/config"
	"enterprise-attendance-api/internal/database"
	"enterprise-attendance-api/internal/mqtt"
	"enterprise-attendance-api/internal/router"
	"enterprise-attendance-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize MQTT client for IoT door relays
	mqttClient, err := mqtt.NewClient(cfg.MQTTBrokerURL, cfg.MQTTClientID)
	if err != nil {
		log.Printf("Warning: Failed to initialize MQTT client: %v", err)
		log.Println("IoT door relay functionality will be disabled")
		mqttClient = nil
	}
	if mqttClient != nil {
		defer mqttClient.Disconnect()
	}

	// Initialize services
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpiry)
	attendanceService := services.NewAttendanceService(
		db.Pool,
		mqttClient,
		cfg.AIServiceURL,
		cfg.AIServiceAPIKey,
		cfg.OfflinePrivateKeyPEM,
		cfg.FaceMatchThreshold,
		cfg.AIVectorizeTimeout,
		cfg.AILivenessTimeout,
		cfg.AICompareTimeout,
		cfg.AIPinTimeout,
	)
	if !attendanceService.IsOfflineDecryptionConfigured() {
		log.Println("Warning: offline decryption key not loaded; /api/v1/kiosk/offline/sync will return 501")
		log.Println("Set OFFLINE_PRIVATE_KEY_PEM or OFFLINE_PRIVATE_KEY_PATH (example: ../keys/kiosk_offline_private.pem)")
	}
	userService := services.NewUserService(db.Pool)
	hrmsService := services.NewHRMSService(db.Pool)
	auditService := services.NewAuditService(db.Pool)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Enterprise Attendance API",
		ServerHeader: "Enterprise-Attendance",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.CORSOrigins, ","),
		AllowCredentials: true,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-API-Key,X-Tenant-ID,X-Kiosk-Code,X-HMAC-Signature,X-Timestamp",
	}))

	// Setup routes
	router.SetupRoutes(app, &router.Services{
		Auth:       authService,
		Attendance: attendanceService,
		User:       userService,
		HRMS:       hrmsService,
		Audit:      auditService,
	}, cfg)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down server...")
		app.Shutdown()
	}()

	// Start server
	log.Printf("Server starting on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
