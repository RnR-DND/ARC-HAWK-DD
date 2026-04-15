package databases

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
	_ "modernc.org/sqlite"
)

// SQLiteConnector scans a SQLite database file.
type SQLiteConnector struct{ db *sql.DB }

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

		for _, tbl := range tables {
			query := fmt.Sprintf(`SELECT * FROM "%s" LIMIT 10000`, tbl)
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
						SourcePath:   fmt.Sprintf("%s.%s", tbl, col),
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
