package service

import (
	"context"
	"fmt"
	"log"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

// InventoryService refreshes the discovery_inventory table from upstream module data.
//
// It is a synthesis layer: it reads asset+finding+classification state from the
// existing assets/scanning modules and writes denormalized rows into discovery_inventory.
// Triggered by the snapshot worker (cron) and on-demand from manual snapshot requests.
type InventoryService struct {
	repo     *Repo
	upstream Upstream
}

// NewInventoryService creates a new inventory service.
func NewInventoryService(repo *Repo, upstream Upstream) *InventoryService {
	return &InventoryService{repo: repo, upstream: upstream}
}

// RefreshInventory rebuilds the discovery_inventory rows for the tenant in ctx.
//
// Strategy: page through ListAssetSummaries (limit 1000), upsert each row.
// This is idempotent — running twice produces the same end state.
//
// Returns the number of rows refreshed and any error encountered. On error, partial
// progress is preserved (callers can retry).
func (s *InventoryService) RefreshInventory(ctx context.Context) (int, error) {
	// Defensive: validate tenant context up front so we fail fast with a clear error.
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return 0, fmt.Errorf("refresh inventory: %w", err)
	}

	const pageSize = 1000
	offset := 0
	total := 0

	for {
		summaries, err := s.upstream.ListAssetSummaries(ctx, pageSize, offset)
		if err != nil {
			return total, fmt.Errorf("refresh inventory: upstream read failed: %w", ErrUpstreamUnavailable)
		}
		if len(summaries) == 0 {
			break
		}

		for _, sum := range summaries {
			// Skip rows with no classification — they have no PII signal yet.
			if sum.Classification == "" {
				continue
			}

			row := &domain.InventoryRow{
				TenantID:       tenantID,
				AssetID:        sum.AssetID,
				AssetName:      sum.AssetName,
				SourceID:       sum.SourceID,
				SourceName:     sum.SourceName,
				Classification: sum.Classification,
				Sensitivity:    sum.Sensitivity,
				FindingCount:   sum.FindingCount,
				LastScannedAt:  sum.LastScannedAt,
			}
			if err := s.repo.UpsertInventoryRow(ctx, row); err != nil {
				log.Printf("⚠️  discovery inventory upsert failed for asset %s: %v", sum.AssetID, err)
				// Continue — one bad asset shouldn't fail the whole refresh.
				continue
			}
			total++
		}

		if len(summaries) < pageSize {
			break
		}
		offset += pageSize
	}

	return total, nil
}

// GetOverviewSummary returns the aggregated dashboard payload for the tenant in ctx.
// Reads from the most recent completed snapshot if available, falls back to live counts.
func (s *InventoryService) GetOverviewSummary(ctx context.Context, repo *Repo) (*domain.OverviewSummary, error) {
	if _, err := persistence.EnsureTenantID(ctx); err != nil {
		return nil, fmt.Errorf("overview summary: %w", err)
	}

	last, err := repo.GetLastCompletedSnapshot(ctx, nil)
	if err != nil {
		return nil, err
	}

	out := &domain.OverviewSummary{}
	if last != nil {
		out.SourceCount = last.SourceCount
		out.AssetCount = last.AssetCount
		out.FindingCount = last.FindingCount
		out.HighRiskCount = last.HighRiskCount
		out.CompositeRiskScore = last.CompositeRiskScore
		out.LastSnapshotAt = &last.TakenAt
	} else {
		// No snapshot yet — give live counts so the UI doesn't look empty on first load.
		count, _ := repo.CountInventoryByTenant(ctx)
		out.AssetCount = count
		srcCount, _ := s.upstream.CountSourcesForTenant(ctx)
		out.SourceCount = srcCount
	}

	hotspots, err := repo.ListTopRiskHotspots(ctx, 5)
	if err == nil {
		for _, h := range hotspots {
			out.TopHotspots = append(out.TopHotspots, *h)
		}
	}

	// 4-quarter trend: last 4 completed snapshots.
	snaps, err := repo.ListSnapshots(ctx, 4, 0)
	if err == nil {
		for i := len(snaps) - 1; i >= 0; i-- {
			s := snaps[i]
			if s.Status != domain.SnapshotCompleted {
				continue
			}
			out.TrendQuarters = append(out.TrendQuarters, domain.TrendPoint{
				Label:              s.TakenAt.Format("2006-01-02"),
				TakenAt:            s.TakenAt,
				AssetCount:         s.AssetCount,
				FindingCount:       s.FindingCount,
				CompositeRiskScore: s.CompositeRiskScore,
			})
		}
	}

	return out, nil
}
