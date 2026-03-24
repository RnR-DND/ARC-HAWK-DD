package connectors

import (
	"context"
	"fmt"
)

// SourceConnector interface for remediation on different data sources
type SourceConnector interface {
	// Connect establishes connection to the source system
	Connect(ctx context.Context, config map[string]interface{}) error

	// Close closes the connection
	Close() error

	// Mask redacts PII in place
	Mask(ctx context.Context, location string, fieldName string, recordID string) error

	// Delete removes the entire record
	Delete(ctx context.Context, location string, recordID string) error

	// Encrypt encrypts PII value
	Encrypt(ctx context.Context, location string, fieldName string, recordID string, encryptionKey string) error

	// GetOriginalValue retrieves original value before remediation (for rollback)
	GetOriginalValue(ctx context.Context, location string, fieldName string, recordID string) (string, error)

	// RestoreValue restores original value (rollback)
	RestoreValue(ctx context.Context, location string, fieldName string, recordID string, originalValue string) error
}

// ConnectorFactory creates appropriate connector based on source type
type ConnectorFactory struct{}

// NewConnector creates a new connector for the given source type
func (f *ConnectorFactory) NewConnector(sourceType string) (SourceConnector, error) {
	switch sourceType {
	case "postgresql":
		return &PostgreSQLConnector{}, nil
	case "mysql":
		return &MySQLConnector{}, nil
	case "s3":
		return &S3Connector{}, nil
	case "mongodb":
		return &MongoDBConnector{}, nil
	case "filesystem":
		return &FilesystemConnector{}, nil
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}
