package entity

import (
	"time"

	"github.com/google/uuid"
)

// Classification represents PII classification
type Classification struct {
	ID                 uuid.UUID              `json:"id"`
	FindingID          uuid.UUID              `json:"finding_id"`
	ClassificationType string                 `json:"classification_type"`
	SubCategory        string                 `json:"sub_category,omitempty"`
	ConfidenceScore    float64                `json:"confidence_score"`
	Justification      string                 `json:"justification"`
	DPDPACategory      string                 `json:"dpdpa_category,omitempty"`
	RequiresConsent    bool                   `json:"requires_consent"`
	RetentionPeriod    string                 `json:"retention_period,omitempty"`
	SignalBreakdown    map[string]interface{} `json:"signal_breakdown,omitempty"`
	EngineVersion      string                 `json:"engine_version,omitempty"`
	RuleScore          *float64               `json:"rule_score,omitempty"`
	PresidioScore      *float64               `json:"presidio_score,omitempty"`
	ContextScore       *float64               `json:"context_score,omitempty"`
	EntropyScore       *float64               `json:"entropy_score,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}
