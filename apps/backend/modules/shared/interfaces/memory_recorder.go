package interfaces

import (
	"context"
	"time"
)

// ScanSummarySnapshot is a module-independent snapshot of a completed scan
// that MemoryRecorder can ingest. Kept narrow to stay under free-tier
// ingestion limits (1M tokens/mo on supermemory.ai).
type ScanSummarySnapshot struct {
	ScanID        string
	TenantID      string
	ScanName      string
	SourceTypes   []string
	PIITypes      []string
	TotalAssets   int
	TotalFindings int
	Critical      int
	High          int
	DurationMs    int64
	CompletedAt   time.Time
}

// MemoryRecorder is the contract the scanning module uses to push
// scan-completion summaries into the memory layer (currently supermemory.ai).
// Implementations MUST be non-blocking in spirit: a nil return on error
// from the memory backend should not break the scan flow. A nil
// MemoryRecorder is also fine — callers guard with if m != nil.
type MemoryRecorder interface {
	RecordScanCompletion(ctx context.Context, s ScanSummarySnapshot) error
	Enabled() bool
}

// NoOpMemoryRecorder is used when the memory module is disabled.
// Safe zero value (Enabled returns false, Record returns nil).
type NoOpMemoryRecorder struct{}

func (NoOpMemoryRecorder) RecordScanCompletion(context.Context, ScanSummarySnapshot) error {
	return nil
}
func (NoOpMemoryRecorder) Enabled() bool { return false }
