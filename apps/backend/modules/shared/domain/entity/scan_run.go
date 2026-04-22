package entity

import (
	"time"

	"github.com/google/uuid"
)

// ScanRun represents a single scan execution
type ScanRun struct {
	ID              uuid.UUID              `json:"id"`
	TenantID        uuid.UUID              `json:"tenant_id"`
	ProfileName     string                 `json:"profile_name"`
	ScanStartedAt   time.Time              `json:"scan_started_at"`
	ScanCompletedAt time.Time              `json:"scan_completed_at"`
	Host            string                 `json:"host"`
	TotalFindings   int                    `json:"total_findings"`
	TotalAssets     int                    `json:"total_assets"`
	Status          string                 `json:"status"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CancelledAt     *time.Time             `json:"cancelled_at,omitempty"`
	CancelledBy     *string                `json:"cancelled_by,omitempty"`
}
