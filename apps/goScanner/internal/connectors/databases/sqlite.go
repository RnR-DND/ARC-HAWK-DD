package databases

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/arc-platform/go-scanner/internal/connectors"
	_ "modernc.org/sqlite"
)

// SQLiteConnector scans a SQLite database file.
type SQLiteConnector struct {
	db         *sql.DB
	sampleSize int
}

func (c *SQLiteConnector) SourceType() string { return "sqlite" }

func (c *SQLiteConnector) Connect(ctx context.Context, config map[string]any) error {
	path := fmt.Sprintf("%v", config["path"])
	if path == "" || path == "<nil>" {
		return fmt.Errorf("sqlite: path is required")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	c.db = db
	c.sampleSize = cfgInt(config, "sample_size", 1000, 50000)
	return db.PingContext(ctx)
}

func (c *SQLiteConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *SQLiteConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		rows, err := c.db.QueryContext(ctx,
			`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
		if err != nil {
			errc <- err
			return
		}
		defer rows.Close()
		var tables []string
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err == nil {
				tables = append(tables, t)
			}
		}
		if err := rows.Err(); err != nil {
			errc <- fmt.Errorf("rows iteration error: %w", err)
			return
		}

		for _, tbl := range tables {
			query := fmt.Sprintf(`SELECT * FROM "%s" LIMIT %d`, tbl, c.sampleSize)
			dataRows, err := c.db.QueryContext(ctx, query)
			if err != nil {
				continue
			}
			defer dataRows.Close()
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
						SourcePath:   fmt.Sprintf("%s.%s", tbl, col),
						IsStructured: true,
					}:
					case <-ctx.Done():
						return
					}
				}
			}
			if err := dataRows.Err(); err != nil {
				log.Printf("WARN: dataRows iteration error for table %s: %v", tbl, err)
			}
		}
	}()
	return out, errc
}
