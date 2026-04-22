//go:build connector_stub

package cloud

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("gdrive", func() connectors.Connector { return &GDriveConnector{} })
	connectors.Register("gdrive_workspace", func() connectors.Connector { return &GDriveWorkspaceConnector{} })
	connectors.Register("azure_blob", func() connectors.Connector { return &AzureBlobConnector{} })
}
