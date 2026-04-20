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
	SourceCount        int           `json:"source_count"`
	AssetCount         int           `json:"asset_count"`
	FindingCount       int           `json:"finding_count"`
	HighRiskCount      int           `json:"high_risk_count"`
	CompositeRiskScore float64       `json:"composite_risk_score"`
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
	Label              string    `json:"label"`
	TakenAt            time.Time `json:"taken_at"`
	AssetCount         int       `json:"asset_count"`
	FindingCount       int       `json:"finding_count"`
	CompositeRiskScore float64   `json:"composite_risk_score"`
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

// DPDPARiskInputs holds the four factors used in the DPDPA 2023 risk formula.
//
//	risk_score = (pii_density × 0.35) + (sensitivity_weight × 0.30) +
//	             (access_exposure × 0.20) + (retention_violation × 0.15)
//
// All inputs are normalised to [0, 1] before the formula is applied.
// The result is scaled to [0, 100] and mapped to a RiskTier.
type DPDPARiskInputs struct {
	// PIIDensity is the fraction of fields in the asset that contain PII (0.0–1.0).
	// Computed as: pii_field_count / total_field_count.
	PIIDensity float64 `json:"pii_density"`

	// SensitivityWeight is the weighted average sensitivity of PII classifications.
	// high=1.0, medium=0.6, low=0.2.
	SensitivityWeight float64 `json:"sensitivity_weight"`

	// AccessExposure is the fraction of PII fields accessible without RBAC (0.0–1.0).
	// Defaults to 0.5 until connections module exposes per-source RBAC metadata.
	AccessExposure float64 `json:"access_exposure"`

	// RetentionViolation is 1 if any finding in the asset exceeds its retention period, else 0.
	RetentionViolation float64 `json:"retention_violation"`
}

// RiskTier is the DPDPA risk tier derived from a 0–100 score.
type RiskTier string

const (
	RiskTierCritical RiskTier = "Critical" // 80–100
	RiskTierHigh     RiskTier = "High"     // 60–79
	RiskTierMedium   RiskTier = "Medium"   // 40–59
	RiskTierLow      RiskTier = "Low"      // 0–39
)

// TierFromScore maps a 0–100 DPDPA score to a RiskTier.
func TierFromScore(score float64) RiskTier {
	switch {
	case score >= 80:
		return RiskTierCritical
	case score >= 60:
		return RiskTierHigh
	case score >= 40:
		return RiskTierMedium
	default:
		return RiskTierLow
	}
}
