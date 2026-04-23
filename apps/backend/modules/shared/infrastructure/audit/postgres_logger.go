package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	entity "github.com/arc-platform/backend/modules/auth/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/arc-platform/backend/modules/shared/middleware"
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

	// Compute chain hash: query the most recent entry for this tenant to get its entry_hash.
	previousHash := "genesis"
	if tenantID != (uuid.UUID{}) {
		var prevHash sql.NullString
		// Use the underlying DB directly via the repo's embedded db — fall back to genesis on any error.
		if db := l.repo.GetDB(); db != nil {
			_ = db.QueryRowContext(ctx,
				`SELECT entry_hash FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1`,
				tenantID,
			).Scan(&prevHash)
			if prevHash.Valid && prevHash.String != "" {
				previousHash = prevHash.String
			}
		}
	}

	// entry_hash = SHA256(previous_hash || action || resource_type || resource_id || metadata || created_at ISO8601)
	raw := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		previousHash,
		auditLog.Action,
		auditLog.ResourceType,
		auditLog.ResourceID,
		string(metadataJSON),
		auditLog.CreatedAt.UTC().Format(time.RFC3339),
	)
	sum := sha256.Sum256([]byte(raw))
	entryHash := hex.EncodeToString(sum[:])

	auditLog.PreviousHash = previousHash
	auditLog.EntryHash = entryHash

	// Extract IP and UserAgent from context — not included in hash chain to avoid chain break
	if ip, ok := ctx.Value(middleware.ContextKeyIPAddress).(string); ok && ip != "" {
		auditLog.IPAddress = ip
	}
	if ua, ok := ctx.Value(middleware.ContextKeyUserAgent).(string); ok && ua != "" {
		auditLog.UserAgent = ua
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
