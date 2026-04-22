package cloud

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("s3", func() connectors.Connector { return &S3Connector{} })
	connectors.Register("gcs", func() connectors.Connector { return &GCSConnector{} })
}
