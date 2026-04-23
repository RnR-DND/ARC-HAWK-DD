package database

import (
	"context"
	"database/sql"
	"fmt"
)

// SetTenantContext sets the PostgreSQL session variable used by RLS policies.
// Must be called inside an open transaction — SET LOCAL is transaction-scoped.
//
// Tenant isolation strategy (two layers):
//  1. Application layer: all repository queries include WHERE tenant_id = $n.
//  2. Database layer: RLS policies on tenant-scoped tables enforce isolation via
//     app.current_tenant_id. Call SetTenantContext at the start of any transaction
//     that touches tables with RLS enabled (scan_runs, findings, assets, connections,
//     audit_logs, fp_learning). Tables without RLS rely on application-layer filtering.
func SetTenantContext(ctx context.Context, tx *sql.Tx, tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("SetTenantContext: tenantID must not be empty")
	}
	_, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	return err
}
