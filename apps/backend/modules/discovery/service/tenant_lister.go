package service

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// TenantLister returns the IDs of every active tenant.
// The discovery snapshot worker depends on this to iterate across tenants without
// holding a tenant context. Implementation reads from the tenants table.
type TenantLister interface {
	ListActiveTenants(ctx context.Context) ([]uuid.UUID, error)
}

// dbTenantLister is the production implementation, backed by the shared DB.
type dbTenantLister struct {
	db *sql.DB
}

// NewTenantListerFromDB returns a TenantLister that queries tenants directly.
func NewTenantListerFromDB(db *sql.DB) TenantLister {
	return &dbTenantLister{db: db}
}

func (l *dbTenantLister) ListActiveTenants(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT id FROM tenants WHERE is_active = true ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
