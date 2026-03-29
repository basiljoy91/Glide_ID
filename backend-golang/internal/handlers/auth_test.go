package handlers

import (
	"context"
	"errors"
	"testing"

	"enterprise-attendance-api/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestVerifyLoginPassword(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("accepts valid bcrypt hash", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("CorrectHorseBatteryStaple!1"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("GenerateFromPassword() error = %v", err)
		}
		user := &models.User{ID: userID, TenantID: tenantID, PasswordHash: testStringPtr(string(hash))}

		if err := verifyLoginPassword(ctx, user, "CorrectHorseBatteryStaple!1", nil); err != nil {
			t.Fatalf("verifyLoginPassword() error = %v", err)
		}
	})

	t.Run("rejects mismatched bcrypt password", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("CorrectHorseBatteryStaple!1"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("GenerateFromPassword() error = %v", err)
		}
		user := &models.User{ID: userID, TenantID: tenantID, PasswordHash: testStringPtr(string(hash))}

		err = verifyLoginPassword(ctx, user, "WrongPassword!1", nil)
		if !errors.Is(err, errInvalidLoginPassword) {
			t.Fatalf("verifyLoginPassword() error = %v, want errInvalidLoginPassword", err)
		}
	})

	t.Run("migrates legacy plaintext password", func(t *testing.T) {
		user := &models.User{ID: userID, TenantID: tenantID, PasswordHash: testStringPtr("LegacyPass!123")}

		var upgradedTenantID string
		var upgradedUserID string
		var upgradedHash string
		err := verifyLoginPassword(ctx, user, "LegacyPass!123", func(_ context.Context, tenantID, userID, passwordHash string) error {
			upgradedTenantID = tenantID
			upgradedUserID = userID
			upgradedHash = passwordHash
			return nil
		})
		if err != nil {
			t.Fatalf("verifyLoginPassword() error = %v", err)
		}
		if upgradedTenantID != tenantID.String() || upgradedUserID != userID.String() {
			t.Fatalf("upgrade callback got tenant=%q user=%q", upgradedTenantID, upgradedUserID)
		}
		if upgradedHash == "" {
			t.Fatal("upgrade callback did not receive a replacement hash")
		}
		if errors.Is(bcrypt.CompareHashAndPassword([]byte(upgradedHash), []byte("LegacyPass!123")), bcrypt.ErrMismatchedHashAndPassword) {
			t.Fatal("replacement hash does not validate the password")
		}
	})

	t.Run("rejects malformed non-matching hash", func(t *testing.T) {
		user := &models.User{ID: userID, TenantID: tenantID, PasswordHash: testStringPtr("not-a-bcrypt-hash")}

		err := verifyLoginPassword(ctx, user, "CorrectHorseBatteryStaple!1", nil)
		if !errors.Is(err, errInvalidLoginPassword) {
			t.Fatalf("verifyLoginPassword() error = %v, want errInvalidLoginPassword", err)
		}
	})

	t.Run("surfaces upgrade persistence failure", func(t *testing.T) {
		user := &models.User{ID: userID, TenantID: tenantID, PasswordHash: testStringPtr("LegacyPass!123")}
		wantErr := errors.New("write failed")

		err := verifyLoginPassword(ctx, user, "LegacyPass!123", func(_ context.Context, _, _, _ string) error {
			return wantErr
		})
		if err == nil || !errors.Is(err, wantErr) {
			t.Fatalf("verifyLoginPassword() error = %v, want wrapped %v", err, wantErr)
		}
	})
}

func testStringPtr(value string) *string {
	return &value
}
