package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
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
	repo *persistence.PostgresRepository
}

// NewScanService creates a new scan service
func NewScanService(repo *persistence.PostgresRepository) *ScanService {
	return &ScanService{
		repo: repo,
	}
}

// TriggerScanRequest represents a scan trigger request
type TriggerScanRequest struct {
	Name          string   `json:"name" binding:"required,min=1,max=100"`
	Sources       []string `json:"sources" binding:"required,min=1,dive,required"`
	PIITypes      []string `json:"pii_types" binding:"required,min=1,dive,required"`
	ExecutionMode string   `json:"execution_mode" binding:"required,oneof=sequential parallel"`
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
