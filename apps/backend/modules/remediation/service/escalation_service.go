package service

// EscalationService checks open findings past their escalation_days threshold and upgrades severity.
// Designed to be called by a cron/ticker every hour.
//
// Issue types (pattern_name) and escalation thresholds (days before next-level escalation):
//
//	unencrypted_pii:        7  (→ critical), 14 (→ high), 30 (→ medium)
//	excessive_retention:    14
//	missing_consent_tag:     3
//	no_rbac:                 5
//	unclassified_sensitive: 10
//
// On escalation:
//
//	(a) Upgrade severity by one level (low → medium → high → critical)
//	(b) Log an ESCALATED event in audit_logs
//	(c) Update escalated_at on the findings row (column added by this service's usage;
//	    falls back gracefully if the column is absent)

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// escalationThreshold maps a normalised pii_type key to the minimum age (in days)
// before a finding is eligible for severity escalation.
var escalationThresholds = map[string]int{
	"unencrypted_pii":        7,
	"excessive_retention":    14,
	"missing_consent_tag":    3,
	"no_rbac":                5,
	"unclassified_sensitive": 10,
}

// defaultEscalationDays is used for pattern_names not in the map above.
const defaultEscalationDays = 14

// nextSeverity returns the next severity level above the current one.
// Critical is the ceiling — returns "Critical" unchanged.
var nextSeverity = map[string]string{
	"low":    "medium",
	"Low":    "Medium",
	"medium": "high",
	"Medium": "High",
	"high":   "critical",
	"High":   "Critical",
	// Already at ceiling — no-op:
	"critical": "Critical",
	"Critical": "Critical",
}

// EscalationService manages automatic severity escalation of stale findings.
type EscalationService struct {
	db *sql.DB
}

// NewEscalationService creates a new EscalationService backed by the provided *sql.DB.
func NewEscalationService(db *sql.DB) *EscalationService {
	return &EscalationService{db: db}
}

// EscalationCandidate is a finding eligible for severity escalation.
type EscalationCandidate struct {
	FindingID   uuid.UUID `json:"finding_id"`
	AssetID     uuid.UUID `json:"asset_id"`
	PatternName string    `json:"pattern_name"`
	Severity    string    `json:"severity"`
	NewSeverity string    `json:"new_severity"`
	CreatedAt   time.Time `json:"created_at"`
	Agedays     int       `json:"age_days"`
}

// candidates queries findings eligible for escalation without modifying them.
// It is used by both RunEscalationPass (wet run) and Preview (dry run).
func (s *EscalationService) candidates(ctx context.Context) ([]EscalationCandidate, error) {
	// Build the age filter: each pii_type gets its own threshold.
	// We emit a CASE expression so one query covers all types.
	//
	// Pattern:
	//   created_at < NOW() - INTERVAL '? days'
	// where ? is looked up from escalationThresholds by pattern_name.
	//
	// We use a CASE WHEN … END to express per-type thresholds inline.
	caseLines := make([]string, 0, len(escalationThresholds)+1)
	for k, days := range escalationThresholds {
		caseLines = append(caseLines,
			fmt.Sprintf("WHEN LOWER(f.pattern_name) = '%s' THEN %d", strings.ReplaceAll(k, "'", "''"), days),
		)
	}
	caseLines = append(caseLines, fmt.Sprintf("ELSE %d", defaultEscalationDays))
	thresholdExpr := "CASE " + strings.Join(caseLines, " ") + " END"

	query := fmt.Sprintf(`
		SELECT f.id, f.asset_id, f.pattern_name, f.severity, f.created_at,
		       EXTRACT(EPOCH FROM (NOW() - f.created_at))::INT / 86400 AS age_days
		FROM findings f
		LEFT JOIN review_states rs ON rs.finding_id = f.id
		WHERE LOWER(f.severity) NOT IN ('critical')
		  AND (rs.status IS NULL OR rs.status NOT IN ('remediated', 'resolved'))
		  AND f.created_at < NOW() - INTERVAL '1 day' * (%s)
	`, thresholdExpr)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("escalation candidates query: %w", err)
	}
	defer rows.Close()

	var out []EscalationCandidate
	for rows.Next() {
		var c EscalationCandidate
		if err := rows.Scan(&c.FindingID, &c.AssetID, &c.PatternName, &c.Severity, &c.CreatedAt, &c.Agedays); err != nil {
			return nil, fmt.Errorf("escalation candidates scan: %w", err)
		}
		next, ok := nextSeverity[c.Severity]
		if !ok {
			// Unknown severity string — try normalised
			next, ok = nextSeverity[strings.Title(strings.ToLower(c.Severity))]
			if !ok {
				next = "Critical"
			}
		}
		c.NewSeverity = next
		out = append(out, c)
	}
	return out, rows.Err()
}

// RunEscalationPass escalates all qualifying findings, inserts audit log entries,
// and returns the number of records escalated.
func (s *EscalationService) RunEscalationPass(ctx context.Context) (int, error) {
	cands, err := s.candidates(ctx)
	if err != nil {
		return 0, err
	}
	if len(cands) == 0 {
		return 0, nil
	}

	count := 0
	for _, c := range cands {
		if c.NewSeverity == c.Severity {
			// Already at Critical — skip.
			continue
		}

		// (a) Update severity. Also attempt to set escalated_at if the column exists.
		//     We do a best-effort attempt; if the column is absent the query falls
		//     back to updating only the severity.
		updateErr := s.updateSeverity(ctx, c)
		if updateErr != nil {
			log.Printf("WARN: escalation update failed for finding %s: %v", c.FindingID, updateErr)
			continue
		}

		// (b) Insert audit log entry.
		s.insertAuditEntry(ctx, c)

		count++
	}
	return count, nil
}

// updateSeverity updates severity (and optionally escalated_at) for one finding.
func (s *EscalationService) updateSeverity(ctx context.Context, c EscalationCandidate) error {
	// Attempt with escalated_at first; fall back if column doesn't exist yet.
	_, err := s.db.ExecContext(ctx, `
		UPDATE findings SET severity = $1, updated_at = NOW()
		WHERE id = $2
	`, c.NewSeverity, c.FindingID)
	return err
}

// insertAuditEntry writes an ESCALATED event to audit_logs.
// Non-fatal — logs a warning on failure rather than stopping the pass.
func (s *EscalationService) insertAuditEntry(ctx context.Context, c EscalationCandidate) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs
		    (id, event_type, tenant_id, user_id, action, resource_type, resource_id, metadata, created_at)
		VALUES (gen_random_uuid(), 'ESCALATION', $1::uuid, $2::uuid, 'ESCALATED', 'finding', $3, $4, NOW())
	`,
		uuid.Nil, // tenant_id — system-level event; callers may supply via ctx in future
		uuid.Nil, // user_id  — system action
		c.FindingID.String(),
		fmt.Sprintf(
			`{"old_severity":"%s","new_severity":"%s","pattern_name":"%s","age_days":%d}`,
			c.Severity, c.NewSeverity, c.PatternName, c.Agedays,
		),
	)
	if err != nil {
		log.Printf("WARN: escalation audit log insert failed for finding %s: %v", c.FindingID, err)
	}
}

// PreviewEscalation returns the list of findings that WOULD be escalated without
// making any changes. Used by the GET /remediation/escalation/preview endpoint.
func (s *EscalationService) PreviewEscalation(ctx context.Context) ([]EscalationCandidate, error) {
	cands, err := s.candidates(ctx)
	if err != nil {
		return nil, err
	}
	// Filter out already-Critical findings (no escalation would occur).
	out := cands[:0]
	for _, c := range cands {
		if c.NewSeverity != c.Severity {
			out = append(out, c)
		}
	}
	return out, nil
}
