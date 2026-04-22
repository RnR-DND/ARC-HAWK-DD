package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType for audit_ledger entries (DPDP Act 2023 compliance events)
type EventType string

const (
	EventScanCompleted        EventType = "scan_completed"
	EventPIIDiscovered        EventType = "pii_discovered"
	EventConsentGranted       EventType = "consent_granted"
	EventConsentRevoked       EventType = "consent_revoked"
	EventDPRSubmitted         EventType = "dpr_submitted"
	EventDPRResolved          EventType = "dpr_resolved"
	EventRemediationApplied   EventType = "remediation_applied"
	EventPolicyEvaluated      EventType = "policy_evaluated"
	EventCrossBorderTransfer  EventType = "cross_border_transfer"
	EventGROEscalation        EventType = "gro_escalation"
	EventEvidencePackageGen   EventType = "evidence_package_generated"
)

// LedgerLogger writes append-only compliance events to audit_ledger.
// Distinct from PostgresAuditLogger (which writes to audit_logs for general activity).
type LedgerLogger struct {
	db *sql.DB
}

// NewLedgerLogger creates a LedgerLogger backed by the given database connection.
func NewLedgerLogger(db *sql.DB) *LedgerLogger {
	return &LedgerLogger{db: db}
}

// LogEntry describes a single compliance audit event.
type LogEntry struct {
	TenantID     uuid.UUID
	EventType    EventType
	ActorID      *uuid.UUID
	ActorEmail   string
	ResourceID   string
	ResourceType string
	Payload      map[string]interface{}
	IPAddress    string
}

// Log inserts one immutable row into audit_ledger.
func (l *LedgerLogger) Log(ctx context.Context, entry LogEntry) error {
	payload, err := json.Marshal(entry.Payload)
	if err != nil {
		payload = []byte("{}")
	}

	var ipAddr interface{}
	if entry.IPAddress != "" {
		ipAddr = entry.IPAddress
	}

	_, err = l.db.ExecContext(ctx, `
		INSERT INTO audit_ledger
			(tenant_id, event_type, actor_id, actor_email, resource_id, resource_type, payload, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::inet, $9)`,
		entry.TenantID, string(entry.EventType),
		entry.ActorID, entry.ActorEmail,
		entry.ResourceID, entry.ResourceType,
		payload, ipAddr, time.Now().UTC(),
	)
	return err
}

// Query returns up to limit ledger rows for a tenant in the given time range.
// eventTypes is optional; pass nil to return all event types.
func (l *LedgerLogger) Query(ctx context.Context, tenantID uuid.UUID, eventTypes []string, from, to time.Time, limit int) ([]map[string]interface{}, error) {
	rows, err := l.db.QueryContext(ctx, `
		SELECT id, event_type, actor_id, actor_email, resource_id, resource_type, payload, ip_address, created_at
		FROM audit_ledger
		WHERE tenant_id = $1
		  AND ($2::text[] IS NULL OR event_type = ANY($2::text[]))
		  AND created_at BETWEEN $3 AND $4
		ORDER BY created_at DESC
		LIMIT $5`,
		tenantID, eventTypes, from, to, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var (
			id, eventType, actorEmail, resourceID, resourceType string
			actorID                                             *string
			ipAddr                                              *string
			payloadB                                            []byte
			createdAt                                           time.Time
		)
		if err := rows.Scan(&id, &eventType, &actorID, &actorEmail, &resourceID, &resourceType, &payloadB, &ipAddr, &createdAt); err != nil {
			continue
		}
		var payload map[string]interface{}
		_ = json.Unmarshal(payloadB, &payload)

		row := map[string]interface{}{
			"id":            id,
			"event_type":    eventType,
			"actor_email":   actorEmail,
			"resource_id":   resourceID,
			"resource_type": resourceType,
			"payload":       payload,
			"created_at":    createdAt.Format(time.RFC3339),
		}
		if actorID != nil {
			row["actor_id"] = *actorID
		}
		if ipAddr != nil {
			row["ip_address"] = *ipAddr
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
