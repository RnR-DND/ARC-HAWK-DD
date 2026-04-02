package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys to avoid collisions (H-1)
type contextKey string

// TenantIDKey is the context key for tenant ID
const TenantIDKey contextKey = "tenant_id"

// DevSystemTenantID is a dedicated UUID for development/test environments.
// This replaces uuid.Nil which must never be accepted as a valid tenant (C-1).
var DevSystemTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

var ErrTenantIDMissing = errors.New("tenant_id missing from context")

// GetTenantID extracts the tenant UUID from the context
func GetTenantID(ctx context.Context) (uuid.UUID, error) {
	// Try typed key first (preferred)
	tenantIDVal := ctx.Value(TenantIDKey)
	if tenantIDVal == nil {
		// Fallback: check bare string key for backward compatibility
		tenantIDVal = ctx.Value("tenant_id")
	}
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
	if id == uuid.Nil {
		return uuid.Nil, errors.New("security violation: nil tenant_id not permitted")
	}
	return id, nil
}
