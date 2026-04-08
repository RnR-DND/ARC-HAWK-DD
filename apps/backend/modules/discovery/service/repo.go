// Package service contains the discovery module's services.
//
// repo.go is the data-access layer. Every method that touches tenant-scoped data
// MUST extract the tenant ID from context via persistence.EnsureTenantID(ctx) at
// the top — never trust callers to pass tenant ID as a separate argument.
//
// Pattern reference: apps/backend/modules/shared/infrastructure/persistence/asset_repository.go
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// Repo is the discovery module's data access layer over the discovery_* tables.
type Repo struct {
	db *sql.DB
}

// NewRepo creates a new discovery repo backed by the provided *sql.DB.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// DB returns the underlying *sql.DB so services can open transactions.
func (r *Repo) DB() *sql.DB {
	return r.db
}

// ─── discovery_inventory ────────────────────────────────────────────────────

// UpsertInventoryRow inserts or updates a single inventory row for the tenant in ctx.
func (r *Repo) UpsertInventoryRow(ctx context.Context, row *domain.InventoryRow) error {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return fmt.Errorf("upsert inventory row: %w", err)
	}
	row.TenantID = tenantID

	query := `
		INSERT INTO discovery_inventory (
			tenant_id, asset_id, asset_name, source_id, source_name,
			classification, sensitivity, finding_count, last_scanned_at, refreshed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (tenant_id, asset_id, classification) DO UPDATE SET
			asset_name      = EXCLUDED.asset_name,
			source_id       = EXCLUDED.source_id,
			source_name     = EXCLUDED.source_name,
			sensitivity     = EXCLUDED.sensitivity,
			finding_count   = EXCLUDED.finding_count,
			last_scanned_at = EXCLUDED.last_scanned_at,
			refreshed_at    = NOW()
		RETURNING id, refreshed_at
	`
	return r.db.QueryRowContext(ctx, query,
		tenantID, row.AssetID, row.AssetName, row.SourceID, row.SourceName,
		row.Classification, row.Sensitivity, row.FindingCount, row.LastScannedAt,
	).Scan(&row.ID, &row.RefreshedAt)
}

// ListInventory returns a paginated slice of inventory rows for the tenant in ctx.
// Optional filters: classification (empty = no filter), sourceID (nil = no filter).
func (r *Repo) ListInventory(ctx context.Context, classification string, sourceID *uuid.UUID, limit, offset int) ([]*domain.InventoryRow, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("list inventory: %w", err)
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	query := `
		SELECT id, tenant_id, asset_id, asset_name, source_id, source_name,
		       classification, sensitivity, finding_count, last_scanned_at, refreshed_at
		FROM discovery_inventory
		WHERE tenant_id = $1
		  AND ($2 = '' OR classification = $2)
		  AND ($3::uuid IS NULL OR source_id = $3)
		ORDER BY refreshed_at DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID, classification, sourceID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.InventoryRow
	for rows.Next() {
		row := &domain.InventoryRow{}
		if err := rows.Scan(
			&row.ID, &row.TenantID, &row.AssetID, &row.AssetName, &row.SourceID, &row.SourceName,
			&row.Classification, &row.Sensitivity, &row.FindingCount, &row.LastScannedAt, &row.RefreshedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CountInventoryByTenant returns the row count for the tenant in ctx.
func (r *Repo) CountInventoryByTenant(ctx context.Context) (int, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return 0, fmt.Errorf("count inventory: %w", err)
	}
	var n int
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM discovery_inventory WHERE tenant_id = $1`, tenantID,
	).Scan(&n)
	return n, err
}

// ─── discovery_snapshots ────────────────────────────────────────────────────

// CreateSnapshot inserts a new snapshot header in 'pending' status and returns the new ID.
func (r *Repo) CreateSnapshot(ctx context.Context, snap *domain.Snapshot) error {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	snap.TenantID = tenantID
	snap.Status = domain.SnapshotPending

	query := `
		INSERT INTO discovery_snapshots (
			tenant_id, taken_at, source_count, asset_count, finding_count,
			high_risk_count, composite_risk_score, trigger, triggered_by, status
		) VALUES ($1, NOW(), 0, 0, 0, 0, 0, $2, $3, 'pending')
		RETURNING id, taken_at
	`
	return r.db.QueryRowContext(ctx, query, tenantID, snap.Trigger, snap.TriggeredBy).
		Scan(&snap.ID, &snap.TakenAt)
}

// CompleteSnapshot marks the snapshot completed with the final aggregate counts.
// Called inside the same transaction as facts insertion.
func (r *Repo) CompleteSnapshotTx(ctx context.Context, tx *sql.Tx, snap *domain.Snapshot) error {
	completed := time.Now()
	snap.CompletedAt = &completed
	snap.Status = domain.SnapshotCompleted

	query := `
		UPDATE discovery_snapshots SET
			source_count         = $1,
			asset_count          = $2,
			finding_count        = $3,
			high_risk_count      = $4,
			composite_risk_score = $5,
			status               = 'completed',
			duration_ms          = $6,
			completed_at         = $7
		WHERE id = $8
	`
	_, err := tx.ExecContext(ctx, query,
		snap.SourceCount, snap.AssetCount, snap.FindingCount,
		snap.HighRiskCount, snap.CompositeRiskScore,
		snap.DurationMS, completed, snap.ID,
	)
	return err
}

// FailSnapshot marks the snapshot as failed with an error message. Uses its own tx.
func (r *Repo) FailSnapshot(ctx context.Context, snapshotID uuid.UUID, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE discovery_snapshots
		SET status = 'failed', error = $1, completed_at = NOW()
		WHERE id = $2
	`, errMsg, snapshotID)
	return err
}

// GetSnapshot returns a snapshot header by ID, scoped to the tenant in ctx.
func (r *Repo) GetSnapshot(ctx context.Context, snapshotID uuid.UUID) (*domain.Snapshot, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	snap := &domain.Snapshot{}
	var errStr sql.NullString
	var triggeredBy sql.NullString
	var completedAt sql.NullTime
	var duration sql.NullInt64

	err = r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, taken_at, source_count, asset_count, finding_count,
		       high_risk_count, composite_risk_score, trigger, triggered_by, status,
		       error, duration_ms, completed_at
		FROM discovery_snapshots
		WHERE id = $1 AND tenant_id = $2
	`, snapshotID, tenantID).Scan(
		&snap.ID, &snap.TenantID, &snap.TakenAt,
		&snap.SourceCount, &snap.AssetCount, &snap.FindingCount,
		&snap.HighRiskCount, &snap.CompositeRiskScore,
		&snap.Trigger, &triggeredBy, &snap.Status,
		&errStr, &duration, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrSnapshotNotFound
	}
	if err != nil {
		return nil, err
	}
	if errStr.Valid {
		snap.Error = errStr.String
	}
	if triggeredBy.Valid {
		id, _ := uuid.Parse(triggeredBy.String)
		snap.TriggeredBy = &id
	}
	if completedAt.Valid {
		snap.CompletedAt = &completedAt.Time
	}
	if duration.Valid {
		snap.DurationMS = duration.Int64
	}
	return snap, nil
}

// ListSnapshots returns the most recent snapshots for the tenant in ctx, paginated.
func (r *Repo) ListSnapshots(ctx context.Context, limit, offset int) ([]*domain.Snapshot, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, taken_at, source_count, asset_count, finding_count,
		       high_risk_count, composite_risk_score, trigger, status, completed_at
		FROM discovery_snapshots
		WHERE tenant_id = $1
		ORDER BY taken_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Snapshot
	for rows.Next() {
		s := &domain.Snapshot{}
		var completedAt sql.NullTime
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.TakenAt, &s.SourceCount, &s.AssetCount, &s.FindingCount,
			&s.HighRiskCount, &s.CompositeRiskScore, &s.Trigger, &s.Status, &completedAt,
		); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			s.CompletedAt = &completedAt.Time
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetLastCompletedSnapshot returns the most recent completed snapshot for the tenant
// in ctx, or nil if none exists. Used by drift detection to find the prior snapshot.
func (r *Repo) GetLastCompletedSnapshot(ctx context.Context, beforeID *uuid.UUID) (*domain.Snapshot, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get last completed snapshot: %w", err)
	}

	query := `
		SELECT id, tenant_id, taken_at, source_count, asset_count, finding_count,
		       high_risk_count, composite_risk_score, trigger, status, completed_at
		FROM discovery_snapshots
		WHERE tenant_id = $1 AND status = 'completed'
		  AND ($2::uuid IS NULL OR id != $2)
		ORDER BY taken_at DESC
		LIMIT 1
	`
	s := &domain.Snapshot{}
	var completedAt sql.NullTime
	err = r.db.QueryRowContext(ctx, query, tenantID, beforeID).Scan(
		&s.ID, &s.TenantID, &s.TakenAt, &s.SourceCount, &s.AssetCount, &s.FindingCount,
		&s.HighRiskCount, &s.CompositeRiskScore, &s.Trigger, &s.Status, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		s.CompletedAt = &completedAt.Time
	}
	return s, nil
}

// ─── discovery_snapshot_facts ───────────────────────────────────────────────

// InsertFactsTx bulk-inserts facts within an existing transaction.
func (r *Repo) InsertFactsTx(ctx context.Context, tx *sql.Tx, facts []*domain.SnapshotFact) error {
	if len(facts) == 0 {
		return nil
	}
	for _, f := range facts {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO discovery_snapshot_facts (
				snapshot_id, tenant_id, source_id, source_name,
				classification, asset_count, finding_count, sensitivity_avg
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, f.SnapshotID, f.TenantID, f.SourceID, f.SourceName,
			f.Classification, f.AssetCount, f.FindingCount, f.SensitivityAvg,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListFactsForSnapshot returns all facts for a snapshot, scoped to the tenant in ctx.
func (r *Repo) ListFactsForSnapshot(ctx context.Context, snapshotID uuid.UUID) ([]*domain.SnapshotFact, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("list facts: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, snapshot_id, tenant_id, source_id, source_name,
		       classification, asset_count, finding_count, sensitivity_avg
		FROM discovery_snapshot_facts
		WHERE snapshot_id = $1 AND tenant_id = $2
		ORDER BY classification, source_name
	`, snapshotID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.SnapshotFact
	for rows.Next() {
		f := &domain.SnapshotFact{}
		if err := rows.Scan(
			&f.ID, &f.SnapshotID, &f.TenantID, &f.SourceID, &f.SourceName,
			&f.Classification, &f.AssetCount, &f.FindingCount, &f.SensitivityAvg,
		); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ─── discovery_risk_scores ──────────────────────────────────────────────────

// InsertRiskScore writes a new risk score entry for the tenant in ctx.
func (r *Repo) InsertRiskScore(ctx context.Context, score *domain.RiskScore) error {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return fmt.Errorf("insert risk score: %w", err)
	}
	score.TenantID = tenantID

	factors, _ := json.Marshal(score.ContributingFactors)
	return r.db.QueryRowContext(ctx, `
		INSERT INTO discovery_risk_scores (
			tenant_id, asset_id, snapshot_id, score, contributing_factors, computed_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, computed_at
	`, tenantID, score.AssetID, score.SnapshotID, score.Score, factors,
	).Scan(&score.ID, &score.ComputedAt)
}

// ListTopRiskHotspots returns the top N highest-scoring assets for the tenant in ctx.
func (r *Repo) ListTopRiskHotspots(ctx context.Context, limit int) ([]*domain.RiskHotspot, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("list top risk hotspots: %w", err)
	}
	if limit <= 0 || limit > 100 {
		limit = 5
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ON (asset_id)
			rs.asset_id, COALESCE(inv.asset_name, ''), rs.score,
			COALESCE(inv.classification, ''), COALESCE(inv.finding_count, 0)
		FROM discovery_risk_scores rs
		LEFT JOIN discovery_inventory inv ON inv.asset_id = rs.asset_id AND inv.tenant_id = rs.tenant_id
		WHERE rs.tenant_id = $1
		ORDER BY asset_id, rs.computed_at DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.RiskHotspot
	for rows.Next() {
		h := &domain.RiskHotspot{}
		if err := rows.Scan(&h.AssetID, &h.AssetName, &h.Score, &h.Classification, &h.FindingCount); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ─── discovery_drift_events ─────────────────────────────────────────────────

// InsertDriftEventTx writes a drift event within an existing transaction.
func (r *Repo) InsertDriftEventTx(ctx context.Context, tx *sql.Tx, event *domain.DriftEvent) error {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return fmt.Errorf("insert drift event: %w", err)
	}
	event.TenantID = tenantID

	beforeJSON, _ := json.Marshal(event.BeforeState)
	afterJSON, _ := json.Marshal(event.AfterState)

	return tx.QueryRowContext(ctx, `
		INSERT INTO discovery_drift_events (
			tenant_id, snapshot_id, event_type, asset_id,
			before_state, after_state, severity, detected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, detected_at
	`, tenantID, event.SnapshotID, event.EventType, event.AssetID,
		beforeJSON, afterJSON, event.Severity,
	).Scan(&event.ID, &event.DetectedAt)
}

// ListDriftSinceSnapshot returns drift events for the tenant in ctx since the given snapshot.
func (r *Repo) ListDriftSinceSnapshot(ctx context.Context, snapshotID uuid.UUID, limit int) ([]*domain.DriftEvent, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("list drift since snapshot: %w", err)
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, snapshot_id, event_type, asset_id,
		       before_state, after_state, severity, detected_at
		FROM discovery_drift_events
		WHERE tenant_id = $1 AND snapshot_id = $2
		ORDER BY detected_at DESC
		LIMIT $3
	`, tenantID, snapshotID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.DriftEvent
	for rows.Next() {
		e := &domain.DriftEvent{}
		var beforeJSON, afterJSON []byte
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.SnapshotID, &e.EventType, &e.AssetID,
			&beforeJSON, &afterJSON, &e.Severity, &e.DetectedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(beforeJSON, &e.BeforeState)
		_ = json.Unmarshal(afterJSON, &e.AfterState)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ─── discovery_reports ──────────────────────────────────────────────────────

// CreateReport inserts a new report job in 'pending' status and returns the new ID.
func (r *Repo) CreateReport(ctx context.Context, report *domain.Report) error {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	report.TenantID = tenantID
	report.Status = domain.ReportPending

	return r.db.QueryRowContext(ctx, `
		INSERT INTO discovery_reports (
			tenant_id, snapshot_id, requested_by, format, status, requested_at
		) VALUES ($1, $2, $3, $4, 'pending', NOW())
		RETURNING id, requested_at
	`, tenantID, report.SnapshotID, report.RequestedBy, report.Format,
	).Scan(&report.ID, &report.RequestedAt)
}

// CompleteReport persists the generated bytes and marks the report completed.
func (r *Repo) CompleteReport(ctx context.Context, reportID uuid.UUID, content []byte, contentType string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE discovery_reports
		SET content = $1, content_type = $2, size_bytes = $3,
		    status = 'completed', completed_at = NOW()
		WHERE id = $4
	`, content, contentType, len(content), reportID)
	return err
}

// FailReport marks the report failed with an error message.
func (r *Repo) FailReport(ctx context.Context, reportID uuid.UUID, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE discovery_reports
		SET status = 'failed', error = $1, completed_at = NOW()
		WHERE id = $2
	`, errMsg, reportID)
	return err
}

// GetReport returns a report by ID, scoped to the tenant in ctx.
func (r *Repo) GetReport(ctx context.Context, reportID uuid.UUID) (*domain.Report, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get report: %w", err)
	}

	rep := &domain.Report{}
	var snapshotID, requestedBy sql.NullString
	var contentType, errStr sql.NullString
	var completedAt sql.NullTime
	var sizeBytes sql.NullInt32

	err = r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, snapshot_id, requested_by, format, status,
		       content_type, error, requested_at, completed_at, size_bytes
		FROM discovery_reports
		WHERE id = $1 AND tenant_id = $2
	`, reportID, tenantID).Scan(
		&rep.ID, &rep.TenantID, &snapshotID, &requestedBy, &rep.Format, &rep.Status,
		&contentType, &errStr, &rep.RequestedAt, &completedAt, &sizeBytes,
	)
	if err == sql.ErrNoRows {
		return nil, ErrReportNotFound
	}
	if err != nil {
		return nil, err
	}
	if snapshotID.Valid {
		id, _ := uuid.Parse(snapshotID.String)
		rep.SnapshotID = &id
	}
	if requestedBy.Valid {
		id, _ := uuid.Parse(requestedBy.String)
		rep.RequestedBy = &id
	}
	if contentType.Valid {
		rep.ContentType = contentType.String
	}
	if errStr.Valid {
		rep.Error = errStr.String
	}
	if completedAt.Valid {
		rep.CompletedAt = &completedAt.Time
	}
	if sizeBytes.Valid {
		rep.SizeBytes = int(sizeBytes.Int32)
	}
	return rep, nil
}

// GetReportContent returns just the bytes for download — content is intentionally
// not part of the JSON read path.
func (r *Repo) GetReportContent(ctx context.Context, reportID uuid.UUID) ([]byte, string, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("get report content: %w", err)
	}
	var content []byte
	var contentType sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT content, content_type FROM discovery_reports
		WHERE id = $1 AND tenant_id = $2 AND status = 'completed'
	`, reportID, tenantID).Scan(&content, &contentType)
	if err == sql.ErrNoRows {
		return nil, "", ErrReportNotReady
	}
	if err != nil {
		return nil, "", err
	}
	ct := "application/octet-stream"
	if contentType.Valid {
		ct = contentType.String
	}
	return content, ct, nil
}

// ListReports returns recent report jobs for the tenant in ctx.
func (r *Repo) ListReports(ctx context.Context, limit, offset int) ([]*domain.Report, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, format, status, requested_at, completed_at, size_bytes
		FROM discovery_reports
		WHERE tenant_id = $1
		ORDER BY requested_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Report
	for rows.Next() {
		rep := &domain.Report{}
		var completedAt sql.NullTime
		var sizeBytes sql.NullInt32
		if err := rows.Scan(
			&rep.ID, &rep.TenantID, &rep.Format, &rep.Status,
			&rep.RequestedAt, &completedAt, &sizeBytes,
		); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			rep.CompletedAt = &completedAt.Time
		}
		if sizeBytes.Valid {
			rep.SizeBytes = int(sizeBytes.Int32)
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}
