package cloud

import (
	"context"
	"fmt"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// AzureBlobConnector scans Azure Blob Storage containers.
// Stub: requires github.com/Azure/azure-sdk-for-go/sdk/storage/azblob.
type AzureBlobConnector struct{}

func (c *AzureBlobConnector) SourceType() string { return "azure_blob" }

func (c *AzureBlobConnector) Connect(_ context.Context, _ map[string]any) error {
	return fmt.Errorf("azure_blob connector: not yet implemented — use connection_string from Azure portal")
}

func (c *AzureBlobConnector) Close() error { return nil }

func (c *AzureBlobConnector) StreamFields(_ context.Context) (<-chan connectors.FieldRecord, <-chan error) {
	out := make(chan connectors.FieldRecord)
	errc := make(chan error, 1)
	close(out)
	errc <- fmt.Errorf("azure_blob connector: not yet implemented")
	close(errc)
	return out, errc
}
