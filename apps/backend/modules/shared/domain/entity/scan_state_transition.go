package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ScanStateTransition records a single state-machine step for a ScanRun.
type ScanStateTransition struct {
	ID             uuid.UUID       `json:"id"`
	ScanRunID      uuid.UUID       `json:"scan_run_id"`
	FromState      *string         `json:"from_state,omitempty"`
	ToState        string          `json:"to_state"`
	TransitionedAt time.Time       `json:"transitioned_at"`
	TransitionedBy *string         `json:"transitioned_by,omitempty"`
	Reason         *string         `json:"reason,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}
