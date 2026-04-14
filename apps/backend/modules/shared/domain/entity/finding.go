package entity

import (
	"time"

	"github.com/google/uuid"
)

// Finding represents an individual PII or secret detection
type Finding struct {
	ID                  uuid.UUID              `json:"id"`
	TenantID            uuid.UUID              `json:"tenant_id"`
	ScanRunID           uuid.UUID              `json:"scan_run_id"`
	AssetID             uuid.UUID              `json:"asset_id"`
	PatternID           *uuid.UUID             `json:"pattern_id,omitempty"`
	PatternName         string                 `json:"pattern_name"`
	Matches             []string               `json:"matches"`
	MaskedValue         string                 `json:"masked_value,omitempty"`
	SampleText          string                 `json:"sample_text"`
	NormalizedValueHash string                 `json:"normalized_value_hash,omitempty"`
	Severity            string                 `json:"severity"`
	SeverityDescription string                 `json:"severity_description"`
	ConfidenceScore     *float64               `json:"confidence_score,omitempty"`
	Environment         string                 `json:"environment"` // "PROD" or "TEST"
	Context             map[string]interface{} `json:"context,omitempty"`
	EnrichmentSignals   map[string]interface{} `json:"enrichment_signals,omitempty"`
	EnrichmentScore     *float64               `json:"enrichment_score,omitempty"`
	EnrichmentFailed    bool                   `json:"enrichment_failed"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}
