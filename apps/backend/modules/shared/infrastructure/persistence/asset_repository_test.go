package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPostgresRepository_ListAssets_TenantIsolation(t *testing.T) {
	// Setup mock DB
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewPostgresRepository(db)

	// Setup Test Data
	tenantID := uuid.New()
	tenantIDStr := tenantID.String()
	ctx := context.WithValue(context.Background(), "tenant_id", tenantIDStr)

	// Expectation: Query MUST include "WHERE tenant_id = $1"
	// We use regex to match the query flexible
	query := `SELECT id, tenant_id, .* FROM assets WHERE tenant_id = \$1 ORDER BY risk_score DESC LIMIT \$2 OFFSET \$3`

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "stable_id", "asset_type", "name", "path", "data_source", "host",
		"environment", "owner", "source_system", "file_metadata", "risk_score", "total_findings",
		"created_at", "updated_at",
	}).AddRow(
		uuid.New(), tenantID, "stable-1", "file", "Test Asset", "/tmp/test", "filesystem", "localhost",
		"prod", "admin", "scanner", nil, 100, 5, time.Now(), time.Now(),
	)

	mock.ExpectQuery(query).
		WithArgs(tenantID, 10, 0).
		WillReturnRows(rows)

	// Action
	results, err := repo.ListAssets(ctx, 10, 0)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, tenantID, results[0].TenantID)

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepository_ListAssets_MissingTenantID(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewPostgresRepository(db)

	// Context WITHOUT tenant_id
	ctx := context.Background()

	// Action
	results, err := repo.ListAssets(ctx, 10, 0)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "tenant_id missing")
}
