//go:build connector_stub

package cloud

import (
	"context"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// GDriveConnector scans Google Drive files for PII.
// Stub: requires OAuth2 service account credentials and google.golang.org/api/drive/v3.
type GDriveConnector struct{}

func (c *GDriveConnector) SourceType() string { return "gdrive" }

func (c *GDriveConnector) Connect(_ context.Context, _ map[string]any) error {
	return fmt.Errorf("gdrive connector: requires OAuth2 service account; see docs")
}

func (c *GDriveConnector) Close() error { return nil }

func (c *GDriveConnector) StreamFields(_ context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord)
	errc := make(chan error, 1)
	close(out)
	errc <- fmt.Errorf("gdrive connector: not yet implemented")
	close(errc)
	return out, errc
}
