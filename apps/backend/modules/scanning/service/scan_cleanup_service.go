package service

import (
	"context"
	"log"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

// ScanCleanupService handles background cleanup of stale scans
type ScanCleanupService struct {
	repo *persistence.PostgresRepository
}

// NewScanCleanupService creates a new scan cleanup service
func NewScanCleanupService(repo *persistence.PostgresRepository) *ScanCleanupService {
	return &ScanCleanupService{
		repo: repo,
	}
}

// StartCleanupWorker starts a background worker to clean up stale scans
func (s *ScanCleanupService) StartCleanupWorker(ctx context.Context, intervalMinutes int) {
	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	log.Printf("🧹 Starting scan cleanup worker (interval: %d minutes)", intervalMinutes)

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Scan cleanup worker stopped")
			return
		case <-ticker.C:
			s.cleanupStaleScans(ctx)
		}
	}
}

// cleanupStaleScans finds and marks stale scans as timed out
func (s *ScanCleanupService) cleanupStaleScans(ctx context.Context) {
	log.Println("🔍 Checking for stale scans...")

	// Find scans that have been running for more than 30 minutes
	query := `
		UPDATE scans 
		SET status = 'timeout', 
		    scan_completed_at = NOW(),
		    updated_at = NOW()
		WHERE status = 'running' 
		  AND scan_started_at < NOW() - INTERVAL '30 minutes'
		  AND scan_completed_at IS NULL
		RETURNING id, profile_name, scan_started_at
	`

	rows, err := s.repo.GetDB().QueryContext(ctx, query)
	if err != nil {
		log.Printf("❌ Error cleaning up stale scans: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var scanID, profileName string
		var startedAt time.Time
		if err := rows.Scan(&scanID, &profileName, &startedAt); err != nil {
			log.Printf("❌ Error scanning stale scan row: %v", err)
			continue
		}

		duration := time.Since(startedAt)
		log.Printf("⏱️  Marked scan %s (%s) as timed out after %v", scanID, profileName, duration)
		count++
	}

	if count > 0 {
		log.Printf("✅ Cleaned up %d stale scan(s)", count)
	}
}

// CleanupStaleScanOnce runs cleanup once (useful for manual triggers or testing)
func (s *ScanCleanupService) CleanupStaleScanOnce(ctx context.Context) int {
	s.cleanupStaleScans(ctx)

	// Return count of cleaned scans
	var count int
	query := `
		SELECT COUNT(*) FROM scans 
		WHERE status = 'timeout' 
		  AND updated_at > NOW() - INTERVAL '1 minute'
	`
	s.repo.GetDB().QueryRowContext(ctx, query).Scan(&count)
	return count
}
