package warehouses

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/arc-platform/go-scanner/internal/connectors"
	_ "github.com/snowflakedb/gosnowflake"
)

// SnowflakeConnector scans a Snowflake data warehouse.
// Config keys: account, user, password, database, warehouse, schema (optional), sample_size (default 1000, max 50000)
type SnowflakeConnector struct {
	db         *sql.DB
	sampleSize int
}

func (c *SnowflakeConnector) SourceType() string { return "snowflake" }

func (c *SnowflakeConnector) Connect(ctx context.Context, cfg map[string]any) error {
	account := fmt.Sprintf("%v", cfg["account"])
	user := fmt.Sprintf("%v", cfg["user"])
	pass := fmt.Sprintf("%v", cfg["password"])
	database := fmt.Sprintf("%v", cfg["database"])
	warehouse := fmt.Sprintf("%v", cfg["warehouse"])
	if warehouse == "<nil>" {
		warehouse = ""
	}
	schema := fmt.Sprintf("%v", cfg["schema"])
	if schema == "<nil>" {
		schema = ""
	}

	dsn := fmt.Sprintf("%s:%s@%s/%s", user, pass, account, database)
	if warehouse != "" {
		dsn += "?warehouse=" + warehouse
		if schema != "" {
			dsn += "&schema=" + schema
		}
	} else if schema != "" {
		dsn += "?schema=" + schema
	}

	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		return err
	}
	c.db = db
	c.sampleSize = cfgInt(cfg, "sample_size", 1000, 50000)
	return db.PingContext(ctx)
}

func (c *SnowflakeConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *SnowflakeConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		rows, err := c.db.QueryContext(ctx,
			`SELECT TABLE_SCHEMA, TABLE_NAME FROM INFORMATION_SCHEMA.TABLES
			 WHERE TABLE_TYPE='BASE TABLE'`)
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
		if err := rows.Err(); err != nil {
			errc <- fmt.Errorf("rows iteration error: %w", err)
			return
		}

		for _, t := range tables {
			query := fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT %d`, t.schema, t.name, c.sampleSize)
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
						SourcePath:   fmt.Sprintf("%s.%s.%s", t.schema, t.name, col),
						IsStructured: true,
					}:
					case <-ctx.Done():
						return
					}
				}
			}
			if err := dataRows.Err(); err != nil {
				log.Printf("WARN: dataRows iteration error for table %s.%s: %v", t.schema, t.name, err)
			}
		}
	}()
	return out, errc
}
