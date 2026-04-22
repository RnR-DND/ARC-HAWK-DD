package entity

import (
	"time"

	"github.com/google/uuid"
)

// Asset represents a normalized file or resource
type Asset struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	StableID        string                 `json:"stable_id"`
	AssetType       string                 `json:"asset_type"`
	Name            string                 `json:"name"`
	Path            string                 `json:"path"`
	DataSource      string                 `json:"data_source"`
	Host            string                 `json:"host"`
	Environment     string                 `json:"environment"`
	Owner           string                 `json:"owner"`
	SourceSystem    string                 `json:"source_system"`
	FileMetadata    map[string]interface{} `json:"file_metadata,omitempty"`
	RiskScore       int                    `json:"risk_score"`
	TotalFindings   int                    `json:"total_findings"`
	IsMasked        bool                   `json:"is_masked"`
	MaskedAt        *time.Time             `json:"masked_at,omitempty"`
	MaskingStrategy string                 `json:"masking_strategy,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       *time.Time             `json:"deleted_at,omitempty"`
}
