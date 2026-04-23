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
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDBName     = "arc_hawk_test"
	testDBUser     = "arc_test"
	testDBPassword = "arc_test_secret"
	postgresImage  = "postgres:16-alpine"
)

// TestDB holds the container and the open database handle for one test run.
type TestDB struct {
	DB        *sql.DB
	DSN       string
	container testcontainers.Container
}

// NewTestDB spins up a throwaway PostgreSQL container, runs all versioned
// migrations from apps/backend/migrations_versioned/, and returns a ready
// *TestDB. Call Cleanup() in a t.Cleanup to terminate the container.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage(postgresImage),
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("testcontainers: start postgres: %v", err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("testcontainers: terminate postgres: %v", err)
		}
	})

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("testcontainers: get host: %v", err)
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("testcontainers: get port: %v", err)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		testDBUser, testDBPassword, host, port.Port(), testDBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("testutil: open db: %v", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connectivity with retries.
	if err := waitForDB(db, 10, 500*time.Millisecond); err != nil {
		t.Fatalf("testutil: db not ready: %v", err)
	}

	migrationsPath := resolveMigrationsPath(t)
	runMigrations(t, dsn, migrationsPath)

	t.Cleanup(func() { _ = db.Close() })

	return &TestDB{DB: db, DSN: dsn, container: pgContainer}
}

// TruncateAll removes all data from core tables between tests without
// dropping schema. Call at the top of each test that needs a clean slate.
func (td *TestDB) TruncateAll(t *testing.T) {
	t.Helper()
	tables := []string{
		"findings", "scan_runs", "source_profiles", "connections",
		"assets", "patterns", "audit_logs", "users", "tenants",
	}
	for _, tbl := range tables {
		if _, err := td.DB.ExecContext(context.Background(),
			fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tbl)); err != nil {
			t.Logf("testutil: truncate %s: %v (may not exist yet)", tbl, err)
		}
	}
}

// ---- integration smoke test ------------------------------------------------

// TestContainerSmokeTest verifies the container boots, migrations complete,
// and the core tables exist.
func TestContainerSmokeTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: container test (use -short=false to enable)")
	}
	// Skip in CI if Docker is unavailable.
	if os.Getenv("DOCKER_HOST") == "" && os.Getenv("CI") == "true" {
		if _, err := os.Stat("/var/run/docker.sock"); os.IsNotExist(err) {
			t.Skip("skip: Docker not available in this CI environment")
		}
	}

	tdb := NewTestDB(t)

	// Verify core tables were created by migration 000001.
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

	// Verify uuid-ossp extension loaded (required by schema).
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

	// Insert a minimal tenant row then truncate.
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

// ---- helpers ---------------------------------------------------------------

func waitForDB(db *sql.DB, attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		if err := db.Ping(); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("database not ready after %d attempts", attempts)
}

func runMigrations(t *testing.T, dsn, migrationsPath string) {
	t.Helper()
	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		t.Fatalf("migrate.New: %v", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate.Up: %v", err)
	}
}

// resolveMigrationsPath walks upward from this file to find migrations_versioned/.
func resolveMigrationsPath(t *testing.T) string {
	t.Helper()
	// __file__ is this test file's path — walk up to apps/backend/.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile: .../apps/backend/modules/shared/testutil/db_container_test.go
	// Walk up 3 levels from testutil/ to reach apps/backend/.
	base := filepath.Dir(thisFile)
	for i := 0; i < 3; i++ {
		base = filepath.Dir(base)
	}
	p := filepath.Join(base, "migrations_versioned")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatalf("migrations_versioned not found at %s", p)
	}
	return p
}
