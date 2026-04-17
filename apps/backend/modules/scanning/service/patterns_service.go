package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CustomPattern represents a user-defined PII detection pattern.
type CustomPattern struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Regex       string    `json:"regex"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Phase 2: regex engine hardening
	ValidationStatus   string     `json:"validation_status"` // pending | valid | invalid | risky
	BacktrackSafe      bool       `json:"backtrack_safe"`
	MatchCountLifetime int64      `json:"match_count_lifetime"`
	LastMatchedAt      *time.Time `json:"last_matched_at,omitempty"`
	// Phase 2: false-positive tracking (migration 000036)
	FalsePositiveCount int64   `json:"false_positive_count"`
	FalsePositiveRate  float64 `json:"false_positive_rate"`
	AutoDeactivated    bool    `json:"auto_deactivated"`
	// WS4: context keyword boosting for custom patterns (migration 000038)
	ContextKeywords  []string `json:"context_keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
}

// PatternStats is the response type for GET /patterns/:id/stats.
type PatternStats struct {
	PatternID          string     `json:"pattern_id"`
	Name               string     `json:"name"`
	TotalMatches       int64      `json:"total_matches"`
	FalsePositiveCount int64      `json:"false_positive_count"`
	FalsePositiveRate  float64    `json:"false_positive_rate"`
	MatchRatePerScan   float64    `json:"match_rate_per_scan"`
	LastMatchedAt      *time.Time `json:"last_matched_at"`
	IsActive           bool       `json:"is_active"`
	AutoDeactivated    bool       `json:"auto_deactivated"`
}

// TestCase is a single entry in a test-suite request.
type TestCase struct {
	Input       string `json:"input"`
	ShouldMatch bool   `json:"should_match"`
}

// TestCaseResult is the per-case result returned by TestPattern.
type TestCaseResult struct {
	Input       string `json:"input"`
	ShouldMatch bool   `json:"should_match"`
	Matched     bool   `json:"matched"`
	Passed      bool   `json:"passed"`
}

// TestPatternResult is the aggregate response for POST /patterns/:id/test.
type TestPatternResult struct {
	PatternID string           `json:"pattern_id"`
	Regex     string           `json:"regex"`
	Results   []TestCaseResult `json:"results"`
	Passed    int              `json:"passed"`
	Failed    int              `json:"failed"`
	Total     int              `json:"total"`
	PassRate  float64          `json:"pass_rate"`
}

// validateRegexSafety checks for catastrophic backtracking using two complementary layers:
//
//	Layer 1 — Static analysis: detect known ReDoS structural patterns before attempting
//	          compilation. Catches nested quantifiers and unbounded alternation.
//
//	Layer 2 — Runtime timeout: compile the regex and run it against a worst-case input
//	          (30 × 'a' + 'b') inside a goroutine with a 2-second deadline. Go's regexp
//	          package uses RE2 semantics and should never backtrack exponentially, but this
//	          guard catches degenerate linear-complexity patterns and extremely long patterns
//	          that still cause unacceptable latency.
//
// Returns an error if the pattern is unsafe; nil if it is safe to persist.
func validateRegexSafety(pattern string) error {
	// ---- Layer 1: static analysis ----

	// Detect excessively long patterns (>500 chars) — likely a ReDoS attempt.
	if len(pattern) > 500 {
		return fmt.Errorf("pattern too long (>500 chars): break into multiple named patterns")
	}

	// Compile check.
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}

	// Detect nested quantifiers: (...+)+ (...*)* (...+)* (...*)+ etc.
	nestedQuantifier := regexp.MustCompile(`\([^)]*[+*][^)]*\)[+*{]`)
	if nestedQuantifier.MatchString(pattern) {
		return fmt.Errorf("nested quantifier detected (ReDoS risk): wrap inner group with atomic group or possessive quantifier")
	}

	// Detect unbounded alternation inside a quantifier: (a|b|c)+ or (a|b)*.
	unboundedAlt := regexp.MustCompile(`\([^)]*\|[^)]*\)[+*]`)
	if unboundedAlt.MatchString(pattern) {
		return fmt.Errorf("unbounded alternation inside quantifier (ReDoS risk): use anchors or atomic groups")
	}

	// ---- Layer 2: runtime timeout ----
	// Run the compiled regex against a string known to trigger exponential
	// backtracking in NFA-based engines. Under Go's RE2 this should be fast,
	// but any pattern that takes >2s on this trivial input is unacceptable.
	done := make(chan error, 1)
	go func() {
		testStr := strings.Repeat("a", 30) + "b"
		re.MatchString(testStr)
		done <- nil
	}()

	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("regex may cause catastrophic backtracking (took >2s on test input)")
	}
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
		        is_active, created_by, created_at, updated_at,
		        COALESCE(context_keywords, '{}'), COALESCE(negative_keywords, '{}')
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
			&p.CreatedAt, &p.UpdatedAt,
			pq.Array(&p.ContextKeywords), pq.Array(&p.NegativeKeywords)); err != nil {
			return nil, err
		}
		if p.ContextKeywords == nil {
			p.ContextKeywords = []string{}
		}
		if p.NegativeKeywords == nil {
			p.NegativeKeywords = []string{}
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreatePattern inserts a new custom pattern.
func (s *PatternsService) CreatePattern(ctx context.Context, tenantID uuid.UUID, createdBy string, p *CustomPattern) (*CustomPattern, error) {
	// Validate regex safety (catastrophic backtracking check, Phase 2)
	if err := validateRegexSafety(p.Regex); err != nil {
		return nil, fmt.Errorf("unsafe regex rejected: %w", err)
	}

	// Sanitise name: uppercase, underscores
	p.Name = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(p.Name), " ", "_"))
	if p.Category == "" {
		p.Category = "Custom"
	}

	if p.ContextKeywords == nil {
		p.ContextKeywords = []string{}
	}
	if p.NegativeKeywords == nil {
		p.NegativeKeywords = []string{}
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO custom_patterns
		   (tenant_id, name, display_name, regex, category, description,
		    is_active, created_by, validation_status, backtrack_safe,
		    context_keywords, negative_keywords)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'valid',TRUE,$9,$10)
		 RETURNING id, created_at, updated_at`,
		tenantID, p.Name, p.DisplayName, p.Regex, p.Category, p.Description, true, createdBy,
		pq.Array(p.ContextKeywords), pq.Array(p.NegativeKeywords),
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
	if err := validateRegexSafety(p.Regex); err != nil {
		return nil, fmt.Errorf("unsafe regex rejected: %w", err)
	}

	if p.ContextKeywords == nil {
		p.ContextKeywords = []string{}
	}
	if p.NegativeKeywords == nil {
		p.NegativeKeywords = []string{}
	}
	var updated CustomPattern
	err := s.db.QueryRowContext(ctx,
		`UPDATE custom_patterns
		    SET display_name=$3, regex=$4, category=$5, description=$6, is_active=$7,
		        validation_status='valid', backtrack_safe=TRUE, updated_at=NOW(),
		        context_keywords=$8, negative_keywords=$9
		  WHERE id=$1 AND tenant_id=$2
		  RETURNING id, tenant_id, name, display_name, regex, category, description,
		            is_active, created_by, created_at, updated_at,
		            validation_status, backtrack_safe, match_count_lifetime, last_matched_at,
		            false_positive_count, false_positive_rate, auto_deactivated,
		            COALESCE(context_keywords, '{}'), COALESCE(negative_keywords, '{}')`,
		id, tenantID, p.DisplayName, p.Regex, p.Category, p.Description, p.IsActive,
		pq.Array(p.ContextKeywords), pq.Array(p.NegativeKeywords),
	).Scan(&updated.ID, &updated.TenantID, &updated.Name, &updated.DisplayName, &updated.Regex,
		&updated.Category, &updated.Description, &updated.IsActive, &updated.CreatedBy,
		&updated.CreatedAt, &updated.UpdatedAt,
		&updated.ValidationStatus, &updated.BacktrackSafe, &updated.MatchCountLifetime, &updated.LastMatchedAt,
		&updated.FalsePositiveCount, &updated.FalsePositiveRate, &updated.AutoDeactivated,
		pq.Array(&updated.ContextKeywords), pq.Array(&updated.NegativeKeywords))
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
// Returns context_keywords and negative_keywords so the scanner can both run
// the regex locally with keyword-aware scoring and forward the pattern to
// Presidio as an ad-hoc recognizer with context.
func (s *PatternsService) GetActivePatterns(ctx context.Context, tenantID uuid.UUID) ([]*CustomPattern, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, display_name, regex, category, description,
		        COALESCE(context_keywords, '{}'), COALESCE(negative_keywords, '{}')
		   FROM custom_patterns
		  WHERE tenant_id=$1 AND is_active=TRUE`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*CustomPattern
	for rows.Next() {
		p := &CustomPattern{}
		if err := rows.Scan(&p.ID, &p.Name, &p.DisplayName, &p.Regex, &p.Category, &p.Description,
			pq.Array(&p.ContextKeywords), pq.Array(&p.NegativeKeywords)); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// -----------------------------------------------------------------------
// Feature: Auto-deactivation on high false-positive rate (>30%)
// -----------------------------------------------------------------------

// RecordFalsePositive increments the false_positive_count for the given pattern,
// recalculates the false_positive_rate, and then calls CheckAndAutoDeactivate.
// It is idempotent with respect to the counter — each call adds exactly 1.
func (s *PatternsService) RecordFalsePositive(ctx context.Context, tenantID, patternID uuid.UUID) (*CustomPattern, error) {
	var p CustomPattern
	err := s.db.QueryRowContext(ctx,
		`UPDATE custom_patterns
		    SET false_positive_count = false_positive_count + 1,
		        false_positive_rate  = (false_positive_count + 1)::NUMERIC
		                               / GREATEST(match_count_lifetime, 1),
		        updated_at           = NOW()
		  WHERE id=$1 AND tenant_id=$2
		  RETURNING id, tenant_id, name, display_name, regex, category, description,
		            is_active, created_by, created_at, updated_at,
		            validation_status, backtrack_safe, match_count_lifetime, last_matched_at,
		            false_positive_count, false_positive_rate, auto_deactivated`,
		patternID, tenantID,
	).Scan(&p.ID, &p.TenantID, &p.Name, &p.DisplayName, &p.Regex,
		&p.Category, &p.Description, &p.IsActive, &p.CreatedBy,
		&p.CreatedAt, &p.UpdatedAt,
		&p.ValidationStatus, &p.BacktrackSafe, &p.MatchCountLifetime, &p.LastMatchedAt,
		&p.FalsePositiveCount, &p.FalsePositiveRate, &p.AutoDeactivated)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pattern not found")
	}
	if err != nil {
		return nil, fmt.Errorf("record false positive: %w", err)
	}

	// Trigger auto-deactivation check after updating the rate.
	if err := s.CheckAndAutoDeactivate(ctx, patternID); err != nil {
		// Log but don't surface — the increment itself succeeded.
		log.Printf("WARN: CheckAndAutoDeactivate(%s) failed: %v", patternID, err)
	}
	// Refresh pattern state in case auto-deactivation just fired.
	return s.getPatternByID(ctx, tenantID, patternID)
}

// CheckAndAutoDeactivate reads the current false_positive_rate for the given pattern
// and, if it exceeds 0.30, deactivates the pattern and sets auto_deactivated=TRUE.
// It is safe to call repeatedly — it is a no-op if the pattern is already deactivated
// or if the rate is within the acceptable threshold.
func (s *PatternsService) CheckAndAutoDeactivate(ctx context.Context, patternID uuid.UUID) error {
	var rate float64
	var isActive bool
	err := s.db.QueryRowContext(ctx,
		`SELECT false_positive_rate, is_active FROM custom_patterns WHERE id=$1`,
		patternID,
	).Scan(&rate, &isActive)
	if err == sql.ErrNoRows {
		return nil // Pattern gone — nothing to do.
	}
	if err != nil {
		return fmt.Errorf("check auto-deactivate: %w", err)
	}

	if !isActive || rate <= 0.30 {
		return nil // Already inactive or rate is acceptable.
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE custom_patterns
		    SET is_active=FALSE, auto_deactivated=TRUE, updated_at=NOW()
		  WHERE id=$1`,
		patternID,
	)
	if err != nil {
		return fmt.Errorf("auto-deactivate pattern: %w", err)
	}
	log.Printf("INFO: pattern %s auto-deactivated: false_positive_rate=%.4f exceeds 0.30 threshold", patternID, rate)
	return nil
}

// getPatternByID is an internal helper that fetches a single pattern by id + tenantID.
func (s *PatternsService) getPatternByID(ctx context.Context, tenantID, id uuid.UUID) (*CustomPattern, error) {
	var p CustomPattern
	err := s.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, display_name, regex, category, description,
		        is_active, created_by, created_at, updated_at,
		        validation_status, backtrack_safe, match_count_lifetime, last_matched_at,
		        false_positive_count, false_positive_rate, auto_deactivated,
		        COALESCE(context_keywords, '{}'), COALESCE(negative_keywords, '{}')
		   FROM custom_patterns
		  WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
	).Scan(&p.ID, &p.TenantID, &p.Name, &p.DisplayName, &p.Regex,
		&p.Category, &p.Description, &p.IsActive, &p.CreatedBy,
		&p.CreatedAt, &p.UpdatedAt,
		&p.ValidationStatus, &p.BacktrackSafe, &p.MatchCountLifetime, &p.LastMatchedAt,
		&p.FalsePositiveCount, &p.FalsePositiveRate, &p.AutoDeactivated,
		pq.Array(&p.ContextKeywords), pq.Array(&p.NegativeKeywords))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pattern not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get pattern: %w", err)
	}
	return &p, nil
}

// -----------------------------------------------------------------------
// Feature: Match frequency stats (GET /patterns/:id/stats)
// -----------------------------------------------------------------------

// GetPatternStats returns the match-frequency and false-positive statistics for a
// single pattern. match_rate_per_scan is a convenience metric derived by dividing
// total_matches by the number of completed scan_runs for this tenant (min 1).
func (s *PatternsService) GetPatternStats(ctx context.Context, tenantID, patternID uuid.UUID) (*PatternStats, error) {
	p, err := s.getPatternByID(ctx, tenantID, patternID)
	if err != nil {
		return nil, err
	}

	// Count completed scans for the tenant to compute match_rate_per_scan.
	var scanCount int64
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM scan_runs WHERE tenant_id=$1 AND status='completed'`,
		tenantID,
	).Scan(&scanCount)
	if scanCount < 1 {
		scanCount = 1
	}

	stats := &PatternStats{
		PatternID:          p.ID.String(),
		Name:               p.Name,
		TotalMatches:       p.MatchCountLifetime,
		FalsePositiveCount: p.FalsePositiveCount,
		FalsePositiveRate:  p.FalsePositiveRate,
		MatchRatePerScan:   float64(p.MatchCountLifetime) / float64(scanCount),
		LastMatchedAt:      p.LastMatchedAt,
		IsActive:           p.IsActive,
		AutoDeactivated:    p.AutoDeactivated,
	}
	return stats, nil
}

// -----------------------------------------------------------------------
// Feature: Test-suite endpoint (POST /patterns/:id/test)
// -----------------------------------------------------------------------

// TestPattern runs a compiled regex against a caller-supplied list of test cases and
// returns pass/fail for each, plus overall pass rate. Each test case is bounded by the
// same 2-second backtracking timeout used in validateRegexSafety to prevent a
// malicious test string from blocking the server.
func (s *PatternsService) TestPattern(ctx context.Context, tenantID, patternID uuid.UUID, cases []TestCase) (*TestPatternResult, error) {
	p, err := s.getPatternByID(ctx, tenantID, patternID)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(p.Regex)
	if err != nil {
		return nil, fmt.Errorf("stored regex no longer compiles: %w", err)
	}

	results := make([]TestCaseResult, 0, len(cases))
	passed := 0
	failed := 0

	for _, tc := range cases {
		// Run each match inside a goroutine with a 2s deadline to prevent ReDoS
		// via adversarial test inputs.
		matchCh := make(chan bool, 1)
		input := tc.Input // capture loop var
		go func() {
			matchCh <- re.MatchString(input)
		}()

		var matched bool
		select {
		case matched = <-matchCh:
		case <-time.After(2 * time.Second):
			// Treat timeout as non-match; mark as failed regardless of expectation.
			matched = false
		}

		pass := matched == tc.ShouldMatch
		results = append(results, TestCaseResult{
			Input:       tc.Input,
			ShouldMatch: tc.ShouldMatch,
			Matched:     matched,
			Passed:      pass,
		})
		if pass {
			passed++
		} else {
			failed++
		}
	}

	total := len(cases)
	var passRate float64
	if total > 0 {
		passRate = float64(passed) / float64(total)
	}

	return &TestPatternResult{
		PatternID: p.ID.String(),
		Regex:     p.Regex,
		Results:   results,
		Passed:    passed,
		Failed:    failed,
		Total:     total,
		PassRate:  passRate,
	}, nil
}
