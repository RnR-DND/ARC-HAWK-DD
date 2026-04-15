package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
	"golang.org/x/net/html"
)

// HTMLConnector extracts visible text from HTML files.
// Config keys: path (file path)
type HTMLConnector struct {
	path string
}

func (c *HTMLConnector) SourceType() string { return "html_files" }

func (c *HTMLConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("html_files: path is required")
	}
	return nil
}

func (c *HTMLConnector) Close() error { return nil }

func (c *HTMLConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
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

		text, err := extractHTMLText(io.LimitReader(f, 10*1024*1024))
		if err != nil {
			errc <- err
			return
		}

		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			select {
			case out <- connectors.FieldRecord{
				Value:        line,
				FieldName:    "text",
				SourcePath:   c.path,
				IsStructured: false,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, errc
}

func extractHTMLText(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString(text)
				buf.WriteString("\n")
			}
		}
		// Skip script and style content
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			extract(ch)
		}
	}
	extract(doc)
	return buf.String(), nil
}
