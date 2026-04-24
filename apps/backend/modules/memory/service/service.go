package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/arc-platform/backend/modules/shared/interfaces"
)

// MemoryService is the domain layer — builds meaningful memory entries
// from scan/finding events and surfaces them for /api/v1/memory/* routes.
type MemoryService struct {
	client *Client
}

func NewMemoryService(client *Client) *MemoryService {
	return &MemoryService{client: client}
}

// Enabled reports whether the underlying client can call the API.
// Handlers use this to short-circuit to 503 when SUPERMEMORY_ENABLED=false.
func (s *MemoryService) Enabled() bool { return s.client != nil && s.client.Enabled() }

// ScanSummary is a type alias for the shared interface snapshot so a single
// struct crosses the module boundary cleanly. Fields are intentionally narrow:
// the free tier bills per token ingested, so we avoid raw findings.
type ScanSummary = interfaces.ScanSummarySnapshot

// RecordScanCompletion writes a one-line memory entry + tags for a finished scan.
// Safe to call even when disabled — it returns nil in that case so callers
// don't need to guard with Enabled().
func (s *MemoryService) RecordScanCompletion(ctx context.Context, sum ScanSummary) error {
	if !s.Enabled() {
		return nil
	}
	tenantHash := fmt.Sprintf("%x", sha256.Sum256([]byte(sum.TenantID)))[:12]
	content := fmt.Sprintf(
		"Scan %q (id=%s) finished at %s. %d assets, %d findings (%d critical, %d high). Source types: %d. PII types: %s. Duration: %dms.",
		sum.ScanName, sum.ScanID,
		sum.CompletedAt.Format(time.RFC3339),
		sum.TotalAssets, sum.TotalFindings, sum.Critical, sum.High,
		len(sum.SourceTypes), strings.Join(sum.PIITypes, ","),
		sum.DurationMs,
	)
	tags := []string{"arc-hawk", "scan", "t:" + tenantHash}
	_, err := s.client.AddDocument(ctx, Document{
		Content: content,
		Title:   "Scan: " + sum.ScanName,
		Tags:    tags,
		Metadata: map[string]interface{}{
			"kind":           "scan_summary",
			"scan_id":        sum.ScanID,
			"tenant_hash":    tenantHash,
			"total_findings": sum.TotalFindings,
			"critical":       sum.Critical,
			"high":           sum.High,
			"completed_at":   sum.CompletedAt.Format(time.RFC3339),
		},
	})
	return err
}

// FindingContext writes a richer memory for a single high-severity finding.
// Use sparingly (1M tokens/mo cap on free tier).
type FindingContext struct {
	FindingID string
	TenantID  string
	AssetName string
	Location  string // table.column or file path
	PIIType   string
	Severity  string
	Redacted  string // value with PII scrubbed
}

func (s *MemoryService) RecordFinding(ctx context.Context, f FindingContext) error {
	if !s.Enabled() {
		return nil
	}
	tenantHash := fmt.Sprintf("%x", sha256.Sum256([]byte(f.TenantID)))[:12]
	content := fmt.Sprintf(
		"Finding %s: %s PII detected in %s (severity=%s). Redacted preview: %s.",
		f.FindingID, f.PIIType, f.AssetName, f.Severity, f.Redacted,
	)
	_, err := s.client.AddDocument(ctx, Document{
		Content: content,
		Title:   fmt.Sprintf("%s finding in %s", f.PIIType, f.AssetName),
		Tags:    []string{"arc-hawk", "finding", "t:" + tenantHash, f.PIIType, f.Severity},
		Metadata: map[string]interface{}{
			"kind":        "finding",
			"finding_id":  f.FindingID,
			"tenant_hash": tenantHash,
			"pii_type":    f.PIIType,
			"severity":    f.Severity,
		},
	})
	return err
}

// Search is a thin forwarder used by the handler. Callers may add tenant filters.
func (s *MemoryService) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	if !s.Enabled() {
		return &SearchResponse{Results: []SearchResult{}}, nil
	}
	return s.client.Search(ctx, q)
}

// Compile-time assertion that MemoryService satisfies interfaces.MemoryRecorder.
// Works because ScanSummary is an alias for interfaces.ScanSummarySnapshot,
// so RecordScanCompletion's signature matches the interface exactly.
var _ interfaces.MemoryRecorder = (*MemoryService)(nil)
