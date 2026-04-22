//go:build connector_stub

package databases

import "github.com/arc-platform/go-scanner/internal/connectors"

func init() {
	connectors.Register("oracle", func() connectors.Connector { return &OracleConnector{} })
}
