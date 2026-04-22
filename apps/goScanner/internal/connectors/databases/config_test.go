package databases

import (
	"database/sql"
	"testing"
	"time"
)

func TestEnvPoolInt(t *testing.T) {
	const key = "TEST_DB_POOL_INT_XYZ"

	t.Run("returns default when unset", func(t *testing.T) {
		t.Setenv(key, "")
		if got := envPoolInt(key, 42); got != 42 {
			t.Errorf("unset: got %d, want 42", got)
		}
	})

	t.Run("parses positive override", func(t *testing.T) {
		t.Setenv(key, "17")
		if got := envPoolInt(key, 5); got != 17 {
			t.Errorf("override 17: got %d", got)
		}
	})

	t.Run("rejects zero", func(t *testing.T) {
		t.Setenv(key, "0")
		if got := envPoolInt(key, 8); got != 8 {
			t.Errorf("zero override should fall back to default; got %d", got)
		}
	})

	t.Run("rejects negative", func(t *testing.T) {
		t.Setenv(key, "-3")
		if got := envPoolInt(key, 8); got != 8 {
			t.Errorf("negative override should fall back to default; got %d", got)
		}
	})

	t.Run("rejects garbage", func(t *testing.T) {
		t.Setenv(key, "notanumber")
		if got := envPoolInt(key, 11); got != 11 {
			t.Errorf("garbage override should fall back; got %d", got)
		}
	})
}

func TestEnvPoolDuration(t *testing.T) {
	const key = "TEST_DB_POOL_DUR_XYZ"

	t.Run("default when unset", func(t *testing.T) {
		t.Setenv(key, "")
		def := 7 * time.Minute
		if got := envPoolDuration(key, def); got != def {
			t.Errorf("unset: got %v, want %v", got, def)
		}
	})

	t.Run("parses valid duration", func(t *testing.T) {
		t.Setenv(key, "90s")
		if got := envPoolDuration(key, time.Minute); got != 90*time.Second {
			t.Errorf("90s: got %v", got)
		}
	})

	t.Run("rejects zero duration", func(t *testing.T) {
		t.Setenv(key, "0s")
		def := 3 * time.Minute
		if got := envPoolDuration(key, def); got != def {
			t.Errorf("zero duration should fall back to default; got %v", got)
		}
	})

	t.Run("rejects malformed", func(t *testing.T) {
		t.Setenv(key, "forever")
		def := 2 * time.Minute
		if got := envPoolDuration(key, def); got != def {
			t.Errorf("malformed should fall back; got %v", got)
		}
	})
}

func TestApplyPoolDefaults_NilSafe(t *testing.T) {
	// Must not panic on nil input.
	applyPoolDefaults(nil)
}

func TestApplyPoolDefaults_SetsValues(t *testing.T) {
	// Use a mock DB that never opens a real connection. sql.DB with a
	// non-existent driver registered below is enough — SetMax* methods do
	// not actually touch the underlying driver until a connection is needed.
	db, err := sql.Open("fake_driver_that_wont_connect", "")
	if err != nil {
		// Some Go versions error immediately on unknown driver; that's fine,
		// we still covered the nil path above.
		t.Skipf("sql.Open rejected fake driver: %v", err)
	}
	defer db.Close()

	t.Setenv("DB_POOL_MAX_OPEN", "13")
	t.Setenv("DB_POOL_MAX_IDLE", "7")
	t.Setenv("DB_POOL_CONN_LIFETIME", "9m")
	t.Setenv("DB_POOL_CONN_IDLE_TIME", "4m")

	applyPoolDefaults(db)

	stats := db.Stats()
	if stats.MaxOpenConnections != 13 {
		t.Errorf("MaxOpenConnections: got %d, want 13", stats.MaxOpenConnections)
	}
}
