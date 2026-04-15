package files

import (
	"context"
	"fmt"
	"os"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/parquet-go/parquet-go"
)

// ParquetConnector reads Apache Parquet files and yields field values.
// Config keys: path (file path)
type ParquetConnector struct {
	path string
}

func (c *ParquetConnector) SourceType() string { return "parquet" }

func (c *ParquetConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("parquet: path is required")
	}
	return nil
}

func (c *ParquetConnector) Close() error { return nil }

func (c *ParquetConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		f, err := os.Open(c.path)
		if err != nil {
			errc <- err
			return
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			errc <- err
			return
		}

		pf, err := parquet.OpenFile(f, info.Size())
		if err != nil {
			errc <- fmt.Errorf("parquet: open: %w", err)
			return
		}

		reader := parquet.NewReader(pf)
		defer reader.Close()

		schema := reader.Schema()
		var fieldNames []string
		for _, field := range schema.Fields() {
			fieldNames = append(fieldNames, field.Name())
		}

		rowBuf := make([]parquet.Row, 100)
		rowNum := 0
		for {
			n, err := reader.ReadRows(rowBuf)
			for i := 0; i < n; i++ {
				rowNum++
				for j, v := range rowBuf[i] {
					if j >= len(fieldNames) {
						continue
					}
					strVal := v.String()
					if strVal == "" || strVal == "<null>" {
						continue
					}
					select {
					case out <- connectors.FieldRecord{
						Value:        strVal,
						FieldName:    fieldNames[j],
						SourcePath:   fmt.Sprintf("%s:row_%d.%s", c.path, rowNum, fieldNames[j]),
						IsStructured: true,
					}:
					case <-ctx.Done():
						return
					}
				}
			}
			if err != nil || n == 0 {
				break
			}
		}
	}()
	return out, errc
}
