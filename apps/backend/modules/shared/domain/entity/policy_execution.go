package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PolicyExecution records one execution of a Policy against a Finding.
// Result values: SUCCESS, FAILED, SKIPPED.
type PolicyExecution struct {
	ID           uuid.UUID       `json:"id"`
	PolicyID     uuid.UUID       `json:"policy_id"`
	FindingID    uuid.UUID       `json:"finding_id"`
	ExecutedAt   time.Time       `json:"executed_at"`
	Result       string          `json:"result"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}
