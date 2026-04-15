package files

import (
	"context"
	"fmt"
	"os"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/scritchley/orc"
)

// ORCConnector reads Apache ORC files and yields field values.
// Config keys: path (file path)
type ORCConnector struct {
	path string
}

func (c *ORCConnector) SourceType() string { return "orc" }

func (c *ORCConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("orc: path is required")
	}
	return nil
}

func (c *ORCConnector) Close() error { return nil }

// sizedFile wraps *os.File to implement orc.SizedReaderAt.
type sizedFile struct {
	*os.File
}

func (sf *sizedFile) Size() int64 {
	info, err := sf.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

func (c *ORCConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
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

		r, err := orc.NewReader(&sizedFile{f})
		if err != nil {
			errc <- fmt.Errorf("orc: open reader: %w", err)
			return
		}

		// Get column names from schema
		schema := r.Schema()
		colNames := schema.Columns()
		if len(colNames) == 0 {
			return
		}

		cursor := r.Select(colNames...)
		rowNum := 0
		for cursor.Stripes() {
			for cursor.Next() {
				rowNum++
				row := cursor.Row()
				for j, val := range row {
					if val == nil || j >= len(colNames) {
						continue
					}
					strVal := fmt.Sprintf("%v", val)
					if strVal == "" {
						continue
					}
					select {
					case out <- connectors.FieldRecord{
						Value:        strVal,
						FieldName:    colNames[j],
						SourcePath:   fmt.Sprintf("%s:row_%d.%s", c.path, rowNum, colNames[j]),
						IsStructured: true,
					}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out, errc
}
