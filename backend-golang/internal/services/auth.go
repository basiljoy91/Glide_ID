package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthService struct {
	db        *pgxpool.Pool
	jwtSecret string
	jwtExpiry time.Duration
	jwtIssuer string
}

func NewAuthService(db *pgxpool.Pool, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
		jwtIssuer: "enterprise-attendance-api",
	}
}

type TokenClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func (s *AuthService) GenerateToken(userID, tenantID, role, email string) (string, error) {
	claims := TokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.jwtIssuer,
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// ValidateToken validates a JWT token
func (s *AuthService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

func (s *AuthService) GenerateTokenWithMetadata(userID, tenantID, role, email string, expiry time.Duration) (string, TokenClaims, error) {
	if expiry <= 0 {
		expiry = s.jwtExpiry
	}
	now := time.Now()
	claims := TokenClaims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.jwtIssuer,
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.jwtSecret))
	return signed, claims, err
}

func (s *AuthService) CreateSession(ctx context.Context, tenantID, userID, jti, ipAddress, userAgent string, expiresAt time.Time) (uuid.UUID, error) {
	sessionID := uuid.New()
	_, err := s.db.Exec(ctx, `
		INSERT INTO auth_sessions (id, tenant_id, user_id, token_jti, ip_address, user_agent, last_seen_at, expires_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::inet, NULLIF($6, ''), NOW(), $7)
	`, sessionID, tenantID, userID, jti, ipAddress, userAgent, expiresAt)
	return sessionID, err
}

func (s *AuthService) ValidateSession(ctx context.Context, jti, userID, tenantID string) error {
	var sessionID uuid.UUID
	err := s.db.QueryRow(ctx, `
		SELECT id
		FROM auth_sessions
		WHERE token_jti = $1
		  AND user_id = $2
		  AND tenant_id = $3
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`, jti, userID, tenantID).Scan(&sessionID)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(ctx, `UPDATE auth_sessions SET last_seen_at = NOW() WHERE id = $1`, sessionID)
	return nil
}

func (s *AuthService) CreateMFAChallenge(ctx context.Context, tenantID, userID, email, ipAddress string, ttl time.Duration) (uuid.UUID, string, time.Time, error) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	code, err := generateMFACode()
	if err != nil {
		return uuid.Nil, "", time.Time{}, err
	}
	challengeID := uuid.New()
	expiresAt := time.Now().UTC().Add(ttl)
	_, err = s.db.Exec(ctx, `
		INSERT INTO auth_mfa_challenges (id, tenant_id, user_id, email, code_hash, expires_at, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::inet)
	`, challengeID, tenantID, userID, email, hashChallengeCode(code), expiresAt, ipAddress)
	if err != nil {
		return uuid.Nil, "", time.Time{}, err
	}
	return challengeID, code, expiresAt, nil
}

func (s *AuthService) VerifyMFAChallenge(ctx context.Context, challengeID, code string) (string, string, string, string, error) {
	var userID, tenantID, role, email string
	var codeHash string
	var consumedAt *time.Time
	var expiresAt time.Time
	var attempts int
	err := s.db.QueryRow(ctx, `
		SELECT u.id::text, u.tenant_id::text, u.role::text, u.email, c.code_hash, c.consumed_at, c.expires_at, c.attempts
		FROM auth_mfa_challenges c
		JOIN users u ON u.id = c.user_id
		WHERE c.id = $1
	`, challengeID).Scan(&userID, &tenantID, &role, &email, &codeHash, &consumedAt, &expiresAt, &attempts)
	if err != nil {
		return "", "", "", "", err
	}
	if consumedAt != nil || time.Now().UTC().After(expiresAt) || attempts >= 5 {
		return "", "", "", "", fmt.Errorf("challenge expired or invalid")
	}
	if codeHash != hashChallengeCode(code) {
		_, _ = s.db.Exec(ctx, `UPDATE auth_mfa_challenges SET attempts = attempts + 1 WHERE id = $1`, challengeID)
		return "", "", "", "", fmt.Errorf("invalid verification code")
	}
	_, err = s.db.Exec(ctx, `UPDATE auth_mfa_challenges SET consumed_at = NOW() WHERE id = $1`, challengeID)
	if err != nil {
		return "", "", "", "", err
	}
	return userID, tenantID, role, email, nil
}

func generateMFACode() (string, error) {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	value := (uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])) % 1000000
	return fmt.Sprintf("%06s", strconv.Itoa(int(value))), nil
}
