package database

import (
	"context"
	"database/sql"
	"fmt"
)

// SetTenantContext sets the PostgreSQL session variable used by RLS policies.
// Must be called inside an open transaction — SET LOCAL is transaction-scoped.
//
// TODO: all repository calls that touch tenant-scoped tables should be wrapped
// in a transaction that begins with SetTenantContext so RLS is consistently enforced.
func SetTenantContext(ctx context.Context, tx *sql.Tx, tenantID string) error {
	if tenantID == "" {
		return fmt.Errorf("SetTenantContext: tenantID must not be empty")
	}
	_, err := tx.ExecContext(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID)
	return err
}
