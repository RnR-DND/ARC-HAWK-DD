package files

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// DOCXConnector extracts text from DOCX files (Office Open XML format).
// DOCX is a ZIP archive containing word/document.xml with the text content.
// No external dependencies — uses stdlib archive/zip + encoding/xml.
// Config keys: path (file path)
type DOCXConnector struct {
	path string
}

func (c *DOCXConnector) SourceType() string { return "docx" }

func (c *DOCXConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("docx: path is required")
	}
	return nil
}

func (c *DOCXConnector) Close() error { return nil }

func (c *DOCXConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord, 100)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)

		text, err := extractDOCXText(c.path)
		if err != nil {
			errc <- err
			return
		}

		for i, para := range strings.Split(text, "\n") {
			para = strings.TrimSpace(para)
			if para == "" {
				continue
			}
			select {
			case out <- connectors.FieldRecord{
				Value:        para,
				FieldName:    "paragraph",
				SourcePath:   fmt.Sprintf("%s:para_%d", c.path, i),
				IsStructured: false,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errc
}

func extractDOCXText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("docx: open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		text, err := extractXMLText(rc)
		rc.Close()
		return text, err
	}
	return "", fmt.Errorf("docx: word/document.xml not found")
}

// extractXMLText pulls all text content from XML w:t elements.
func extractXMLText(r io.Reader) (string, error) {
	var buf strings.Builder
	dec := xml.NewDecoder(r)
	inText := false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return buf.String(), nil // partial result is OK
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inText = true
			} else if t.Name.Local == "p" {
				buf.WriteString("\n")
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			}
		case xml.CharData:
			if inText {
				buf.Write(t)
			}
		}
	}
	return buf.String(), nil
}
