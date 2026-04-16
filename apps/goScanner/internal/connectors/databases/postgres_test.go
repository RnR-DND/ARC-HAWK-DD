package databases

import (
	"context"
	"testing"
	"time"
)

func TestPostgresConnector_SourceType(t *testing.T) {
	c := &PostgresConnector{}
	if got := c.SourceType(); got != "postgresql" {
		t.Errorf("SourceType() = %q, want %q", got, "postgresql")
	}
}

func TestPostgresConnector_CloseNilDB(t *testing.T) {
	c := &PostgresConnector{db: nil}
	if err := c.Close(); err != nil {
		t.Errorf("Close() with nil db returned unexpected error: %v", err)
	}
}

func TestPostgresConnector_ConnectBadHost(t *testing.T) {
	c := &PostgresConnector{}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	config := map[string]any{
		"host":     "127.0.0.1",
		"port":     "9999", // nothing listening here
		"user":     "nobody",
		"password": "secret",
		"dbname":   "nonexistent",
	}
	err := c.Connect(ctx, config)
	if err == nil {
		t.Error("Connect() with unreachable host should return an error")
		_ = c.Close()
	}
}

func TestPostgresConnector_DefaultPort(t *testing.T) {
	// Verify the connector tolerates a missing port config key (falls back to 5432).
	// The connection will still fail — we just ensure no panic from port handling.
	c := &PostgresConnector{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	config := map[string]any{
		"host":   "127.0.0.1",
		"user":   "nobody",
		"dbname": "testdb",
		// "port" intentionally omitted
	}
	// Expect an error (no DB at 5432 in CI), but not a panic.
	_ = c.Connect(ctx, config)
}
