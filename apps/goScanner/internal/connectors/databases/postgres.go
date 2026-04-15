package databases

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
	_ "github.com/lib/pq"
)

// PostgresConnector scans a PostgreSQL database.
type PostgresConnector struct{ db *sql.DB }

func (c *PostgresConnector) SourceType() string { return "postgresql" }

func (c *PostgresConnector) Connect(ctx context.Context, config map[string]any) error {
	host := fmt.Sprintf("%v", config["host"])
	user := fmt.Sprintf("%v", config["user"])
	pass := fmt.Sprintf("%v", config["password"])
	dbname := fmt.Sprintf("%v", config["dbname"])
	port := fmt.Sprintf("%v", config["port"])
	if port == "<nil>" || port == "" {
		port = "5432"
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, pass, dbname, port)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	c.db = db
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
		rows, err := c.db.QueryContext(ctx,
			`SELECT table_schema, table_name FROM information_schema.tables
			 WHERE table_type='BASE TABLE' AND table_schema NOT IN ('pg_catalog','information_schema')`)
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
			query := fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT 10000`, t.schema, t.name)
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
