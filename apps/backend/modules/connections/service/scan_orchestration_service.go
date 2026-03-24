package service

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// ScanOrchestrationService manages scan jobs across all assets
type ScanOrchestrationService struct {
	pgRepo *persistence.PostgresRepository
	jobs   map[string]*ScanJob
	mu     sync.RWMutex
}

// ScanJob represents a scan job for an asset
type ScanJob struct {
	ID          string    `json:"id"`
	AssetID     uuid.UUID `json:"asset_id"`
	AssetName   string    `json:"asset_name"`
	AssetPath   string    `json:"asset_path"`
	Status      string    `json:"status"` // queued, running, completed, failed
	Progress    int       `json:"progress"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// ScanAllStatus represents the overall scan status
type ScanAllStatus struct {
	TotalJobs       int       `json:"total_jobs"`
	QueuedJobs      int       `json:"queued_jobs"`
	RunningJobs     int       `json:"running_jobs"`
	CompletedJobs   int       `json:"completed_jobs"`
	FailedJobs      int       `json:"failed_jobs"`
	OverallStatus   string    `json:"overall_status"` // idle, scanning, completed
	StartedAt       time.Time `json:"started_at,omitempty"`
	CompletedAt     time.Time `json:"completed_at,omitempty"`
	ProgressPercent int       `json:"progress_percent"`
}

// NewScanOrchestrationService creates a new scan orchestration service
func NewScanOrchestrationService(pgRepo *persistence.PostgresRepository) *ScanOrchestrationService {
	return &ScanOrchestrationService{
		pgRepo: pgRepo,
		jobs:   make(map[string]*ScanJob),
	}
}

// ScanAllAssets triggers scans for all assets
func (s *ScanOrchestrationService) ScanAllAssets(ctx context.Context) (*ScanAllStatus, error) {
	s.mu.Lock()

	fmt.Printf("🚀 Starting Scan All Assets...\n")

	// NOTE: Connection sync removed - connections are now managed via database
	// in the new architecture. Assets should already exist in DB from connection creation.
	// This legacy "Scan All" flow will be replaced by Temporal workflows in Phase 3.

	// Get all assets from database
	assets, err := s.pgRepo.ListAssets(ctx, 10000, 0)
	if err != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}

	fmt.Printf("📊 Found %d assets to scan\n", len(assets))

	// Clear old jobs
	s.jobs = make(map[string]*ScanJob)

	// Create scan jobs for each asset
	// Note: In Global Scan mode, we update all these jobs simultaneously
	for _, asset := range assets {
		jobID := uuid.New().String()
		job := &ScanJob{
			ID:        jobID,
			AssetID:   asset.ID,
			AssetName: asset.Name,
			AssetPath: asset.Path,
			Status:    "queued",
			Progress:  0,
			StartedAt: time.Now(),
		}
		s.jobs[jobID] = job
	}

	// If no assets in DB, create a dummy job for "Discovery"
	if len(assets) == 0 {
		jobID := uuid.New().String()
		s.jobs[jobID] = &ScanJob{
			ID:        jobID,
			AssetName: "Global Discovery Scan",
			Status:    "queued",
			Progress:  0,
			StartedAt: time.Now(),
		}
	}

	fmt.Printf("🚀 Created %d scan jobs (Global Scan Mode)\n", len(s.jobs))

	// Build status manually to avoid lock contention
	status := &ScanAllStatus{
		TotalJobs:       len(s.jobs),
		QueuedJobs:      len(s.jobs),
		RunningJobs:     0,
		CompletedJobs:   0,
		FailedJobs:      0,
		OverallStatus:   "scanning",
		StartedAt:       time.Now(),
		ProgressPercent: 0,
	}

	// CRITICAL: Release lock BEFORE starting goroutine
	s.mu.Unlock()

	// Start background processing (now lock is released)
	go s.processJobs()

	return status, nil
}

// processJobs executes the REAL Python Scanner
func (s *ScanOrchestrationService) processJobs() {
	s.mu.Lock()
	// Set all jobs to running
	for _, job := range s.jobs {
		job.Status = "running"
		job.Progress = 10
	}
	s.mu.Unlock()

	fmt.Println("🦅 Launching Hawk Scanner SDK...")

	// Construct command to run scanner
	// We run python3 from the scanner directory to ensure imports work
	// NOTE: Removed --json to allow auto-ingest to run (--json causes early exit)
	cmd := exec.Command("python3", "hawk_scanner/main.py", "all",
		"--connection", "config/connection.yml",
		"--fingerprint", "../../fingerprint.yml",
		"--ingest-url", "http://localhost:8080/api/v1/scans/ingest-verified",
		"--quiet") // Use --quiet to suppress table output

	// Set working directory to apps/scanner (relative to apps/backend)
	cmd.Dir = "../scanner"

	// Start scanner asynchronously (non-blocking)
	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ Failed to start scanner: %v\n", err)
		s.mu.Lock()
		for _, job := range s.jobs {
			job.Status = "failed"
			job.Error = "Failed to start scanner process"
			job.CompletedAt = time.Now()
		}
		s.mu.Unlock()
		return
	}

	fmt.Printf("✅ Scanner process started (PID: %d)\n", cmd.Process.Pid)

	// Wait for scanner to complete in background
	err := cmd.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		fmt.Printf("❌ Scanner execution failed: %v\n", err)
		// Mark all as failed
		for _, job := range s.jobs {
			job.Status = "failed"
			job.Error = "Scanner execution failed. Check backend logs."
			job.CompletedAt = time.Now()
		}
		return
	}

	fmt.Println("✅ Scanner completed successfully!")

	// Mark all as completed
	for _, job := range s.jobs {
		job.Status = "completed"
		job.Progress = 100
		job.CompletedAt = time.Now()
	}
}

// GetScanStatus returns the current scan status
func (s *ScanOrchestrationService) GetScanStatus(ctx context.Context) *ScanAllStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &ScanAllStatus{
		TotalJobs:     len(s.jobs),
		QueuedJobs:    0,
		RunningJobs:   0,
		CompletedJobs: 0,
		FailedJobs:    0,
	}

	var earliestStart time.Time
	var latestCompletion time.Time

	for _, job := range s.jobs {
		switch job.Status {
		case "queued":
			status.QueuedJobs++
		case "running":
			status.RunningJobs++
		case "completed":
			status.CompletedJobs++
		case "failed":
			status.FailedJobs++
		}

		if !job.StartedAt.IsZero() && (earliestStart.IsZero() || job.StartedAt.Before(earliestStart)) {
			earliestStart = job.StartedAt
		}
		if !job.CompletedAt.IsZero() && job.CompletedAt.After(latestCompletion) {
			latestCompletion = job.CompletedAt
		}
	}

	// Calculate overall status
	if status.TotalJobs == 0 {
		status.OverallStatus = "idle"
	} else if status.CompletedJobs+status.FailedJobs == status.TotalJobs {
		status.OverallStatus = "completed"
		status.CompletedAt = latestCompletion
	} else {
		status.OverallStatus = "scanning"
	}

	if !earliestStart.IsZero() {
		status.StartedAt = earliestStart
	}

	// Calculate progress percentage
	if status.TotalJobs > 0 {
		status.ProgressPercent = (status.CompletedJobs * 100) / status.TotalJobs
	}

	// If we are running (Python scan active) but haven't finished, fake some progress
	if status.RunningJobs > 0 && status.ProgressPercent < 90 {
		// Just a visual indicator that it's not stuck
		status.ProgressPercent = 50
	}

	return status
}

// GetAllJobs returns all scan jobs
func (s *ScanOrchestrationService) GetAllJobs(ctx context.Context) []*ScanJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*ScanJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}
