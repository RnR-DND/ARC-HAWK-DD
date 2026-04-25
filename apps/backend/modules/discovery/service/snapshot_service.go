package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
)

// SnapshotService captures point-in-time inventory state into discovery_snapshots
// and discovery_snapshot_facts. Snapshots are atomic: header + facts are written
// in a single transaction (per E5).
type SnapshotService struct {
	repo        *Repo
	inventory   *InventoryService
	auditLogger interfaces.AuditLogger
}

// NewSnapshotService creates a new snapshot service.
func NewSnapshotService(repo *Repo, inventory *InventoryService, auditLogger interfaces.AuditLogger) *SnapshotService {
	return &SnapshotService{
		repo:        repo,
		inventory:   inventory,
		auditLogger: auditLogger,
	}
}

// TakeSnapshot creates a new snapshot for the tenant in ctx. Refreshes inventory
// first, then aggregates current state into a snapshot header + per-source facts.
//
// trigger: domain.TriggerManual or domain.TriggerCron
// triggeredBy: user UUID for manual triggers; nil for cron
//
// Returns the new snapshot ID. On any error inside the transaction, the snapshot is
// marked failed (via a separate connection) and the error is returned.
func (s *SnapshotService) TakeSnapshot(ctx context.Context, trigger domain.SnapshotTrigger, triggeredBy *uuid.UUID) (snap *domain.Snapshot, retErr error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("take snapshot: %w", err)
	}

	startAt := time.Now()

	// 1. Create header in 'pending' status.
	snap = &domain.Snapshot{
		TenantID:    tenantID,
		Trigger:     trigger,
		TriggeredBy: triggeredBy,
	}
	if err := s.repo.CreateSnapshot(ctx, snap); err != nil {
		return nil, fmt.Errorf("create snapshot header: %w", err)
	}

	// 2. Refresh inventory from upstream — this is the slow part. Do it OUTSIDE the
	// transaction so we don't hold a long-running tx open.
	if _, err := s.inventory.RefreshInventory(ctx); err != nil {
		_ = s.repo.FailSnapshot(ctx, snap.ID, fmt.Sprintf("inventory refresh failed: %v", err))
		return snap, err
	}

	// 3. Inside a tx: aggregate inventory into facts + header counts.
	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		_ = s.repo.FailSnapshot(ctx, snap.ID, fmt.Sprintf("begin tx: %v", err))
		return snap, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			retErr = fmt.Errorf("snapshot tx panic: %v", p)
		}
	}()

	facts, header, err := s.aggregateInventoryTx(ctx, tx, snap.ID, tenantID)
	if err != nil {
		_ = tx.Rollback()
		_ = s.repo.FailSnapshot(ctx, snap.ID, fmt.Sprintf("aggregate inventory: %v", err))
		return snap, err
	}

	// Apply header counts to the snapshot row.
	snap.SourceCount = header.SourceCount
	snap.AssetCount = header.AssetCount
	snap.FindingCount = header.FindingCount
	snap.HighRiskCount = header.HighRiskCount
	snap.CompositeRiskScore = header.CompositeRiskScore
	snap.DurationMS = time.Since(startAt).Milliseconds()

	if err := s.repo.InsertFactsTx(ctx, tx, facts); err != nil {
		_ = tx.Rollback()
		_ = s.repo.FailSnapshot(ctx, snap.ID, fmt.Sprintf("insert facts: %v", err))
		return snap, err
	}

	if err := s.repo.CompleteSnapshotTx(ctx, tx, snap); err != nil {
		_ = tx.Rollback()
		_ = s.repo.FailSnapshot(ctx, snap.ID, fmt.Sprintf("complete snapshot: %v", err))
		return snap, err
	}

	if err := tx.Commit(); err != nil {
		_ = s.repo.FailSnapshot(ctx, snap.ID, fmt.Sprintf("commit: %v", err))
		return snap, err
	}

	// 4. Audit log the manual trigger (per E9). Cron snapshots don't audit.
	if trigger == domain.TriggerManual && s.auditLogger != nil {
		md := map[string]interface{}{
			"snapshot_id": snap.ID.String(),
			"asset_count": snap.AssetCount,
			"trigger":     string(trigger),
		}
		_ = s.auditLogger.Record(ctx, "trigger_snapshot", "discovery_snapshot", snap.ID.String(), md)
	}

	log.Printf("📸 Discovery snapshot %s completed for tenant %s (assets=%d, findings=%d, dur=%dms)",
		snap.ID.String()[:8], tenantID, snap.AssetCount, snap.FindingCount, snap.DurationMS)

	return snap, nil
}

// aggregateInventoryTx reads the discovery_inventory rows for the tenant and produces
// per-source-classification facts plus header counts. All inside the tx.
func (s *SnapshotService) aggregateInventoryTx(ctx context.Context, tx *sql.Tx, snapshotID, tenantID uuid.UUID) ([]*domain.SnapshotFact, *domain.Snapshot, error) {
	header := &domain.Snapshot{
		ID:       snapshotID,
		TenantID: tenantID,
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT source_id, COALESCE(source_name, ''), classification,
		       COUNT(DISTINCT asset_id) AS asset_count,
		       SUM(finding_count) AS finding_count,
		       AVG(sensitivity) AS sensitivity_avg
		FROM discovery_inventory
		WHERE tenant_id = $1
		GROUP BY source_id, source_name, classification
	`, tenantID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var facts []*domain.SnapshotFact
	uniqueSources := map[string]bool{}
	totalFindings := 0
	totalAssets := 0
	highRisk := 0
	weightedSum := 0.0

	for rows.Next() {
		fact := &domain.SnapshotFact{
			SnapshotID: snapshotID,
			TenantID:   tenantID,
		}
		var srcID sql.NullString
		var sensAvg sql.NullFloat64
		if err := rows.Scan(
			&srcID, &fact.SourceName, &fact.Classification,
			&fact.AssetCount, &fact.FindingCount, &sensAvg,
		); err != nil {
			return nil, nil, err
		}
		if srcID.Valid {
			id, _ := uuid.Parse(srcID.String)
			fact.SourceID = &id
			uniqueSources[srcID.String] = true
		}
		if sensAvg.Valid {
			fact.SensitivityAvg = sensAvg.Float64
		}
		facts = append(facts, fact)

		totalFindings += fact.FindingCount
		totalAssets += fact.AssetCount
		if fact.SensitivityAvg >= 70 {
			highRisk += fact.AssetCount
		}
		weightedSum += float64(fact.FindingCount) * fact.SensitivityAvg
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	header.SourceCount = len(uniqueSources)
	header.AssetCount = totalAssets
	header.FindingCount = totalFindings
	header.HighRiskCount = highRisk
	if totalFindings > 0 {
		header.CompositeRiskScore = weightedSum / float64(totalFindings)
	}

	return facts, header, nil
}
