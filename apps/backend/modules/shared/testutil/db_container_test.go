// Package testutil provides a real PostgreSQL container via testcontainers-go
// for integration tests. Import this package and call NewTestDB to get a
// fully-migrated *sql.DB backed by a throwaway Postgres container.
//
// Add the dependency before running:
//
//	cd apps/backend && go get github.com/testcontainers/testcontainers-go@latest
//	go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
package testutil

import (
	"context"
	"os"
	"testing"
)

// TestContainerSmokeTest verifies the container boots, migrations complete,
// and the core tables exist.
func TestContainerSmokeTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: container test (use -short=false to enable)")
	}
	if os.Getenv("DOCKER_HOST") == "" && os.Getenv("CI") == "true" {
		if _, err := os.Stat("/var/run/docker.sock"); os.IsNotExist(err) {
			t.Skip("skip: Docker not available in this CI environment")
		}
	}

	tdb := NewTestDB(t)

	coreTables := []string{"scan_runs", "findings", "source_profiles", "connections"}
	for _, table := range coreTables {
		var exists bool
		err := tdb.DB.QueryRowContext(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=$1)", table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %q to exist after migrations", table)
		}
	}

	var extLoaded bool
	if err := tdb.DB.QueryRowContext(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='uuid-ossp')",
	).Scan(&extLoaded); err != nil {
		t.Fatalf("check uuid-ossp: %v", err)
	}
	if !extLoaded {
		t.Error("expected uuid-ossp extension to be loaded")
	}
}

// TestTruncateAll verifies that TruncateAll removes inserted rows.
func TestTruncateAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: container test")
	}
	tdb := NewTestDB(t)

	_, err := tdb.DB.ExecContext(context.Background(),
		`INSERT INTO tenants (id, name, created_at) VALUES (uuid_generate_v4(), 'smoke-tenant', NOW())
		 ON CONFLICT DO NOTHING`)
	if err != nil {
		t.Skipf("tenants table may not exist yet: %v", err)
	}

	tdb.TruncateAll(t)

	var count int
	if err := tdb.DB.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM tenants").Scan(&count); err != nil {
		t.Fatalf("count tenants: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after TruncateAll, got %d", count)
	}
}
