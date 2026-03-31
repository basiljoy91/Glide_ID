package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthService struct {
	db        *pgxpool.Pool
	jwtSecret string
	jwtExpiry time.Duration
	jwtIssuer string
}

type authSessionStore interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type emailChallengeStore interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
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
	sessionID, err := validateActiveSession(ctx, s.db, jti, userID, tenantID)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(ctx, `UPDATE auth_sessions SET last_seen_at = NOW() WHERE id = $1`, sessionID)
	return nil
}

func validateActiveSession(ctx context.Context, store authSessionStore, jti, userID, tenantID string) (uuid.UUID, error) {
	var sessionID uuid.UUID
	err := store.QueryRow(ctx, `
		SELECT s.id
		FROM auth_sessions s
		JOIN tenants t ON t.id = s.tenant_id
		JOIN users u ON u.id = s.user_id
		WHERE s.token_jti = $1
		  AND s.user_id = $2
		  AND s.tenant_id = $3
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
		  AND t.deleted_at IS NULL
		  AND u.deleted_at IS NULL
		  AND u.is_active = true
	`, jti, userID, tenantID).Scan(&sessionID)
	if err != nil {
		return uuid.Nil, err
	}
	return sessionID, nil
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

func (s *AuthService) CreateEmailVerificationChallenge(ctx context.Context, email, scope string, ttl time.Duration, ipAddress string) (uuid.UUID, string, time.Time, error) {
	return createEmailVerificationChallenge(ctx, s.db, email, scope, ttl, ipAddress)
}

func (s *AuthService) VerifyEmailVerificationChallenge(ctx context.Context, challengeID, email, code, scope string) error {
	return verifyEmailVerificationChallenge(ctx, s.db, challengeID, email, code, scope)
}

func (s *AuthService) ConsumeEmailVerificationChallenge(ctx context.Context, challengeID, email, scope string) error {
	return consumeEmailVerificationChallenge(ctx, s.db, challengeID, email, scope)
}

func (s *AuthService) ConsumeEmailVerificationChallengeTx(ctx context.Context, store emailChallengeStore, challengeID, email, scope string) error {
	return consumeEmailVerificationChallenge(ctx, store, challengeID, email, scope)
}

func (s *AuthService) CheckEmailVerificationRateLimit(ctx context.Context, email, scope string, cooldown, window time.Duration, maxChallenges int) (time.Duration, bool, error) {
	if cooldown <= 0 {
		cooldown = time.Minute
	}
	if window <= 0 {
		window = time.Hour
	}
	if maxChallenges <= 0 {
		maxChallenges = 5
	}

	var latestCreatedAt *time.Time
	var recentCount int
	err := s.db.QueryRow(ctx, `
		SELECT
			MAX(created_at) FILTER (
				WHERE consumed_at IS NULL
				  AND expires_at > NOW()
			) AS latest_created_at,
			COUNT(*) FILTER (
				WHERE created_at >= NOW() - ($3 || ' seconds')::interval
				  AND consumed_at IS NULL
			) AS recent_count
		FROM email_verification_challenges
		WHERE LOWER(email) = LOWER($1)
		  AND scope = $2
	`, email, scope, int(window.Seconds())).Scan(&latestCreatedAt, &recentCount)
	if err != nil {
		return 0, false, err
	}

	if recentCount >= maxChallenges {
		return 0, true, nil
	}
	if latestCreatedAt != nil {
		nextAllowedAt := latestCreatedAt.Add(cooldown)
		if nextAllowedAt.After(time.Now().UTC()) {
			return time.Until(nextAllowedAt), false, nil
		}
	}
	return 0, false, nil
}

func createEmailVerificationChallenge(ctx context.Context, store emailChallengeStore, email, scope string, ttl time.Duration, ipAddress string) (uuid.UUID, string, time.Time, error) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	code, err := generateMFACode()
	if err != nil {
		return uuid.Nil, "", time.Time{}, err
	}
	challengeID := uuid.New()
	expiresAt := time.Now().UTC().Add(ttl)
	_, err = store.Exec(ctx, `
		INSERT INTO email_verification_challenges (id, email, scope, code_hash, expires_at, ip_address)
		VALUES ($1, LOWER($2), $3, $4, $5, NULLIF($6, '')::inet)
	`, challengeID, email, scope, hashChallengeCode(code), expiresAt, ipAddress)
	if err != nil {
		return uuid.Nil, "", time.Time{}, err
	}
	return challengeID, code, expiresAt, nil
}

func verifyEmailVerificationChallenge(ctx context.Context, store emailChallengeStore, challengeID, email, code, scope string) error {
	var codeHash string
	var storedEmail string
	var storedScope string
	var expiresAt time.Time
	var verifiedAt *time.Time
	var consumedAt *time.Time
	var attempts int
	err := store.QueryRow(ctx, `
		SELECT email, scope, code_hash, expires_at, verified_at, consumed_at, attempts
		FROM email_verification_challenges
		WHERE id = $1
	`, challengeID).Scan(&storedEmail, &storedScope, &codeHash, &expiresAt, &verifiedAt, &consumedAt, &attempts)
	if err != nil {
		return err
	}
	if storedScope != scope || storedEmail != email {
		return fmt.Errorf("challenge scope or email mismatch")
	}
	if consumedAt != nil || time.Now().UTC().After(expiresAt) || attempts >= 5 {
		return fmt.Errorf("challenge expired or invalid")
	}
	if codeHash != hashChallengeCode(code) {
		_, _ = store.Exec(ctx, `UPDATE email_verification_challenges SET attempts = attempts + 1 WHERE id = $1`, challengeID)
		return fmt.Errorf("invalid verification code")
	}
	if verifiedAt != nil {
		return nil
	}
	_, err = store.Exec(ctx, `UPDATE email_verification_challenges SET verified_at = NOW() WHERE id = $1`, challengeID)
	return err
}

func consumeEmailVerificationChallenge(ctx context.Context, store emailChallengeStore, challengeID, email, scope string) error {
	tag, err := store.Exec(ctx, `
		UPDATE email_verification_challenges
		SET consumed_at = NOW()
		WHERE id = $1
		  AND LOWER(email) = LOWER($2)
		  AND scope = $3
		  AND verified_at IS NOT NULL
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
	`, challengeID, email, scope)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func generateMFACode() (string, error) {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	value := (uint32(raw[0])<<24 | uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])) % 1000000
	return fmt.Sprintf("%06s", strconv.Itoa(int(value))), nil
}
