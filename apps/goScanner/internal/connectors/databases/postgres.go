package databases

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/arc-platform/go-scanner/internal/connectors"
	_ "github.com/lib/pq"
)

// PostgresConnector scans a PostgreSQL database.
type PostgresConnector struct {
	db         *sql.DB
	sampleSize int
}

func (c *PostgresConnector) SourceType() string { return "postgresql" }

func (c *PostgresConnector) Connect(ctx context.Context, config map[string]any) error {
	host := cfgString(config, "host")
	user := cfgString(config, "user", "username")
	pass := cfgString(config, "password")
	dbname := cfgString(config, "dbname", "database")
	port := cfgString(config, "port")
	if port == "" {
		port = "5432"
	}
	// sslmode is configurable per-source; defaults to "prefer" (opportunistic
	// TLS) to avoid plaintext when the server supports it, while still
	// allowing connection to dev/test databases without certs. Override with
	// sslmode in the connection config, or set DB_SSLMODE_DEFAULT for a
	// stricter default (e.g. "verify-full") in release environments.
	sslmode := cfgString(config, "sslmode")
	if sslmode == "" {
		if v := os.Getenv("DB_SSLMODE_DEFAULT"); v != "" {
			sslmode = v
		} else {
			sslmode = "prefer"
		}
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", host, user, pass, dbname, port, sslmode)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	applyPoolDefaults(db)
	c.db = db
	c.sampleSize = cfgInt(config, "sample_size", 1000, 50000)
	return db.PingContext(ctx)
}

func (c *PostgresConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *PostgresConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)
		// Exclude arc-hawk's own bookkeeping tables. When the scanner points
		// at the backend's own database, these tables would otherwise be
		// scanned and produce recursive findings ("findings about findings")
		// plus huge false-positive counts from hex hash columns in audit_logs.
		rows, err := c.db.QueryContext(ctx,
			`SELECT table_schema, table_name FROM information_schema.tables
			 WHERE table_type='BASE TABLE'
			   AND table_schema NOT IN ('pg_catalog','information_schema')
			   AND table_name NOT IN (
			     'findings','classifications','assets','asset_relationships',
			     'scan_runs','audit_logs','api_keys','custom_patterns','patterns',
			     'connections','policies','agent_sync_log','consent_records',
			     'data_principal_requests','discovery_drift_events','discovery_inventory',
			     'discovery_reports','discovery_risk_scores','discovery_snapshots',
			     'discovery_snapshot_facts','masking_audit_log','tenants',
			     'schema_migrations','users','user_settings'
			   )`)
		if err != nil {
			errc <- err
			return
		}
		defer rows.Close()
		type tableRef struct{ schema, name string }
		var tables []tableRef
		for rows.Next() {
			var t tableRef
			if err := rows.Scan(&t.schema, &t.name); err == nil {
				tables = append(tables, t)
			}
		}
		for _, t := range tables {
			query := fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT %d`, t.schema, t.name, c.sampleSize)
			dataRows, err := c.db.QueryContext(ctx, query)
			if err != nil {
				continue
			}
			cols, _ := dataRows.Columns()
			for dataRows.Next() {
				vals := make([]interface{}, len(cols))
				ptrs := make([]interface{}, len(cols))
				for i := range vals {
					ptrs[i] = &vals[i]
				}
				if err := dataRows.Scan(ptrs...); err != nil {
					continue
				}
				for i, col := range cols {
					if vals[i] == nil {
						continue
					}
					strVal := fmt.Sprintf("%v", vals[i])
					if strVal == "" {
						continue
					}
					select {
					case out <- connectors.FieldRecord{
						Value:        strVal,
						FieldName:    col,
						SourcePath:   fmt.Sprintf("%s.%s.%s", t.schema, t.name, col),
						IsStructured: true,
					}:
					case <-ctx.Done():
						dataRows.Close()
						return
					}
				}
			}
			dataRows.Close()
		}
	}()
	return out, errc
}
