package files

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/linkedin/goavro/v2"
)

// AvroConnector reads Avro container files and yields decoded record fields.
// Config keys: path (file path)
type AvroConnector struct {
	path string
}

func (c *AvroConnector) SourceType() string { return "avro" }

func (c *AvroConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("avro: path is required")
	}
	return nil
}

func (c *AvroConnector) Close() error { return nil }

func (c *AvroConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		data, err := os.ReadFile(c.path)
		if err != nil {
			errc <- err
			return
		}

		ocfReader, err := goavro.NewOCFReader(bytes.NewReader(data))
		if err != nil {
			errc <- fmt.Errorf("avro: open OCF: %w", err)
			return
		}

		rowNum := 0
		for ocfReader.Scan() {
			datum, err := ocfReader.Read()
			if err != nil {
				continue
			}
			rowNum++
			if m, ok := datum.(map[string]interface{}); ok {
				for key, val := range m {
					if val == nil {
						continue
					}
					strVal := fmt.Sprintf("%v", val)
					if strVal == "" {
						continue
					}
					select {
					case out <- connectors.FieldRecord{
						Value:        strVal,
						FieldName:    key,
						SourcePath:   fmt.Sprintf("%s:row_%d.%s", c.path, rowNum, key),
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
