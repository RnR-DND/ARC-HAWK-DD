package audit

import (
	"context"
	"encoding/json"
	"log"
	"time"

	entity "github.com/arc-platform/backend/modules/auth/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
)

// PostgresAuditLogger implements AuditLogger using PostgresRepository
type PostgresAuditLogger struct {
	repo *persistence.PostgresRepository
}

// NewPostgresAuditLogger creates a new audit logger
func NewPostgresAuditLogger(repo *persistence.PostgresRepository) interfaces.AuditLogger {
	return &PostgresAuditLogger{
		repo: repo,
	}
}

// Record records an audit log entry
func (l *PostgresAuditLogger) Record(ctx context.Context, action, resourceType, resourceID string, metadata map[string]interface{}) error {
	// Extract user context from context if available
	var userID, tenantID uuid.UUID

	if uid, ok := ctx.Value("user_id").(string); ok && uid != "" {
		if id, err := uuid.Parse(uid); err == nil {
			userID = id
		}
	}
	// Also support parsing from uuid type directly
	if uid, ok := ctx.Value("user_id").(uuid.UUID); ok {
		userID = uid
	}

	if tid, ok := ctx.Value("tenant_id").(string); ok && tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			tenantID = id
		}
	}
	if tid, ok := ctx.Value("tenant_id").(uuid.UUID); ok {
		tenantID = tid
	}

	// Marshal metadata
	metadataJSON, _ := json.Marshal(metadata)

	auditLog := &entity.AuditLog{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     string(metadataJSON),
		CreatedAt:    time.Now(),
		// IP and UserAgent could be extracted if passed in context, but typically handled by controller
	}

	// Fire and forget (don't block main flow), or synchronous?
	// Interface returns error, so synchronous is implied.
	// However, we shouldn't fail the operation if audit fails (usually), but strict compliance says otherwise.
	// For now, allow error return.
	if err := l.repo.CreateAuditLog(ctx, auditLog); err != nil {
		log.Printf("ERROR: Failed to record audit log: %v", err)
		return err
	}

	return nil
}
