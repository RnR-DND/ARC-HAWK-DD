package files

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// TextConnector reads a plain text file (or directory of .txt files) line by line.
// Config keys: path (file or directory)
type TextConnector struct {
	path string
}

func (c *TextConnector) SourceType() string { return "text" }

func (c *TextConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("text: path is required")
	}
	return nil
}

func (c *TextConnector) Close() error { return nil }

func (c *TextConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		info, err := os.Stat(c.path)
		if err != nil {
			errc <- err
			return
		}

		if info.IsDir() {
			entries, err := os.ReadDir(c.path)
			if err != nil {
				errc <- err
				return
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if !strings.HasSuffix(strings.ToLower(e.Name()), ".txt") {
					continue
				}
				c.streamFile(ctx, c.path+"/"+e.Name(), out)
			}
		} else {
			c.streamFile(ctx, c.path, out)
		}
	}()
	return out, errc
}

func (c *TextConnector) streamFile(ctx context.Context, path string, out chan<- connectors.FieldRecord) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(io.LimitReader(f, 10*1024*1024))
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		if line == "" {
			continue
		}
		select {
		case out <- connectors.FieldRecord{
			Value:        line,
			FieldName:    "line",
			SourcePath:   fmt.Sprintf("%s:line_%d", path, lineNum),
			IsStructured: false,
		}:
		case <-ctx.Done():
			return
		}
	}
}
