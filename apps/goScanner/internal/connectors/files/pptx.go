package files

import (
	"archive/zip"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// PPTXConnector extracts text from PPTX files (Office Open XML format).
// PPTX is a ZIP archive containing ppt/slides/slide*.xml files.
// Config keys: path (file path)
type PPTXConnector struct {
	path string
}

func (c *PPTXConnector) SourceType() string { return "pptx" }

func (c *PPTXConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("pptx: path is required")
	}
	return nil
}

func (c *PPTXConnector) Close() error { return nil }

func (c *PPTXConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		r, err := zip.OpenReader(c.path)
		if err != nil {
			errc <- err
			return
		}
		defer r.Close()

		slideNum := 0
		for _, f := range r.File {
			matched, _ := filepath.Match("ppt/slides/slide*.xml", f.Name)
			if !matched {
				continue
			}
			slideNum++
			rc, err := f.Open()
			if err != nil {
				continue
			}
			text, err := extractXMLText(rc)
			rc.Close()
			if err != nil {
				continue
			}
			for i, para := range strings.Split(text, "\n") {
				para = strings.TrimSpace(para)
				if para == "" {
					continue
				}
				select {
				case out <- connectors.FieldRecord{
					Value:        para,
					FieldName:    "text",
					SourcePath:   fmt.Sprintf("%s:slide_%d:para_%d", c.path, slideNum, i),
					IsStructured: false,
				}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, errc
}
