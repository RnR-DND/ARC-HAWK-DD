package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ConsentRecord tracks a data subject's consent for a specific PII type and purpose.
type ConsentRecord struct {
	ID                 uuid.UUID       `json:"id"`
	DataSubjectID      string          `json:"data_subject_id"`
	PIIType            string          `json:"pii_type"`
	ConsentGivenAt     time.Time       `json:"consent_given_at"`
	ConsentExpiresAt   *time.Time      `json:"consent_expires_at,omitempty"`
	ConsentWithdrawnAt *time.Time      `json:"consent_withdrawn_at,omitempty"`
	ConsentPurpose     string          `json:"consent_purpose"`
	ConsentSource      *string         `json:"consent_source,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}
