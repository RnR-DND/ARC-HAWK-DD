//go:build connector_stub

package files

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("scanned_images", func() connectors.Connector { return &ImagesConnector{} })
}
