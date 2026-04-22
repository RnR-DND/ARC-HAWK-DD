package databases

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

// applyPoolDefaults applies conservative pool settings to every database/sql
// connection opened by a scanner connector. Without this, Go's sql package
// uses defaults (unlimited open conns, 2 idle), which under concurrent scans
// blows through the source database's connection limit and starves other
// tenants. See system-audit P1-3.
//
// Overridable via env for ops tuning:
//
//	DB_POOL_MAX_OPEN       default 20
//	DB_POOL_MAX_IDLE       default 5
//	DB_POOL_CONN_LIFETIME  default 10 minutes
//	DB_POOL_CONN_IDLE_TIME default 5 minutes
func applyPoolDefaults(db *sql.DB) {
	if db == nil {
		return
	}
	db.SetMaxOpenConns(envPoolInt("DB_POOL_MAX_OPEN", 20))
	db.SetMaxIdleConns(envPoolInt("DB_POOL_MAX_IDLE", 5))
	db.SetConnMaxLifetime(envPoolDuration("DB_POOL_CONN_LIFETIME", 10*time.Minute))
	db.SetConnMaxIdleTime(envPoolDuration("DB_POOL_CONN_IDLE_TIME", 5*time.Minute))
}

// cfgString returns the first non-empty string value for any of the given
// keys in the config map. JSON numbers unmarshal as float64, which fmt.Sprintf
// renders with trailing ".000000" — coerce those to integer text for port
// and similar fields by trimming the fractional part when it's zero.
func cfgString(config map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := config[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			if t != "" {
				return t
			}
		case float64:
			if t == float64(int64(t)) {
				return fmt.Sprintf("%d", int64(t))
			}
			return fmt.Sprintf("%v", t)
		case int:
			return fmt.Sprintf("%d", t)
		case int64:
			return fmt.Sprintf("%d", t)
		default:
			s := fmt.Sprintf("%v", t)
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

// cfgInt returns an integer from config[key], clamped to [1, maxVal].
// Falls back to def when the key is absent, zero, or unparseable.
func cfgInt(config map[string]any, key string, def, maxVal int) int {
	v, ok := config[key]
	if !ok || v == nil {
		return def
	}
	var n int
	switch t := v.(type) {
	case float64:
		n = int(t)
	case int:
		n = t
	case int64:
		n = int(t)
	case string:
		fmt.Sscanf(t, "%d", &n) //nolint:errcheck
	}
	if n <= 0 {
		return def
	}
	if n > maxVal {
		return maxVal
	}
	return n
}

func envPoolInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func envPoolDuration(name string, def time.Duration) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}
