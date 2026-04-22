//go:build connector_stub

package files

import (
	"context"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// ImagesConnector scans image files using OCR (Tesseract).
// Stub: requires Tesseract OCR to be installed system-wide and
// github.com/otiai10/gosseract to be added to go.mod.
// To enable: install Tesseract (`apt-get install tesseract-ocr` or brew),
// add gosseract to go.mod, and implement the StreamFields method below.
type ImagesConnector struct{}

func (c *ImagesConnector) SourceType() string { return "scanned_images" }

func (c *ImagesConnector) Connect(_ context.Context, _ map[string]any) error {
	return fmt.Errorf("scanned_images connector: requires Tesseract OCR; run: apt-get install tesseract-ocr")
}

func (c *ImagesConnector) Close() error { return nil }

func (c *ImagesConnector) StreamFields(_ context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord)
	errc := make(chan error, 1)
	close(out)
	errc <- fmt.Errorf("scanned_images connector: Tesseract OCR not available in this build")
	close(errc)
	return out, errc
}
