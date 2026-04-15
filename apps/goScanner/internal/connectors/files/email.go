package files

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// EmailConnector parses EML files (RFC 5322) and extracts headers + body text.
// Config keys: path (file path or directory of .eml files)
type EmailConnector struct {
	path string
}

func (c *EmailConnector) SourceType() string { return "email_files" }

func (c *EmailConnector) Connect(_ context.Context, cfg map[string]any) error {
	c.path = fmt.Sprintf("%v", cfg["path"])
	if c.path == "" || c.path == "<nil>" {
		return fmt.Errorf("email_files: path is required")
	}
	return nil
}

func (c *EmailConnector) Close() error { return nil }

func (c *EmailConnector) StreamFields(ctx context.Context) (<-chan connectors.FieldRecord, <-chan error) {
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
			entries, _ := os.ReadDir(c.path)
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				lower := strings.ToLower(e.Name())
				if strings.HasSuffix(lower, ".eml") || strings.HasSuffix(lower, ".msg") {
					c.streamEmail(ctx, c.path+"/"+e.Name(), out)
				}
			}
		} else {
			c.streamEmail(ctx, c.path, out)
		}
	}()
	return out, errc
}

func (c *EmailConnector) streamEmail(ctx context.Context, path string, out chan<- connectors.FieldRecord) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	msg, err := mail.ReadMessage(bufio.NewReader(f))
	if err != nil {
		return
	}

	// Emit important headers
	for _, hdr := range []string{"From", "To", "Cc", "Subject", "Reply-To"} {
		val := msg.Header.Get(hdr)
		if val == "" {
			continue
		}
		select {
		case out <- connectors.FieldRecord{
			Value:        val,
			FieldName:    strings.ToLower(hdr),
			SourcePath:   fmt.Sprintf("%s:header.%s", path, hdr),
			IsStructured: false,
		}:
		case <-ctx.Done():
			return
		}
	}

	// Parse body
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		// Fallback: read raw body
		body, _ := io.ReadAll(io.LimitReader(msg.Body, 5*1024*1024))
		if len(body) > 0 {
			out <- connectors.FieldRecord{
				Value: string(body), FieldName: "body",
				SourcePath: path + ":body", IsStructured: false,
			}
		}
		return
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(msg.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			ct := part.Header.Get("Content-Type")
			if strings.HasPrefix(ct, "text/") || ct == "" {
				data, _ := io.ReadAll(io.LimitReader(part, 5*1024*1024))
				if len(data) > 0 {
					select {
					case out <- connectors.FieldRecord{
						Value:        string(data),
						FieldName:    "body_part",
						SourcePath:   path + ":body",
						IsStructured: false,
					}:
					case <-ctx.Done():
						return
					}
				}
			}
			part.Close()
		}
	} else if strings.HasPrefix(mediaType, "text/") {
		data, _ := io.ReadAll(io.LimitReader(msg.Body, 5*1024*1024))
		if len(data) > 0 {
			select {
			case out <- connectors.FieldRecord{
				Value:        string(data),
				FieldName:    "body",
				SourcePath:   path + ":body",
				IsStructured: false,
			}:
			case <-ctx.Done():
			}
		}
	}
}
