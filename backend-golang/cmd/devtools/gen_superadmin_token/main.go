package main

import (
	"fmt"
	"time"

	"enterprise-attendance-api/internal/config"
	"enterprise-attendance-api/internal/services"

	"github.com/google/uuid"
)

func main() {
	cfg := config.Load()
	auth := services.NewAuthService(nil, cfg.JWTSecret, 24*time.Hour)

	userID := uuid.New().String()
	tenantID := uuid.New().String()

	token, err := auth.GenerateToken(userID, tenantID, "super_admin", "superadmin@example.com")
	if err != nil {
		panic(err)
	}

	fmt.Println(token)
}
