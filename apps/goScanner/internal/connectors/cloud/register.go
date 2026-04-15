package cloud

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("s3", func() connectors.Connector { return &S3Connector{} })
	connectors.Register("gcs", func() connectors.Connector { return &GCSConnector{} })
	connectors.Register("azure_blob", func() connectors.Connector { return &AzureBlobConnector{} })
	connectors.Register("gdrive", func() connectors.Connector { return &GDriveConnector{} })
	connectors.Register("gdrive_workspace", func() connectors.Connector { return &GDriveWorkspaceConnector{} })
}
