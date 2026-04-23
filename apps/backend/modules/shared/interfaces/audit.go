package interfaces

import (
	"context"
)

// AuditLogger defines the contract for recording audit events
type AuditLogger interface {
	Record(ctx context.Context, action, resourceType, resourceID string, metadata map[string]interface{}) error
}

// NoOpAuditLogger is a no-op implementation for when audit logging is not configured.
type NoOpAuditLogger struct{}

func (NoOpAuditLogger) Record(_ context.Context, _, _, _ string, _ map[string]interface{}) error {
	return nil
}
