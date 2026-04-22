//go:build connector_stub

package databases

import (
	"context"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// OracleConnector is a stub. Oracle requires the Oracle Instant Client and the
// godror driver (CGO), which are not bundled in the standard scanner image.
// To enable: install Oracle Instant Client, add github.com/godror/godror, and
// implement Connect/StreamFields using database/sql with the "godror" driver.
type OracleConnector struct{}

func (c *OracleConnector) SourceType() string { return "oracle" }

func (c *OracleConnector) Connect(_ context.Context, _ map[string]any) error {
	return fmt.Errorf("oracle connector requires Oracle Instant Client; see docs for setup")
}

func (c *OracleConnector) Close() error { return nil }

func (c *OracleConnector) StreamFields(_ context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord)
	errc := make(chan error, 1)
	close(out)
	errc <- fmt.Errorf("oracle connector not available in this build")
	close(errc)
	return out, errc
}
