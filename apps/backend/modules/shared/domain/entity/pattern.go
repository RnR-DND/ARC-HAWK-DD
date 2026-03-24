package entity

import (
	"time"

	"github.com/google/uuid"
)

// Pattern represents a detection pattern definition
type Pattern struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	PatternType       string    `json:"pattern_type"`
	Category          string    `json:"category"`
	Description       string    `json:"description"`
	PatternDefinition string    `json:"pattern_definition"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
