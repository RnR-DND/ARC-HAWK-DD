package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"sync"
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
	blacklist     sync.Map
}

func NewJWTService() *JWTService {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		// SECURITY: In production, JWT_SECRET must be set
		// Generate a random key for development only
		if os.Getenv("GIN_MODE") == "release" {
			// In production mode, fail if no secret is set
			panic("FATAL: JWT_SECRET environment variable is required in production mode")
		}
		// Development mode: generate a random secret and warn
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			panic("Failed to generate random JWT secret: " + err.Error())
		}
		secretKey = base64.StdEncoding.EncodeToString(randomBytes)
		// Log warning - this secret is ephemeral and will change on restart
		println("⚠️  WARNING: Using auto-generated JWT secret. Set JWT_SECRET env var for persistent sessions.")
	}

	return &JWTService{
		secretKey:     []byte(secretKey),
		tokenExpiry:   24 * time.Hour,
		refreshExpiry: 7 * 24 * time.Hour,
	}
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
	// Reject blacklisted tokens
	if _, revoked := s.blacklist.Load(tokenString); revoked {
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
	s.blacklist.Store(tokenString, struct{}{})
	return nil
}
