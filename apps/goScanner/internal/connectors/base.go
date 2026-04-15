package connectors

import "context"

// Connector is the interface every data source connector must implement.
type Connector interface {
	Connect(ctx context.Context, config map[string]any) error
	Close() error
	StreamFields(ctx context.Context) (<-chan FieldRecord, <-chan error)
	SourceType() string
}

// FieldRecord is one scannable field value from any data source.
type FieldRecord struct {
	Value        string
	FieldName    string
	SourcePath   string // "schema.table.column" or "/dir/file.csv:colname"
	RowContext    string // ±100 chars around value
	Metadata     map[string]any
	IsStructured bool
}

// ConnectorFactory creates a new Connector instance.
type ConnectorFactory func() Connector
