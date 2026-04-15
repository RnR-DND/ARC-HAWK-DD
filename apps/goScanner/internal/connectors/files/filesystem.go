package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// FilesystemConnector walks a local directory and yields file contents.
// Config keys: path (required), max_file_size_mb (default 10)
type FilesystemConnector struct {
	rootPath string
	maxBytes int64
}

func (c *FilesystemConnector) SourceType() string { return "filesystem" }

func (c *FilesystemConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.rootPath = fmt.Sprintf("%v", cfg["path"])
	if c.rootPath == "" || c.rootPath == "<nil>" {
		return fmt.Errorf("filesystem: path is required")
	}
	c.maxBytes = 10 * 1024 * 1024 // 10MB default
	return nil
}

func (c *FilesystemConnector) Close() error { return nil }

var readableExts = map[string]bool{
	".txt": true, ".csv": true, ".json": true, ".xml": true,
	".log": true, ".md": true, ".yaml": true, ".yml": true,
	".html": true, ".htm": true, ".tsv": true, ".conf": true,
	".cfg": true, ".ini": true, ".env": true, ".sql": true,
	".sh": true, ".go": true, ".py": true, ".js": true,
}

func (c *FilesystemConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		err := filepath.WalkDir(c.rootPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			ext := strings.ToLower(filepath.Ext(path))
			if !readableExts[ext] {
				return nil
			}

			info, err := d.Info()
			if err != nil || info.Size() > c.maxBytes {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			data, err := io.ReadAll(io.LimitReader(f, c.maxBytes))
			f.Close()
			if err != nil {
				return nil
			}

			rel, _ := filepath.Rel(c.rootPath, path)
			select {
			case out <- connectors.FieldRecord{
				Value:        string(data),
				FieldName:    filepath.Base(path),
				SourcePath:   rel,
				IsStructured: ext == ".csv" || ext == ".tsv",
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
		if err != nil && err != ctx.Err() {
			errc <- err
		}
	}()
	return out, errc
}
