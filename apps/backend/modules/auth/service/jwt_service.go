package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"time"

	"github.com/arc-platform/backend/modules/auth/entity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrTokenExpired  = errors.New("token expired")
	ErrInvalidClaims = errors.New("invalid claims")
)

type JWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TenantID  string `json:"tenant_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey     []byte
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
	db            *sql.DB // C-2: persistent blacklist storage
	stop          chan struct{}
}

// NewJWTService creates a JWT service. Pass db for persistent token blacklist (C-2).
// If db is nil, blacklist operations are no-ops (for tests).
func NewJWTService(db *sql.DB) *JWTService {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		if os.Getenv("GIN_MODE") == "release" {
			// H-7: use log.Fatal instead of panic for cleaner shutdown
			log.Fatal("FATAL: JWT_SECRET environment variable is required in production mode")
		}
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			log.Fatalf("Failed to generate random JWT secret: %v", err)
		}
		secretKey = base64.StdEncoding.EncodeToString(randomBytes)
		log.Println("⚠️  WARNING: Using auto-generated JWT secret. Set JWT_SECRET env var for persistent sessions.")
	}

	svc := &JWTService{
		secretKey:     []byte(secretKey),
		tokenExpiry:   24 * time.Hour,
		refreshExpiry: 7 * 24 * time.Hour,
		db:            db,
		stop:          make(chan struct{}),
	}

	// Start background cleanup of expired blacklist entries
	if db != nil {
		go svc.cleanupExpiredTokens()
	}

	return svc
}

func (s *JWTService) GenerateToken(user *entity.User, sessionID uuid.UUID) (string, string, error) {
	now := time.Now()
	expiresAt := now.Add(s.tokenExpiry)
	refreshExpiresAt := now.Add(s.refreshExpiry)

	claims := JWTClaims{
		UserID:    user.ID.String(),
		Email:     user.Email,
		Role:      string(user.Role),
		TenantID:  user.TenantID.String(),
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "arc-hawk",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", "", err
	}

	refreshClaims := JWTClaims{
		UserID:    user.ID.String(),
		Email:     user.Email,
		Role:      string(user.Role),
		TenantID:  user.TenantID.String(),
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "arc-hawk-refresh",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.secretKey)
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshTokenString, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// C-2: Check persistent blacklist
	if s.isTokenBlacklisted(tokenString) {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

func (s *JWTService) ValidateRefreshToken(refreshTokenString string) (*JWTClaims, error) {
	if s.isTokenBlacklisted(refreshTokenString) {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(refreshTokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *JWTService) GenerateResetToken(userID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	claims := JWTClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "arc-hawk-reset",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (s *JWTService) ValidateResetToken(tokenString string) (uuid.UUID, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, err
	}

	if claims.Issuer != "arc-hawk-reset" {
		return uuid.Nil, ErrInvalidToken
	}

	return uuid.Parse(claims.UserID)
}

func (s *JWTService) InvalidateToken(tokenString string) error {
	if s.db == nil {
		return nil
	}
	hash := HashToken(tokenString)
	// Store with TTL matching the longest token expiry (refresh = 7 days)
	expiresAt := time.Now().Add(s.refreshExpiry)
	_, err := s.db.ExecContext(context.Background(),
		`INSERT INTO token_blacklist (token_hash, expires_at) VALUES ($1, $2) ON CONFLICT (token_hash) DO NOTHING`,
		hash, expiresAt)
	return err
}

// isTokenBlacklisted checks the persistent blacklist
func (s *JWTService) isTokenBlacklisted(tokenString string) bool {
	if s.db == nil {
		return false
	}
	hash := HashToken(tokenString)
	var exists bool
	err := s.db.QueryRowContext(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM token_blacklist WHERE token_hash = $1 AND expires_at > NOW())`,
		hash).Scan(&exists)
	if err != nil {
		log.Printf("WARN: blacklist check failed: %v", err)
		return false
	}
	return exists
}

// cleanupExpiredTokens periodically removes expired entries from the blacklist
func (s *JWTService) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if s.db != nil {
				if _, err := s.db.ExecContext(context.Background(),
					`DELETE FROM token_blacklist WHERE expires_at < NOW()`); err != nil {
					log.Printf("WARN: token_blacklist cleanup failed: %v", err)
				}
			}
		case <-s.stop:
			return
		}
	}
}

// StopCleanup shuts down the background cleanup goroutine.
func (s *JWTService) StopCleanup() {
	close(s.stop)
}
