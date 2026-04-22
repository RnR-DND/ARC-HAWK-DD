package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Policy defines a compliance automation rule: conditions that trigger it and
// actions to execute when triggered. Types: REMEDIATION, RETENTION, CONSENT.
type Policy struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	PolicyType  string          `json:"policy_type"`
	Conditions  json.RawMessage `json:"conditions"`
	Actions     json.RawMessage `json:"actions"`
	IsActive    bool            `json:"is_active"`
	CreatedBy   *string         `json:"created_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
