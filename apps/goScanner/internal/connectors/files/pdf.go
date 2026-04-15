package files

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// PDFConnector extracts readable text from PDF files using raw byte scanning.
// This approach extracts printable ASCII strings ≥6 chars embedded in the PDF byte stream,
// similar to the Unix `strings` utility. Adequate for plain-text PDFs.
// For scanned PDFs, use the scanned_images connector with OCR.
// Config keys: path (file path), min_string_length (default 6)
type PDFConnector struct {
	path      string
	minLength int
}

func (c *PDFConnector) SourceType() string { return "pdf" }

func (c *PDFConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("pdf: path is required")
	}
	c.minLength = 6
	return nil
}

func (c *PDFConnector) Close() error { return nil }

func (c *PDFConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
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

		strings_ := extractPrintableStrings(data, c.minLength)
		for i, s := range strings_ {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			select {
			case out <- connectors.FieldRecord{
				Value:        s,
				FieldName:    "text",
				SourcePath:   fmt.Sprintf("%s:str_%d", c.path, i),
				IsStructured: false,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errc
}

// extractPrintableStrings finds sequences of printable ASCII chars (length >= minLen).
func extractPrintableStrings(data []byte, minLen int) []string {
	var results []string
	var current strings.Builder

	flush := func() {
		if current.Len() >= minLen {
			results = append(results, current.String())
		}
		current.Reset()
	}

	for _, b := range data {
		r := rune(b)
		if unicode.IsPrint(r) && b < 128 {
			current.WriteByte(b)
			if current.Len() > 200 {
				flush()
			}
		} else {
			flush()
		}
	}
	flush()
	return results
}
