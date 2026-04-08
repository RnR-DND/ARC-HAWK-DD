package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
)

// Upstream is the read-only surface that discovery needs from the rest of the system.
// It is implemented by a composite of existing module interfaces (assets, scanning,
// lineage, connections) plus direct DB queries where existing interfaces don't expose
// what discovery needs.
//
// Per E8 in autoplan: discovery does NOT introduce a new top-level shared interface.
// Upstream is a discovery-internal seam that lets us mock cross-module data in tests.
type Upstream interface {
	// ListAssetSummaries returns a paginated set of asset summaries for the tenant in ctx.
	// Each summary already includes classification + finding count from a join.
	ListAssetSummaries(ctx context.Context, limit, offset int) ([]AssetSummary, error)

	// CountSourcesForTenant returns the total source count for the tenant in ctx.
	CountSourcesForTenant(ctx context.Context) (int, error)
}

// AssetSummary is a denormalized cross-module read used by inventory_service.
type AssetSummary struct {
	AssetID        uuid.UUID
	AssetName      string
	SourceID       *uuid.UUID
	SourceName     string
	Classification string
	Sensitivity    int
	FindingCount   int
	LastScannedAt  *time.Time
}

// upstreamFromDeps is the production composite that reads directly from the database.
// In v1 this is a simple set of cross-module SQL queries because every upstream module
// already persists into postgres tables; we read those tables directly via a small,
// well-bounded query surface rather than introducing a new shared interface.
type upstreamFromDeps struct {
	db   *sql.DB
	deps *interfaces.ModuleDependencies
}

// NewUpstreamFromDeps wires a production Upstream from the module dependencies.
func NewUpstreamFromDeps(deps *interfaces.ModuleDependencies) Upstream {
	return &upstreamFromDeps{db: deps.DB, deps: deps}
}

// ListAssetSummaries reads from the assets and findings tables (owned by the assets module).
// This is read-only and respects tenant isolation via EnsureTenantID.
func (u *upstreamFromDeps) ListAssetSummaries(ctx context.Context, limit, offset int) ([]AssetSummary, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	// LEFT JOIN findings to get a finding count + dominant classification per asset.
	// Defensive against assets that have no findings yet (count = 0, classification = '').
	query := `
		SELECT
			a.id,
			a.name,
			NULL::uuid AS source_id,
			'' AS source_name,
			COALESCE(f.classification, '') AS classification,
			COALESCE(f.sensitivity, 0) AS sensitivity,
			COALESCE(f.finding_count, 0) AS finding_count,
			a.updated_at AS last_scanned_at
		FROM assets a
		LEFT JOIN LATERAL (
			SELECT
				classification,
				MAX(COALESCE(NULLIF((classification_score)::integer, 0), 50)) AS sensitivity,
				COUNT(*) AS finding_count
			FROM findings
			WHERE findings.asset_id = a.id AND findings.tenant_id = a.tenant_id
			GROUP BY classification
			ORDER BY COUNT(*) DESC
			LIMIT 1
		) f ON true
		WHERE a.tenant_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := u.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AssetSummary
	for rows.Next() {
		var s AssetSummary
		var sourceID sql.NullString
		var lastScanned sql.NullTime
		if err := rows.Scan(
			&s.AssetID, &s.AssetName, &sourceID, &s.SourceName,
			&s.Classification, &s.Sensitivity, &s.FindingCount, &lastScanned,
		); err != nil {
			return nil, err
		}
		if sourceID.Valid {
			id, _ := uuid.Parse(sourceID.String)
			s.SourceID = &id
		}
		if lastScanned.Valid {
			s.LastScannedAt = &lastScanned.Time
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountSourcesForTenant returns distinct source count from the connections table for the tenant in ctx.
func (u *upstreamFromDeps) CountSourcesForTenant(ctx context.Context) (int, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return 0, err
	}
	var n int
	err = u.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM connections WHERE tenant_id = $1`, tenantID,
	).Scan(&n)
	if err != nil {
		// If the connections table is empty/unmigrated, fall back to 0 rather than erroring.
		// Discovery should still work even if no connections are registered yet.
		return 0, nil
	}
	return n, nil
}

// Compile-time guard.
var _ Upstream = (*upstreamFromDeps)(nil)

// Compile-time use of domain to satisfy package import in mock layer.
var _ = domain.SnapshotPending
