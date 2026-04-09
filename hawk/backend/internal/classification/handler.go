package classification

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/arc/hawk/backend/internal/auth"
	"github.com/arc/hawk/backend/internal/shared"
)

// CustomRegex represents a custom regex pattern.
type CustomRegex struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Pattern         string    `json:"pattern"`
	Classification  string    `json:"classification"`
	Confidence      float64   `json:"confidence"`
	IsActive        bool      `json:"is_active"`
	IsDeleted       bool      `json:"is_deleted,omitempty"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateRegexRequest is the request body for creating a custom regex pattern.
type CreateRegexRequest struct {
	Name           string  `json:"name" binding:"required"`
	Description    string  `json:"description"`
	Pattern        string  `json:"pattern" binding:"required"`
	Classification string  `json:"classification" binding:"required"`
	Confidence     float64 `json:"confidence" binding:"required,min=0,max=1"`
}

// UpdateRegexRequest is the request body for updating a custom regex pattern.
type UpdateRegexRequest struct {
	Name           *string  `json:"name"`
	Description    *string  `json:"description"`
	Pattern        *string  `json:"pattern"`
	Classification *string  `json:"classification"`
	Confidence     *float64 `json:"confidence"`
	IsActive       *bool    `json:"is_active"`
}

// TestRegexRequest is the request body for testing a regex pattern.
type TestRegexRequest struct {
	SampleData []string `json:"sample_data" binding:"required"`
}

// TestRegexResult is a single test result for a sample input.
type TestRegexResult struct {
	Input   string   `json:"input"`
	Matched bool     `json:"matched"`
	Matches []string `json:"matches,omitempty"`
}

// TestRegexResponse contains the full test suite results.
type TestRegexResponse struct {
	PatternID    string            `json:"pattern_id"`
	Pattern      string            `json:"pattern"`
	TotalSamples int               `json:"total_samples"`
	MatchCount   int               `json:"match_count"`
	MatchRate    float64           `json:"match_rate"`
	Results      []TestRegexResult `json:"results"`
	ExecutionMs  int64             `json:"execution_ms"`
}

// RegexStats contains match frequency statistics.
type RegexStats struct {
	PatternID        string  `json:"pattern_id"`
	TotalScans       int     `json:"total_scans"`
	TotalMatches     int     `json:"total_matches"`
	TotalFields      int     `json:"total_fields_checked"`
	MatchRate        float64 `json:"match_rate"`
	FalsePositives   int     `json:"false_positives"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
	LastMatchedAt    *time.Time `json:"last_matched_at,omitempty"`
}

// RegisterRoutes registers classification-related HTTP handlers.
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/custom-regex", createRegex)
	rg.GET("/custom-regex", listRegex)
	rg.GET("/custom-regex/:id", getRegex)
	rg.PUT("/custom-regex/:id", updateRegex)
	rg.DELETE("/custom-regex/:id", deleteRegex)
	rg.POST("/custom-regex/:id/test", testRegex)
	rg.GET("/custom-regex/:id/stats", getRegexStats)
}

// createRegex creates a new custom regex pattern with validation.
func createRegex(c *gin.Context) {
	var req CreateRegexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.ErrBadRequest(c, "invalid request body", err.Error())
		return
	}

	user := auth.GetUser(c)
	if user == nil {
		shared.ErrUnauthorized(c, "authentication required")
		return
	}

	// Validate the regex compiles
	if err := validateRegexPattern(req.Pattern); err != nil {
		shared.ErrBadRequest(c, "invalid regex pattern", err.Error())
		return
	}

	// Check for catastrophic backtracking
	if isCatastrophicBacktracking(req.Pattern) {
		shared.ErrBadRequest(c, "rejected: pattern has catastrophic backtracking risk",
			"nested quantifiers detected, which can cause exponential matching time")
		return
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	db := shared.WriteDB()
	_, err := db.Exec(c.Request.Context(), `
		INSERT INTO custom_regex_patterns
			(id, name, description, pattern, classification, confidence,
			 is_active, is_deleted, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, false, $7, $8, $8)
	`, id, req.Name, req.Description, req.Pattern, req.Classification,
		req.Confidence, user.Subject, now)
	if err != nil {
		shared.ErrInternal(c, "failed to create regex pattern", err)
		return
	}

	c.JSON(http.StatusCreated, CustomRegex{
		ID:             id,
		Name:           req.Name,
		Description:    req.Description,
		Pattern:        req.Pattern,
		Classification: req.Classification,
		Confidence:     req.Confidence,
		IsActive:       true,
		CreatedBy:      user.Subject,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
}

// listRegex returns all active (non-deleted) custom regex patterns.
func listRegex(c *gin.Context) {
	pg := shared.ParsePagination(c)

	includeInactive := c.Query("include_inactive") == "true"
	condition := "WHERE is_deleted = false"
	if !includeInactive {
		condition += " AND is_active = true"
	}

	var total int64
	err := shared.ReadDB().QueryRow(c.Request.Context(),
		"SELECT COUNT(*) FROM custom_regex_patterns "+condition).Scan(&total)
	if err != nil {
		shared.ErrInternal(c, "failed to count patterns", err)
		return
	}

	rows, err := shared.ReadDB().Query(c.Request.Context(), fmt.Sprintf(`
		SELECT id, name, description, pattern, classification, confidence,
		       is_active, false_positive_rate, created_by, created_at, updated_at
		FROM custom_regex_patterns
		%s
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, condition), pg.PageSize, pg.Offset)
	if err != nil {
		shared.ErrInternal(c, "failed to list patterns", err)
		return
	}
	defer rows.Close()

	var patterns []CustomRegex
	for rows.Next() {
		var p CustomRegex
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Pattern, &p.Classification,
			&p.Confidence, &p.IsActive, &p.FalsePositiveRate,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			shared.ErrInternal(c, "failed to scan pattern row", err)
			return
		}
		patterns = append(patterns, p)
	}

	if patterns == nil {
		patterns = []CustomRegex{}
	}

	c.JSON(http.StatusOK, shared.PaginatedResponse{
		Data:       patterns,
		Pagination: pg,
		Total:      total,
	})
}

// getRegex returns a single custom regex pattern by ID.
func getRegex(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		shared.ErrBadRequest(c, "pattern ID required")
		return
	}

	var p CustomRegex
	err := shared.ReadDB().QueryRow(c.Request.Context(), `
		SELECT id, name, description, pattern, classification, confidence,
		       is_active, is_deleted, false_positive_rate, created_by, created_at, updated_at
		FROM custom_regex_patterns WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Pattern, &p.Classification,
		&p.Confidence, &p.IsActive, &p.IsDeleted, &p.FalsePositiveRate,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		shared.ErrNotFound(c, "pattern not found")
		return
	}

	c.JSON(http.StatusOK, p)
}

// updateRegex updates an existing custom regex pattern.
func updateRegex(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		shared.ErrBadRequest(c, "pattern ID required")
		return
	}

	var req UpdateRegexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.ErrBadRequest(c, "invalid request body", err.Error())
		return
	}

	// If pattern is being updated, validate it
	if req.Pattern != nil {
		if err := validateRegexPattern(*req.Pattern); err != nil {
			shared.ErrBadRequest(c, "invalid regex pattern", err.Error())
			return
		}
		if isCatastrophicBacktracking(*req.Pattern) {
			shared.ErrBadRequest(c, "rejected: pattern has catastrophic backtracking risk")
			return
		}
	}

	db := shared.WriteDB()
	now := time.Now().UTC()

	// Build dynamic update query
	setClauses := []string{"updated_at = $2"}
	args := []any{id, now}
	argIdx := 3

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Pattern != nil {
		setClauses = append(setClauses, fmt.Sprintf("pattern = $%d", argIdx))
		args = append(args, *req.Pattern)
		argIdx++
	}
	if req.Classification != nil {
		setClauses = append(setClauses, fmt.Sprintf("classification = $%d", argIdx))
		args = append(args, *req.Classification)
		argIdx++
	}
	if req.Confidence != nil {
		setClauses = append(setClauses, fmt.Sprintf("confidence = $%d", argIdx))
		args = append(args, *req.Confidence)
		argIdx++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	query := fmt.Sprintf(`
		UPDATE custom_regex_patterns
		SET %s
		WHERE id = $1 AND is_deleted = false
	`, strings.Join(setClauses, ", "))

	result, err := db.Exec(c.Request.Context(), query, args...)
	if err != nil {
		shared.ErrInternal(c, "failed to update pattern", err)
		return
	}

	if result.RowsAffected() == 0 {
		shared.ErrNotFound(c, "pattern not found or already deleted")
		return
	}

	// Fetch updated pattern
	getRegex(c)
}

// deleteRegex performs a soft delete on a custom regex pattern.
func deleteRegex(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		shared.ErrBadRequest(c, "pattern ID required")
		return
	}

	db := shared.WriteDB()
	now := time.Now().UTC()

	result, err := db.Exec(c.Request.Context(), `
		UPDATE custom_regex_patterns
		SET is_deleted = true, is_active = false, updated_at = $1
		WHERE id = $2 AND is_deleted = false
	`, now, id)
	if err != nil {
		shared.ErrInternal(c, "failed to delete pattern", err)
		return
	}

	if result.RowsAffected() == 0 {
		shared.ErrNotFound(c, "pattern not found or already deleted")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"deleted": true,
		"message": "pattern soft deleted",
	})
}

// testRegex runs a validation suite against sample data.
func testRegex(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		shared.ErrBadRequest(c, "pattern ID required")
		return
	}

	var req TestRegexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.ErrBadRequest(c, "invalid request body", err.Error())
		return
	}

	if len(req.SampleData) == 0 {
		shared.ErrBadRequest(c, "sample_data must not be empty")
		return
	}

	if len(req.SampleData) > 1000 {
		shared.ErrBadRequest(c, "sample_data exceeds maximum of 1000 entries")
		return
	}

	// Fetch the pattern
	var pattern string
	err := shared.ReadDB().QueryRow(c.Request.Context(), `
		SELECT pattern FROM custom_regex_patterns WHERE id = $1 AND is_deleted = false
	`, id).Scan(&pattern)
	if err != nil {
		shared.ErrNotFound(c, "pattern not found")
		return
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		shared.ErrInternal(c, "stored pattern failed to compile", err)
		return
	}

	start := time.Now()
	var results []TestRegexResult
	matchCount := 0

	for _, sample := range req.SampleData {
		matches := re.FindAllString(sample, -1)
		matched := len(matches) > 0
		if matched {
			matchCount++
		}
		results = append(results, TestRegexResult{
			Input:   sample,
			Matched: matched,
			Matches: matches,
		})
	}

	executionMs := time.Since(start).Milliseconds()
	matchRate := 0.0
	if len(req.SampleData) > 0 {
		matchRate = float64(matchCount) / float64(len(req.SampleData)) * 100
	}

	c.JSON(http.StatusOK, TestRegexResponse{
		PatternID:    id,
		Pattern:      pattern,
		TotalSamples: len(req.SampleData),
		MatchCount:   matchCount,
		MatchRate:    matchRate,
		Results:      results,
		ExecutionMs:  executionMs,
	})
}

// getRegexStats returns match frequency statistics for a pattern.
func getRegexStats(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		shared.ErrBadRequest(c, "pattern ID required")
		return
	}

	var stats RegexStats
	err := shared.ReadDB().QueryRow(c.Request.Context(), `
		SELECT
			p.id,
			COALESCE(s.total_scans, 0),
			COALESCE(s.total_matches, 0),
			COALESCE(s.total_fields_checked, 0),
			CASE WHEN COALESCE(s.total_fields_checked, 0) > 0
				THEN CAST(s.total_matches AS FLOAT) / s.total_fields_checked * 100
				ELSE 0 END,
			COALESCE(s.false_positives, 0),
			p.false_positive_rate,
			s.last_matched_at
		FROM custom_regex_patterns p
		LEFT JOIN custom_regex_stats s ON s.pattern_id = p.id
		WHERE p.id = $1
	`, id).Scan(
		&stats.PatternID, &stats.TotalScans, &stats.TotalMatches,
		&stats.TotalFields, &stats.MatchRate,
		&stats.FalsePositives, &stats.FalsePositiveRate,
		&stats.LastMatchedAt,
	)
	if err != nil {
		shared.ErrNotFound(c, "pattern not found")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// validateRegexPattern checks that a pattern is valid and compilable.
func validateRegexPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern must not be empty")
	}

	if !utf8.ValidString(pattern) {
		return fmt.Errorf("pattern contains invalid UTF-8")
	}

	if len(pattern) > 10000 {
		return fmt.Errorf("pattern exceeds maximum length of 10000 characters")
	}

	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("regex compilation failed: %w", err)
	}

	return nil
}

// isCatastrophicBacktracking detects patterns that could cause exponential backtracking.
// It checks for nested quantifiers, the most common source of ReDoS vulnerabilities.
func isCatastrophicBacktracking(pattern string) bool {
	// Detect nested quantifiers: (a+)+ , (a*)* , (a+)*  etc.
	// These are the primary cause of catastrophic backtracking.
	dangerousPatterns := []string{
		`\([^)]*[+*][^)]*\)[+*]`,       // (x+)+ or (x*)+ etc.
		`\([^)]*[+*][^)]*\)\{`,          // (x+){n} etc.
		`\([^)]*\|[^)]*\)[+*]`,          // (a|b)+ with overlapping alternatives
	}

	for _, dp := range dangerousPatterns {
		re, err := regexp.Compile(dp)
		if err != nil {
			continue
		}
		if re.MatchString(pattern) {
			return true
		}
	}

	// Check for deeply nested groups with quantifiers
	depth := 0
	quantifierAtDepth := make(map[int]bool)
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '(':
			if i == 0 || pattern[i-1] != '\\' {
				depth++
			}
		case ')':
			if i == 0 || pattern[i-1] != '\\' {
				if i+1 < len(pattern) && (pattern[i+1] == '+' || pattern[i+1] == '*') {
					if quantifierAtDepth[depth] {
						return true // Nested quantifier at same or inner depth
					}
					quantifierAtDepth[depth] = true
				}
				depth--
				if depth < 0 {
					depth = 0
				}
			}
		case '+', '*':
			if i > 0 && pattern[i-1] != '\\' && pattern[i-1] != ')' {
				quantifierAtDepth[depth] = true
			}
		}
	}

	return false
}
