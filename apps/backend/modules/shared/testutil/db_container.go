// Package testutil provides a real PostgreSQL container via testcontainers-go
// for integration tests. Import this package and call NewTestDB to get a
// fully-migrated *sql.DB backed by a throwaway Postgres container.
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
	TestDBName     = "arc_hawk_test"
	TestDBUser     = "arc_test"
	TestDBPassword = "arc_test_secret"
	PostgresImage  = "postgres:16-alpine"
)

// TestDB holds the container and the open database handle for one test run.
type TestDB struct {
	DB        *sql.DB
	DSN       string
	container testcontainers.Container
}

// NewTestDB spins up a throwaway PostgreSQL container, runs all versioned
// migrations from apps/backend/migrations_versioned/, and returns a ready
// *TestDB. Call t.Cleanup (registered internally) to terminate the container.
//
// Skips automatically when Docker is unavailable (CI without Docker or dev
// machine without Docker running).
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	if os.Getenv("DOCKER_HOST") == "" {
		if _, err := os.Stat("/var/run/docker.sock"); os.IsNotExist(err) {
			if _, err2 := os.Stat(os.Getenv("HOME") + "/.docker/run/docker.sock"); os.IsNotExist(err2) {
				t.Skip("skip: Docker not available")
			}
		}
	}

	ctx := context.Background()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage(PostgresImage),
		postgres.WithDatabase(TestDBName),
		postgres.WithUsername(TestDBUser),
		postgres.WithPassword(TestDBPassword),
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
		TestDBUser, TestDBPassword, host, port.Port(), TestDBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("testutil: open db: %v", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := waitForDB(db, 10, 500*time.Millisecond); err != nil {
		t.Fatalf("testutil: db not ready: %v", err)
	}

	migrationsPath := ResolveMigrationsPath(t)
	RunMigrations(t, dsn, migrationsPath)

	t.Cleanup(func() { _ = db.Close() })

	return &TestDB{DB: db, DSN: dsn, container: pgContainer}
}

// TruncateAll removes all data from core tables between tests.
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

func waitForDB(db *sql.DB, attempts int, delay time.Duration) error {
	for i := 0; i < attempts; i++ {
		if err := db.Ping(); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("database not ready after %d attempts", attempts)
}

func RunMigrations(t *testing.T, dsn, migrationsPath string) {
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

// ResolveMigrationsPath walks upward from the caller's file to find migrations_versioned/.
func ResolveMigrationsPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	base := filepath.Dir(thisFile)
	for i := 0; i < 3; i++ {
		base = filepath.Dir(base)
	}
	p := filepath.Join(base, "migrations_versioned")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		// Try walking up one more level (called from a different depth).
		p = filepath.Join(filepath.Dir(base), "migrations_versioned")
		if _, err2 := os.Stat(p); os.IsNotExist(err2) {
			t.Fatalf("migrations_versioned not found near %s", base)
		}
	}
	return p
}
