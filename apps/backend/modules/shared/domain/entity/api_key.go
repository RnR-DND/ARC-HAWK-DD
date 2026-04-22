package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// APIKey represents a service-to-service or CI/CD API key.
// The plain-text key is shown once at creation; only the SHA-256 hash is stored.
type APIKey struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	CreatedBy  uuid.UUID       `json:"created_by"`
	Name       string          `json:"name"`
	KeyHash    string          `json:"key_hash"`
	KeyPrefix  string          `json:"key_prefix"`
	Scopes     []string        `json:"scopes"`
	ExpiresAt  *time.Time      `json:"expires_at,omitempty"`
	LastUsedAt *time.Time      `json:"last_used_at,omitempty"`
	Revoked    bool            `json:"revoked"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	// Populated only when creating a new key; never read back from DB.
	PlainKey string `json:"plain_key,omitempty"`
	// Metadata holds any extra information attached to the key.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}
