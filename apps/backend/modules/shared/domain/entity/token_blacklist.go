package entity

import "time"

// TokenBlacklist represents a revoked JWT token.
// Tokens are identified by their SHA-256 hash; the plaintext JWT is never stored.
type TokenBlacklist struct {
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	RevokedAt time.Time `json:"revoked_at"`
}
