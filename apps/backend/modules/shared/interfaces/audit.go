package interfaces

import (
	"context"
)

// AuditLogger defines the contract for recording audit events
type AuditLogger interface {
	Record(ctx context.Context, action, resourceType, resourceID string, metadata map[string]interface{}) error
}
