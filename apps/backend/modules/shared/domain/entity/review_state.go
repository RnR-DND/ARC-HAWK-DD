package entity

import (
	"time"

	"github.com/google/uuid"
)

// ReviewState represents audit trail for finding reviews
type ReviewState struct {
	ID         uuid.UUID  `json:"id"`
	FindingID  uuid.UUID  `json:"finding_id"`
	Status     string     `json:"status"`
	ReviewedBy string     `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	Comments   string     `json:"comments,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
