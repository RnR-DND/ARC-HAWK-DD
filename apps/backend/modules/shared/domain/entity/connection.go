package entity

import (
	"time"

	"github.com/google/uuid"
)

// Connection represents a data source connection configuration
type Connection struct {
	ID               uuid.UUID              `json:"id"`
	TenantID         uuid.UUID              `json:"tenant_id"`
	SourceType       string                 `json:"source_type"`       // 'database', 'filesystem', 's3', 'gcs'
	ProfileName      string                 `json:"profile_name"`      // Unique name for this connection
	ConfigEncrypted  []byte                 `json:"-"`                 // Never serialize encrypted config
	Config           map[string]interface{} `json:"config,omitempty"`  // Decrypted config (only populated when needed)
	ValidationStatus string                 `json:"validation_status"` // 'pending', 'valid', 'invalid'
	LastValidatedAt  *time.Time             `json:"last_validated_at,omitempty"`
	ValidationError  *string                `json:"validation_error,omitempty"`
	CreatedBy        string                 `json:"created_by"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}
