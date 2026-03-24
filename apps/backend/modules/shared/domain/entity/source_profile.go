package entity

import (
	"time"

	"github.com/google/uuid"
)

// SourceProfile represents a scanner configuration profile
type SourceProfile struct {
	ID             uuid.UUID              `json:"id"`
	TenantID       uuid.UUID              `json:"tenant_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	DataSourceType string                 `json:"data_source_type"`
	Configuration  map[string]interface{} `json:"configuration,omitempty"`
	IsActive       bool                   `json:"is_active"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}
