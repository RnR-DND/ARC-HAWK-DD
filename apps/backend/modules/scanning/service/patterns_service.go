package service

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// CustomPattern represents a user-defined PII detection pattern.
type CustomPattern struct {
	ID                  uuid.UUID  `json:"id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	Name                string     `json:"name"`
	DisplayName         string     `json:"display_name"`
	Regex               string     `json:"regex"`
	Category            string     `json:"category"`
	Description         string     `json:"description"`
	IsActive            bool       `json:"is_active"`
	CreatedBy           string     `json:"created_by"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	// Phase 2: regex engine hardening
	ValidationStatus    string     `json:"validation_status"`    // pending | valid | invalid | risky
	BacktrackSafe       bool       `json:"backtrack_safe"`
	MatchCountLifetime  int64      `json:"match_count_lifetime"`
	LastMatchedAt       *time.Time `json:"last_matched_at,omitempty"`
}

// validateRegexSafety checks for common catastrophic backtracking patterns.
// These are not exhaustive but catch the most common ReDoS attack vectors:
//   - Nested quantifiers: (a+)+ or (a*)* or (a|aa)+
//   - Alternation inside unbounded quantifier: (a|b)*c
//
// Returns (isSafe bool, riskReason string)
func validateRegexSafety(pattern string) (bool, string) {
	// Compile check first
	if _, err := regexp.Compile(pattern); err != nil {
		return false, "compilation failed: " + err.Error()
	}

	// Detect nested quantifiers: a quantifier immediately containing another quantifier
	// Patterns: (...+)+ (...*)* (...+)* (...*)+ (...{n,})+ etc.
	nestedQuantifier := regexp.MustCompile(`\([^)]*[+*][^)]*\)[+*{]`)
	if nestedQuantifier.MatchString(pattern) {
		return false, "nested quantifier detected (ReDoS risk): wrap inner group with atomic group or possessive quantifier"
	}

	// Detect unbounded alternation: (a|b|c)+ or (a|b)* at top level
	unboundedAlt := regexp.MustCompile(`\([^)]*\|[^)]*\)[+*]`)
	if unboundedAlt.MatchString(pattern) {
		return false, "unbounded alternation inside quantifier (ReDoS risk): use anchors or atomic groups"
	}

	// Detect excessively long patterns (>500 chars) — likely a ReDoS attempt
	if len(pattern) > 500 {
		return false, "pattern too long (>500 chars): break into multiple named patterns"
	}

	return true, ""
}

// PatternsService handles CRUD for user-defined PII patterns.
type PatternsService struct {
	db *sql.DB
}

// NewPatternsService creates a PatternsService.
func NewPatternsService(repo *persistence.PostgresRepository) *PatternsService {
	return &PatternsService{db: repo.GetDB()}
}

// ListPatterns returns all active patterns for the given tenant.
func (s *PatternsService) ListPatterns(ctx context.Context, tenantID uuid.UUID) ([]*CustomPattern, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, tenant_id, name, display_name, regex, category, description,
		        is_active, created_by, created_at, updated_at
		   FROM custom_patterns
		  WHERE tenant_id = $1
		  ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list patterns: %w", err)
	}
	defer rows.Close()

	var out []*CustomPattern
	for rows.Next() {
		p := &CustomPattern{}
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.DisplayName, &p.Regex,
			&p.Category, &p.Description, &p.IsActive, &p.CreatedBy,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreatePattern inserts a new custom pattern.
func (s *PatternsService) CreatePattern(ctx context.Context, tenantID uuid.UUID, createdBy string, p *CustomPattern) (*CustomPattern, error) {
	// Validate regex safety (catastrophic backtracking check, Phase 2)
	safe, riskReason := validateRegexSafety(p.Regex)
	if !safe {
		return nil, fmt.Errorf("unsafe regex rejected: %s", riskReason)
	}

	// Sanitise name: uppercase, underscores
	p.Name = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(p.Name), " ", "_"))
	if p.Category == "" {
		p.Category = "Custom"
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO custom_patterns
		   (tenant_id, name, display_name, regex, category, description,
		    is_active, created_by, validation_status, backtrack_safe)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'valid',TRUE)
		 RETURNING id, created_at, updated_at`,
		tenantID, p.Name, p.DisplayName, p.Regex, p.Category, p.Description, true, createdBy,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create pattern: %w", err)
	}
	p.TenantID = tenantID
	p.IsActive = true
	p.CreatedBy = createdBy
	p.ValidationStatus = "valid"
	p.BacktrackSafe = true
	return p, nil
}

// UpdatePattern updates an existing pattern (regex, displayName, category, description, isActive).
func (s *PatternsService) UpdatePattern(ctx context.Context, tenantID, id uuid.UUID, p *CustomPattern) (*CustomPattern, error) {
	safe, riskReason := validateRegexSafety(p.Regex)
	if !safe {
		return nil, fmt.Errorf("unsafe regex rejected: %s", riskReason)
	}

	var updated CustomPattern
	err := s.db.QueryRowContext(ctx,
		`UPDATE custom_patterns
		    SET display_name=$3, regex=$4, category=$5, description=$6, is_active=$7,
		        validation_status='valid', backtrack_safe=TRUE, updated_at=NOW()
		  WHERE id=$1 AND tenant_id=$2
		  RETURNING id, tenant_id, name, display_name, regex, category, description,
		            is_active, created_by, created_at, updated_at,
		            validation_status, backtrack_safe, match_count_lifetime, last_matched_at`,
		id, tenantID, p.DisplayName, p.Regex, p.Category, p.Description, p.IsActive,
	).Scan(&updated.ID, &updated.TenantID, &updated.Name, &updated.DisplayName, &updated.Regex,
		&updated.Category, &updated.Description, &updated.IsActive, &updated.CreatedBy,
		&updated.CreatedAt, &updated.UpdatedAt,
		&updated.ValidationStatus, &updated.BacktrackSafe, &updated.MatchCountLifetime, &updated.LastMatchedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pattern not found")
	}
	if err != nil {
		return nil, fmt.Errorf("update pattern: %w", err)
	}
	return &updated, nil
}

// DeletePattern removes a pattern owned by tenantID.
func (s *PatternsService) DeletePattern(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM custom_patterns WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete pattern: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("pattern not found")
	}
	return nil
}

// GetActivePatterns returns active patterns for a tenant (used by scan trigger).
func (s *PatternsService) GetActivePatterns(ctx context.Context, tenantID uuid.UUID) ([]*CustomPattern, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, display_name, regex, category, description
		   FROM custom_patterns
		  WHERE tenant_id=$1 AND is_active=TRUE`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*CustomPattern
	for rows.Next() {
		p := &CustomPattern{}
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.Regex, &p.Category, &p.Description); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
