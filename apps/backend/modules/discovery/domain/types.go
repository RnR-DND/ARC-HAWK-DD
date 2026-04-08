// Package domain contains the discovery module's domain types — pure data, no behavior.
//
// These types map 1:1 to the discovery_* tables in migrations 000019-000024.
// Services and the repo layer operate on these types; handlers serialize them as JSON.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// SnapshotTrigger indicates how a snapshot was initiated.
type SnapshotTrigger string

const (
	TriggerManual SnapshotTrigger = "manual"
	TriggerCron   SnapshotTrigger = "cron"
)

// SnapshotStatus tracks the lifecycle of a snapshot job.
type SnapshotStatus string

const (
	SnapshotPending   SnapshotStatus = "pending"
	SnapshotRunning   SnapshotStatus = "running"
	SnapshotCompleted SnapshotStatus = "completed"
	SnapshotFailed    SnapshotStatus = "failed"
)

// DriftEventType is the closed enum of drift event categories matching the DB CHECK.
type DriftEventType string

const (
	DriftAssetAdded            DriftEventType = "asset_added"
	DriftAssetRemoved          DriftEventType = "asset_removed"
	DriftClassificationChanged DriftEventType = "classification_changed"
	DriftRiskIncreased         DriftEventType = "risk_increased"
	DriftRiskDecreased         DriftEventType = "risk_decreased"
	DriftFindingCountSpike     DriftEventType = "finding_count_spike"
)

// DriftSeverity classifies drift event impact.
type DriftSeverity string

const (
	SeverityLow      DriftSeverity = "low"
	SeverityMedium   DriftSeverity = "medium"
	SeverityHigh     DriftSeverity = "high"
	SeverityCritical DriftSeverity = "critical"
)

// ReportFormat is the supported set of board-report output formats.
//
// v1 supports html, csv, json natively. The "pdf" format is accepted but currently
// returns HTML with a Content-Type header that triggers the browser's print-to-PDF
// flow. v1.5 will add real server-side PDF generation via gofpdf.
type ReportFormat string

const (
	ReportPDF  ReportFormat = "pdf"
	ReportCSV  ReportFormat = "csv"
	ReportJSON ReportFormat = "json"
	ReportHTML ReportFormat = "html"
)

// ReportStatus tracks async report generation lifecycle.
type ReportStatus string

const (
	ReportPending   ReportStatus = "pending"
	ReportRunning   ReportStatus = "running"
	ReportCompleted ReportStatus = "completed"
	ReportFailed    ReportStatus = "failed"
)

// InventoryRow is one denormalized row in the discovery_inventory table.
// One row per (tenant, asset, classification).
type InventoryRow struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	AssetID        uuid.UUID  `json:"asset_id"`
	AssetName      string     `json:"asset_name"`
	SourceID       *uuid.UUID `json:"source_id,omitempty"`
	SourceName     string     `json:"source_name,omitempty"`
	Classification string     `json:"classification"`
	Sensitivity    int        `json:"sensitivity"`
	FindingCount   int        `json:"finding_count"`
	LastScannedAt  *time.Time `json:"last_scanned_at,omitempty"`
	RefreshedAt    time.Time  `json:"refreshed_at"`
}

// Snapshot is a header row in discovery_snapshots — one per snapshot run.
type Snapshot struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	TakenAt            time.Time       `json:"taken_at"`
	SourceCount        int             `json:"source_count"`
	AssetCount         int             `json:"asset_count"`
	FindingCount       int             `json:"finding_count"`
	HighRiskCount      int             `json:"high_risk_count"`
	CompositeRiskScore float64         `json:"composite_risk_score"`
	Trigger            SnapshotTrigger `json:"trigger"`
	TriggeredBy        *uuid.UUID      `json:"triggered_by,omitempty"`
	Status             SnapshotStatus  `json:"status"`
	Error              string          `json:"error,omitempty"`
	DurationMS         int64           `json:"duration_ms,omitempty"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
}

// SnapshotFact is one row in discovery_snapshot_facts — per-snapshot drilldown by source+classification.
type SnapshotFact struct {
	ID             uuid.UUID  `json:"id"`
	SnapshotID     uuid.UUID  `json:"snapshot_id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	SourceID       *uuid.UUID `json:"source_id,omitempty"`
	SourceName     string     `json:"source_name,omitempty"`
	Classification string     `json:"classification"`
	AssetCount     int        `json:"asset_count"`
	FindingCount   int        `json:"finding_count"`
	SensitivityAvg float64    `json:"sensitivity_avg"`
}

// RiskScore is one entry in discovery_risk_scores — per-asset composite score history.
type RiskScore struct {
	ID                  uuid.UUID              `json:"id"`
	TenantID            uuid.UUID              `json:"tenant_id"`
	AssetID             uuid.UUID              `json:"asset_id"`
	SnapshotID          *uuid.UUID             `json:"snapshot_id,omitempty"`
	Score               float64                `json:"score"`
	ContributingFactors map[string]interface{} `json:"contributing_factors"`
	ComputedAt          time.Time              `json:"computed_at"`
}

// DriftEvent is one entry in discovery_drift_events — what changed between two snapshots.
type DriftEvent struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	SnapshotID  uuid.UUID              `json:"snapshot_id"`
	EventType   DriftEventType         `json:"event_type"`
	AssetID     uuid.UUID              `json:"asset_id"`
	BeforeState map[string]interface{} `json:"before_state,omitempty"`
	AfterState  map[string]interface{} `json:"after_state,omitempty"`
	Severity    DriftSeverity          `json:"severity"`
	DetectedAt  time.Time              `json:"detected_at"`
}

// Report is one entry in discovery_reports — async generated board report.
type Report struct {
	ID          uuid.UUID    `json:"id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	SnapshotID  *uuid.UUID   `json:"snapshot_id,omitempty"`
	RequestedBy *uuid.UUID   `json:"requested_by,omitempty"`
	Format      ReportFormat `json:"format"`
	Status      ReportStatus `json:"status"`
	Content     []byte       `json:"-"` // Never serialized inline; use download endpoint.
	ContentType string       `json:"content_type,omitempty"`
	Error       string       `json:"error,omitempty"`
	RequestedAt time.Time    `json:"requested_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	SizeBytes   int          `json:"size_bytes,omitempty"`
}

// OverviewSummary is the aggregated dashboard payload for GET /api/discovery/overview.
type OverviewSummary struct {
	SourceCount        int          `json:"source_count"`
	AssetCount         int          `json:"asset_count"`
	FindingCount       int          `json:"finding_count"`
	HighRiskCount      int          `json:"high_risk_count"`
	CompositeRiskScore float64      `json:"composite_risk_score"`
	TopHotspots        []RiskHotspot `json:"top_hotspots"`
	TrendQuarters      []TrendPoint  `json:"trend_quarters"`
	LastSnapshotAt     *time.Time    `json:"last_snapshot_at,omitempty"`
}

// RiskHotspot is one entry in the "top N highest-risk assets" list shown on the overview.
type RiskHotspot struct {
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	Score          float64   `json:"score"`
	Classification string    `json:"classification"`
	FindingCount   int       `json:"finding_count"`
}

// TrendPoint is one (label, value) pair on the overview's trend chart.
type TrendPoint struct {
	Label              string  `json:"label"`
	TakenAt            time.Time `json:"taken_at"`
	AssetCount         int     `json:"asset_count"`
	FindingCount       int     `json:"finding_count"`
	CompositeRiskScore float64 `json:"composite_risk_score"`
}

// RiskWeights is the config-driven weighting for the composite risk score formula.
// Score = sum(finding_count * sensitivity_weight * exposure_weight)
type RiskWeights struct {
	Volume      float64 // weight on raw finding count
	Sensitivity float64 // weight on classification sensitivity
	Exposure    float64 // weight on source exposure (e.g. external > internal)
}

// DefaultRiskWeights returns the v1 defaults. Overridable via config.
func DefaultRiskWeights() RiskWeights {
	return RiskWeights{
		Volume:      1.0,
		Sensitivity: 2.0,
		Exposure:    1.5,
	}
}
