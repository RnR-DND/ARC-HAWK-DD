package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
)

// Valid scan statuses
const (
	ScanStatusPending   = "pending"
	ScanStatusRunning   = "running"
	ScanStatusCompleted = "completed"
	ScanStatusFailed    = "failed"
	ScanStatusCancelled = "cancelled"
	ScanStatusTimeout   = "timeout"
)

// validTransitions defines allowed state transitions
var validTransitions = map[string][]string{
	ScanStatusPending:   {ScanStatusRunning, ScanStatusCancelled, ScanStatusFailed},
	ScanStatusRunning:   {ScanStatusCompleted, ScanStatusFailed, ScanStatusCancelled, ScanStatusTimeout},
	ScanStatusCompleted: {},
	ScanStatusFailed:    {},
	ScanStatusCancelled: {},
	ScanStatusTimeout:   {},
}

// ValidateStatusTransition checks if a status transition is allowed
func ValidateStatusTransition(from, to string) error {
	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("unknown current status: %s", from)
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("invalid transition from %q to %q", from, to)
}

// ScanService manages scan execution and state
type ScanService struct {
	repo     *persistence.PostgresRepository
	memoryFn func(context.Context, interfaces.ScanSummarySnapshot) // best-effort; may be nil
}

// NewScanService creates a new scan service
func NewScanService(repo *persistence.PostgresRepository) *ScanService {
	return &ScanService{
		repo: repo,
	}
}

// SetMemoryRecorder injects a MemoryRecorder (e.g., supermemory-backed).
// Call from module.Initialize after deps.MemoryRecorder is available.
// Called by scanning/module.go; safe to skip (scans still work, memory no-ops).
func (s *ScanService) SetMemoryRecorder(rec interfaces.MemoryRecorder) {
	if rec == nil || !rec.Enabled() {
		return
	}
	s.memoryFn = func(ctx context.Context, snap interfaces.ScanSummarySnapshot) {
		if err := rec.RecordScanCompletion(ctx, snap); err != nil {
			// Best-effort: log and continue. Never break a scan because memory backend hiccuped.
			log.Printf("memory: record scan %s failed: %v", snap.ScanID, err)
		}
	}
}

// TriggerScanRequest represents a scan trigger request
type TriggerScanRequest struct {
	Name          string   `json:"name" binding:"required,min=1,max=100"`
	Sources       []string `json:"sources" binding:"required,min=1,dive,required"`
	PIITypes      []string `json:"pii_types" binding:"required,min=1,dive,required"`
	ExecutionMode string   `json:"execution_mode" binding:"required,oneof=sequential parallel"`
	// PIITypesPerSource maps profile_name → PII types for that source.
	// When set, overrides the global PIITypes for the specified sources.
	// Sources not listed here fall back to the global PIITypes.
	PIITypesPerSource map[string][]string `json:"pii_types_per_source,omitempty"`
	// ClassificationMode controls which detection engines the scanner uses.
	// Options: "regex" (regex only), "ner" (regex+spaCy NER), "contextual" (regex+NER+contextual).
	// Default (empty) = "contextual" (all engines enabled).
	ClassificationMode string `json:"classification_mode"`
	// CustomPatterns are user-defined patterns appended to the scan; populated by backend at trigger time.
	CustomPatterns []map[string]any `json:"custom_patterns,omitempty"`
}

// CreateScanRun creates a new scan run entity
func (s *ScanService) CreateScanRun(ctx context.Context, req *TriggerScanRequest, triggeredBy string) (*entity.ScanRun, error) {
	scanRun := &entity.ScanRun{
		ID:            uuid.New(),
		ProfileName:   req.Name,
		Status:        "pending",
		ScanStartedAt: time.Now().UTC(),
		Metadata: map[string]interface{}{
			"sources":         req.Sources,
			"pii_types":       req.PIITypes,
			"execution_mode":  req.ExecutionMode,
			"triggered_by":    triggeredBy,
			"trigger_source":  "ui",
			"timeout_minutes": 30, // Default timeout
		},
	}

	if err := s.repo.CreateScanRun(ctx, scanRun); err != nil {
		return nil, fmt.Errorf("failed to create scan run: %w", err)
	}

	return scanRun, nil
}

// UpdateScanStatus updates the status of a scan run with transition validation
func (s *ScanService) UpdateScanStatus(ctx context.Context, scanID uuid.UUID, status string) error {
	scanRun, err := s.repo.GetScanRunByID(ctx, scanID)
	if err != nil {
		return fmt.Errorf("scan run not found: %w", err)
	}

	if err := ValidateStatusTransition(scanRun.Status, status); err != nil {
		log.Printf("WARN: Scan %s status transition rejected: %v", scanID, err)
		return err
	}

	scanRun.Status = status
	if status == "running" && scanRun.ScanStartedAt.IsZero() {
		scanRun.ScanStartedAt = time.Now().UTC()
	}
	if status == "completed" || status == "failed" || status == "cancelled" || status == "timeout" {
		scanRun.ScanCompletedAt = time.Now().UTC()
	}

	if err := s.repo.UpdateScanRun(ctx, scanRun); err != nil {
		return fmt.Errorf("failed to update scan run: %w", err)
	}

	// Best-effort: on "completed" only, push a narrow summary to the memory layer.
	// Runs in a detached goroutine so memory-backend latency never stalls the DB write.
	if status == "completed" && s.memoryFn != nil {
		snap := interfaces.ScanSummarySnapshot{
			ScanID:      scanRun.ID.String(),
			ScanName:    scanRun.ProfileName,
			CompletedAt: scanRun.ScanCompletedAt,
			DurationMs:  scanRun.ScanCompletedAt.Sub(scanRun.ScanStartedAt).Milliseconds(),
		}
		if scanRun.TenantID != uuid.Nil {
			snap.TenantID = scanRun.TenantID.String()
		}
		if md := scanRun.Metadata; md != nil {
			if v, ok := md["sources"].([]string); ok {
				snap.SourceTypes = v
			} else if v, ok := md["sources"].([]interface{}); ok {
				for _, s := range v {
					if str, ok := s.(string); ok {
						snap.SourceTypes = append(snap.SourceTypes, str)
					}
				}
			}
			if v, ok := md["pii_types"].([]string); ok {
				snap.PIITypes = v
			} else if v, ok := md["pii_types"].([]interface{}); ok {
				for _, s := range v {
					if str, ok := s.(string); ok {
						snap.PIITypes = append(snap.PIITypes, str)
					}
				}
			}
		}
		// Detach: ingestion is async on supermemory's side too, and we don't want
		// a slow/dead memory backend to hold up the DB write the caller just did.
		go s.memoryFn(context.Background(), snap)
	}

	return nil
}

// CancelScan cancels a running scan
func (s *ScanService) CancelScan(ctx context.Context, scanID uuid.UUID) error {
	scanRun, err := s.repo.GetScanRunByID(ctx, scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan run: %w", err)
	}

	// Only allow cancellation of pending or running scans
	if scanRun.Status != "pending" && scanRun.Status != "running" {
		return fmt.Errorf("cannot cancel scan with status: %s", scanRun.Status)
	}

	return s.UpdateScanStatus(ctx, scanID, "cancelled")
}

// CheckScanTimeout checks if a scan has exceeded its timeout and marks it as timed out
func (s *ScanService) CheckScanTimeout(ctx context.Context, scanID uuid.UUID) error {
	scanRun, err := s.repo.GetScanRunByID(ctx, scanID)
	if err != nil {
		return fmt.Errorf("failed to get scan run: %w", err)
	}

	if scanRun.Status != "running" {
		return nil // Only check running scans
	}

	timeoutMinutes := 30 // Default
	if timeout, ok := scanRun.Metadata["timeout_minutes"].(float64); ok {
		timeoutMinutes = int(timeout)
	}

	elapsed := time.Since(scanRun.ScanStartedAt)
	if elapsed > time.Duration(timeoutMinutes)*time.Minute {
		return s.UpdateScanStatus(ctx, scanID, "timeout")
	}

	return nil
}

// GetScanRun retrieves a scan run by ID
func (s *ScanService) GetScanRun(ctx context.Context, scanID uuid.UUID) (*entity.ScanRun, error) {
	return s.repo.GetScanRunByID(ctx, scanID)
}

// ListScanRuns retrieves a list of scan runs
func (s *ScanService) ListScanRuns(ctx context.Context, limit, offset int) ([]*entity.ScanRun, error) {
	return s.repo.ListScanRuns(ctx, limit, offset)
}

// ScanDelta holds findings added and removed between two consecutive scans.
type ScanDelta struct {
	ScanID       uuid.UUID      `json:"scan_id"`
	PrevScanID   uuid.UUID      `json:"prev_scan_id"`
	NewFindings  int            `json:"new_findings"`
	GoneFindings int            `json:"gone_findings"`
	NetChange    int            `json:"net_change"` // positive = more PII, negative = less PII
	NewByType    map[string]int `json:"new_by_type"`
	GoneByType   map[string]int `json:"gone_by_type"`
}

// GetScanDelta compares findings in scanID against the previous completed scan
// for the same sources. Returns nil if there is no previous scan to compare.
func (s *ScanService) GetScanDelta(ctx context.Context, scanID uuid.UUID) (*ScanDelta, error) {
	// Find the sources this scan covered
	current, err := s.repo.GetScanRunByID(ctx, scanID)
	if err != nil {
		return nil, fmt.Errorf("scan not found: %w", err)
	}

	// Find the most recent completed scan before this one
	const prevQuery = `
		SELECT id FROM scan_runs
		 WHERE status = 'completed'
		   AND id != $1
		   AND scan_started_at < (SELECT scan_started_at FROM scan_runs WHERE id = $1)
		 ORDER BY scan_started_at DESC
		 LIMIT 1
	`
	var prevID uuid.UUID
	err = s.repo.GetDB().QueryRowContext(ctx, prevQuery, current.ID).Scan(&prevID)
	if err != nil {
		return nil, nil // No previous scan — caller returns 204
	}

	// Count findings per PII type in each scan
	const findingsQuery = `
		SELECT cl.sub_category AS pii_type, COUNT(*) AS cnt
		  FROM findings f
		  JOIN classifications cl ON cl.finding_id = f.id
		 WHERE f.scan_run_id = $1
		 GROUP BY cl.sub_category
	`
	countByType := func(id uuid.UUID) (map[string]int, error) {
		rows, err := s.repo.GetDB().QueryContext(ctx, findingsQuery, id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		m := map[string]int{}
		for rows.Next() {
			var t string
			var c int
			if err := rows.Scan(&t, &c); err != nil {
				continue
			}
			m[t] = c
		}
		return m, rows.Err()
	}

	currMap, err := countByType(scanID)
	if err != nil {
		return nil, fmt.Errorf("current findings: %w", err)
	}
	prevMap, err := countByType(prevID)
	if err != nil {
		return nil, fmt.Errorf("prev findings: %w", err)
	}

	newByType := map[string]int{}
	goneByType := map[string]int{}
	allTypes := map[string]bool{}
	for t := range currMap {
		allTypes[t] = true
	}
	for t := range prevMap {
		allTypes[t] = true
	}

	for t := range allTypes {
		c, p := currMap[t], prevMap[t]
		if c > p {
			newByType[t] = c - p
		}
		if p > c {
			goneByType[t] = p - c
		}
	}

	totalNew := 0
	for _, v := range newByType {
		totalNew += v
	}
	totalGone := 0
	for _, v := range goneByType {
		totalGone += v
	}

	return &ScanDelta{
		ScanID:       scanID,
		PrevScanID:   prevID,
		NewFindings:  totalNew,
		GoneFindings: totalGone,
		NetChange:    totalNew - totalGone,
		NewByType:    newByType,
		GoneByType:   goneByType,
	}, nil
}

// CheckAllScanTimeouts checks all active scans for timeout.
// Called by the background ticker started in the scanning module.
func (s *ScanService) CheckAllScanTimeouts(ctx context.Context) {
	scans, err := s.repo.ListScanRuns(ctx, 100, 0)
	if err != nil {
		log.Printf("ERROR: Failed to list scans for timeout check: %v", err)
		return
	}
	for _, scan := range scans {
		if scan.Status == ScanStatusPending || scan.Status == ScanStatusRunning {
			if err := s.CheckScanTimeout(ctx, scan.ID); err != nil {
				log.Printf("ERROR: Timeout check failed for scan %s: %v", scan.ID, err)
			}
		}
	}
}
