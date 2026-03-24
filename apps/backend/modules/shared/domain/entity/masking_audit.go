package entity

import (
	"time"

	"github.com/google/uuid"
)

// MaskingAudit represents an audit log entry for masking operations
type MaskingAudit struct {
	ID              uuid.UUID              `json:"id"`
	AssetID         uuid.UUID              `json:"asset_id"`
	MaskedBy        string                 `json:"masked_by,omitempty"`
	MaskingStrategy string                 `json:"masking_strategy"`
	FindingsCount   int                    `json:"findings_count"`
	MaskedAt        time.Time              `json:"masked_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}
