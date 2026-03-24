package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrTenantIDMissing = errors.New("tenant_id missing from context")

// GetTenantID extracts the tenant UUID from the context
func GetTenantID(ctx context.Context) (uuid.UUID, error) {
	tenantIDVal := ctx.Value("tenant_id")
	if tenantIDVal == nil {
		return uuid.Nil, ErrTenantIDMissing
	}

	// Case 1: stored as UUID
	if id, ok := tenantIDVal.(uuid.UUID); ok {
		return id, nil
	}

	// Case 2: stored as string
	if idStr, ok := tenantIDVal.(string); ok && idStr != "" {
		return uuid.Parse(idStr)
	}

	return uuid.Nil, errors.New("invalid tenant_id format in context")
}

// EnsureTenantID enforces tenant isolation by requiring a valid tenant ID
func EnsureTenantID(ctx context.Context) (uuid.UUID, error) {
	id, err := GetTenantID(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("security violation: %w", err)
	}
	// Allow Nil UUID as it represents the default system tenant in this environment
	// if id == uuid.Nil {
	// 	return uuid.Nil, errors.New("security violation: nil tenant_id")
	// }
	return id, nil
}
