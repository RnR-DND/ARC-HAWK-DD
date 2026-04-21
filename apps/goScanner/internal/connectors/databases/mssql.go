package databases

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
	_ "github.com/microsoft/go-mssqldb"
)

// MSSQLConnector scans a Microsoft SQL Server database.
type MSSQLConnector struct{ db *sql.DB }

func (c *MSSQLConnector) SourceType() string { return "mssql" }

func (c *MSSQLConnector) Connect(ctx context.Context, config map[string]any) error {
	host := cfgString(config, "host")
	port := cfgString(config, "port")
	if port == "" {
		port = "1433"
	}
	user := cfgString(config, "user", "username")
	pass := cfgString(config, "password")
	dbname := cfgString(config, "dbname", "database")
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s", user, pass, host, port, dbname)
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return err
	}
	applyPoolDefaults(db)
	c.db = db
	return db.PingContext(ctx)
}

func (c *MSSQLConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *MSSQLConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
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

		for _, t := range tables {
			query := fmt.Sprintf(`SELECT TOP 10000 * FROM [%s].[%s]`, t.schema, t.name)
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
