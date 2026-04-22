//go:build connector_stub

package cloud

import (
	"context"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// GDriveWorkspaceConnector scans Google Workspace (Docs, Sheets, Slides) via Drive API.
// Stub: requires domain-wide delegation and google.golang.org/api/drive/v3.
type GDriveWorkspaceConnector struct{}

func (c *GDriveWorkspaceConnector) SourceType() string { return "gdrive_workspace" }

func (c *GDriveWorkspaceConnector) Connect(_ context.Context, _ map[string]any) error {
	return fmt.Errorf("gdrive_workspace connector: requires Google Workspace domain delegation; see docs")
}

func (c *GDriveWorkspaceConnector) Close() error { return nil }

func (c *GDriveWorkspaceConnector) StreamFields(_ context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord)
	errc := make(chan error, 1)
	close(out)
	errc <- fmt.Errorf("gdrive_workspace connector: not yet implemented")
	close(errc)
	return out, errc
}
