package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// AuditResult represents the result of an integrity check
type AuditResult struct {
	TestName      string    `json:"test_name"`
	Status        string    `json:"status"` // PASS, FAIL, WARNING, CRITICAL
	Details       string    `json:"details"`
	CountAffected int       `json:"count_affected"`
	Timestamp     time.Time `json:"timestamp"`
}

// AuditReport represents a complete audit report
type AuditReport struct {
	Results       []AuditResult `json:"results"`
	Summary       Summary       `json:"summary"`
	GeneratedAt   time.Time     `json:"generated_at"`
	TotalFindings int           `json:"total_findings"`
}

// Summary provides overall system health
type Summary struct {
	Critical int `json:"critical"`
	Fail     int `json:"fail"`
	Warning  int `json:"warning"`
	Pass     int `json:"pass"`
}

// FindingsValidator performs comprehensive integrity checks
type FindingsValidator struct {
	db *sql.DB
}

// NewFindingsValidator creates a new validator instance
func NewFindingsValidator(dbURL string) (*FindingsValidator, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &FindingsValidator{db: db}, nil
}

// RunComprehensiveAudit performs all integrity checks
func (v *FindingsValidator) RunComprehensiveAudit(ctx context.Context) (*AuditReport, error) {
	log.Println("Starting comprehensive findings integrity audit...")

	report := &AuditReport{
		GeneratedAt: time.Now(),
		Results:     make([]AuditResult, 0),
	}

	// Get total findings count
	totalFindings, err := v.getTotalFindings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total findings: %w", err)
	}
	report.TotalFindings = totalFindings

	// 1. Finding Structure Verification
	log.Println("Verifying finding structure...")
	report.Results = append(report.Results, v.verifyFindingStructure(ctx)...)

	// 2. Scan-Finding Relationship Verification
	log.Println("Verifying scan-finding relationships...")
	report.Results = append(report.Results, v.verifyScanFindingRelationship(ctx)...)

	// 3. Asset-Finding Relationship Verification
	log.Println("Verifying asset-finding relationships...")
	report.Results = append(report.Results, v.verifyAssetFindingRelationship(ctx)...)

	// 4. Location Traceability Verification
	log.Println("Verifying location traceability...")
	report.Results = append(report.Results, v.verifyLocationTraceability(ctx)...)

	// 5. Validation Logic Reference Verification
	log.Println("Verifying validation logic references...")
	report.Results = append(report.Results, v.verifyValidationLogicReference(ctx)...)

	// 6. Aggregation Integrity Verification
	log.Println("Verifying aggregation integrity...")
	report.Results = append(report.Results, v.verifyAggregationIntegrity(ctx)...)

	// 7. Cascade Deletion Behavior Verification
	log.Println("Verifying cascade deletion behavior...")
	report.Results = append(report.Results, v.verifyCascadeDeletionBehavior(ctx)...)

	// 8. Complete Traceability Test
	log.Println("Verifying complete traceability...")
	report.Results = append(report.Results, v.verifyCompleteTraceability(ctx)...)

	// Generate summary
	report.Summary = v.generateSummary(report.Results)

	log.Printf("Audit completed. Critical: %d, Fail: %d, Warning: %d, Pass: %d",
		report.Summary.Critical, report.Summary.Fail, report.Summary.Warning, report.Summary.Pass)

	return report, nil
}

// verifyFindingStructure checks required fields and data validity
func (v *FindingsValidator) verifyFindingStructure(ctx context.Context) []AuditResult {
	var results []AuditResult

	// Check required fields
	query := `
		SELECT COUNT(*) 
		FROM findings f 
		WHERE f.scan_run_id IS NULL 
		   OR f.asset_id IS NULL 
		   OR f.pattern_name IS NULL 
		   OR f.pattern_name = ''
		   OR f.severity IS NULL 
		   OR f.severity = ''
	`

	var count int
	err := v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "finding_required_fields",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking required fields: %v", err),
		})
		return results
	}

	status := "PASS"
	details := "All findings have required fields"
	if count > 0 {
		status = "FAIL"
		details = fmt.Sprintf("Findings missing required fields: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "finding_required_fields",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	// Check valid PII types
	query = `
		SELECT COUNT(*) 
		FROM findings f 
		LEFT JOIN patterns p ON f.pattern_name = p.name 
		WHERE p.name IS NULL
	`

	err = v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "valid_pii_types",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking PII types: %v", err),
		})
		return results
	}

	status = "PASS"
	details = "All findings use valid PII patterns"
	if count > 0 {
		status = "FAIL"
		details = fmt.Sprintf("Findings with invalid PII types: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "valid_pii_types",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	// Check confidence score range
	query = `
		SELECT COUNT(*) 
		FROM findings f 
		WHERE f.confidence_score IS NOT NULL 
		  AND (f.confidence_score < 0.0 OR f.confidence_score > 1.0)
	`

	err = v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "confidence_score_range",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking confidence scores: %v", err),
		})
		return results
	}

	status = "PASS"
	details = "All confidence scores are valid (0.0-1.0)"
	if count > 0 {
		status = "FAIL"
		details = fmt.Sprintf("Findings with invalid confidence scores: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "confidence_score_range",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	return results
}

// verifyScanFindingRelationship checks for orphaned findings
func (v *FindingsValidator) verifyScanFindingRelationship(ctx context.Context) []AuditResult {
	var results []AuditResult

	// Check for orphaned findings
	query := `
		SELECT COUNT(*) 
		FROM findings f 
		LEFT JOIN scan_runs sr ON f.scan_run_id = sr.id 
		WHERE sr.id IS NULL
	`

	var count int
	err := v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "scan_finding_relationship",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking scan-finding relationship: %v", err),
		})
		return results
	}

	status := "PASS"
	details := "All findings have valid scan runs"
	if count > 0 {
		status = "CRITICAL"
		details = fmt.Sprintf("Orphaned findings without valid scan runs: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "scan_finding_relationship",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	// Verify scan run statistics
	query = `
		SELECT COUNT(*) 
		FROM scan_runs sr 
		LEFT JOIN (
			SELECT scan_run_id, COUNT(*) as actual_count
			FROM findings
			GROUP BY scan_run_id
		) fc ON sr.id = fc.scan_run_id 
		WHERE COALESCE(fc.actual_count, 0) != sr.total_findings
	`

	err = v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "scan_statistics_accuracy",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking scan statistics: %v", err),
		})
		return results
	}

	status = "PASS"
	details = "Scan run statistics are accurate"
	if count > 0 {
		status = "WARNING"
		details = fmt.Sprintf("Scan runs with inaccurate finding counts: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "scan_statistics_accuracy",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	return results
}

// verifyAssetFindingRelationship checks findings without valid assets
func (v *FindingsValidator) verifyAssetFindingRelationship(ctx context.Context) []AuditResult {
	var results []AuditResult

	// Check for findings without valid assets
	query := `
		SELECT COUNT(*) 
		FROM findings f 
		LEFT JOIN assets a ON f.asset_id = a.id 
		WHERE a.id IS NULL
	`

	var count int
	err := v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "asset_finding_relationship",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking asset-finding relationship: %v", err),
		})
		return results
	}

	status := "PASS"
	details := "All findings belong to valid assets"
	if count > 0 {
		status = "CRITICAL"
		details = fmt.Sprintf("Findings without valid assets: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "asset_finding_relationship",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	// Verify asset finding counts
	query = `
		SELECT COUNT(*) 
		FROM assets a 
		LEFT JOIN (
			SELECT asset_id, COUNT(*) as actual_count
			FROM findings
			GROUP BY asset_id
		) fc ON a.id = fc.asset_id 
		WHERE COALESCE(fc.actual_count, 0) != a.total_findings
	`

	err = v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "asset_statistics_accuracy",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking asset statistics: %v", err),
		})
		return results
	}

	status = "PASS"
	details = "Asset finding counts are accurate"
	if count > 0 {
		status = "WARNING"
		details = fmt.Sprintf("Assets with inaccurate finding counts: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "asset_statistics_accuracy",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	return results
}

// verifyLocationTraceability checks location data preservation
func (v *FindingsValidator) verifyLocationTraceability(ctx context.Context) []AuditResult {
	var results []AuditResult

	// Check findings without location data
	query := `
		SELECT COUNT(*) 
		FROM findings f 
		LEFT JOIN assets a ON f.asset_id = a.id 
		WHERE a.path IS NULL OR a.path = ''
	`

	var count int
	err := v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "location_traceability",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking location traceability: %v", err),
		})
		return results
	}

	status := "PASS"
	details := "All findings have traceable locations"
	if count > 0 {
		status = "FAIL"
		details = fmt.Sprintf("Findings without location data: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "location_traceability",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	return results
}

// verifyValidationLogicReference checks pattern references
func (v *FindingsValidator) verifyValidationLogicReference(ctx context.Context) []AuditResult {
	var results []AuditResult

	// Check findings reference valid patterns
	query := `
		SELECT COUNT(*) 
		FROM findings f 
		LEFT JOIN patterns p ON f.pattern_id = p.id 
		WHERE f.pattern_id IS NOT NULL AND p.id IS NULL
	`

	var count int
	err := v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "validation_logic_reference",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking validation logic references: %v", err),
		})
		return results
	}

	status := "PASS"
	details := "All findings reference valid patterns"
	if count > 0 {
		status = "FAIL"
		details = fmt.Sprintf("Findings referencing invalid patterns: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "validation_logic_reference",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	// Check classification completeness
	query = `
		SELECT COUNT(*) 
		FROM findings f 
		LEFT JOIN classifications c ON f.id = c.finding_id 
		WHERE c.finding_id IS NULL
	`

	err = v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "classification_completeness",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking classification completeness: %v", err),
		})
		return results
	}

	status = "PASS"
	details = "All findings have classifications"
	if count > 0 {
		status = "WARNING"
		details = fmt.Sprintf("Findings without classifications: %d", count)
	}

	results = append(results, AuditResult{
		TestName:      "classification_completeness",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	return results
}

// verifyAggregationIntegrity checks summary statistics
func (v *FindingsValidator) verifyAggregationIntegrity(ctx context.Context) []AuditResult {
	var results []AuditResult

	// Get actual findings count
	var actualCount int
	err := v.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM findings").Scan(&actualCount)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "aggregation_integrity",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error getting actual findings count: %v", err),
		})
		return results
	}

	// Get reported count from scan runs
	var reportedCount int
	err = v.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_findings), 0) FROM scan_runs").Scan(&reportedCount)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "aggregation_integrity",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error getting reported findings count: %v", err),
		})
		return results
	}

	status := "PASS"
	details := "Aggregation statistics are accurate"
	count := 0
	if actualCount != reportedCount {
		status = "CRITICAL"
		details = fmt.Sprintf("Aggregation mismatch: reported=%d, actual=%d", reportedCount, actualCount)
		count = abs(actualCount - reportedCount)
	}

	results = append(results, AuditResult{
		TestName:      "aggregation_integrity",
		Status:        status,
		Details:       details,
		CountAffected: count,
		Timestamp:     time.Now(),
	})

	return results
}

// verifyCascadeDeletionBehavior checks cascade constraints
func (v *FindingsValidator) verifyCascadeDeletionBehavior(ctx context.Context) []AuditResult {
	var results []AuditResult

	// Check cascade constraints are properly set
	query := `
		SELECT COUNT(*) 
		FROM information_schema.referential_constraints rc 
		JOIN information_schema.table_constraints tc ON rc.constraint_name = tc.constraint_name 
		WHERE rc.delete_rule = 'CASCADE' 
		AND tc.table_name IN ('findings', 'classifications', 'review_states', 'finding_feedback', 'asset_relationships')
	`

	var count int
	err := v.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "cascade_constraints",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking cascade constraints: %v", err),
		})
		return results
	}

	expectedConstraints := 6 // Expected number of cascade constraints
	status := "PASS"
	details := "All cascade constraints are properly configured"
	affected := 0
	if count != expectedConstraints {
		status = "FAIL"
		details = fmt.Sprintf("Missing cascade constraints: %d", expectedConstraints-count)
		affected = expectedConstraints - count
	}

	results = append(results, AuditResult{
		TestName:      "cascade_constraints",
		Status:        status,
		Details:       details,
		CountAffected: affected,
		Timestamp:     time.Now(),
	})

	return results
}

// verifyCompleteTraceability checks end-to-end traceability
func (v *FindingsValidator) verifyCompleteTraceability(ctx context.Context) []AuditResult {
	var results []AuditResult

	// Check complete traceability chain
	query := `
		SELECT COUNT(*) 
		FROM findings f 
		JOIN assets a ON f.asset_id = a.id 
		JOIN scan_runs sr ON f.scan_run_id = sr.id 
		LEFT JOIN classifications c ON f.id = c.finding_id 
		WHERE a.id IS NOT NULL AND sr.id IS NOT NULL
	`

	var traceableCount int
	err := v.db.QueryRowContext(ctx, query).Scan(&traceableCount)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "complete_traceability",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error checking complete traceability: %v", err),
		})
		return results
	}

	// Get total findings
	totalFindings, err := v.getTotalFindings(ctx)
	if err != nil {
		results = append(results, AuditResult{
			TestName: "complete_traceability",
			Status:   "FAIL",
			Details:  fmt.Sprintf("Error getting total findings: %v", err),
		})
		return results
	}

	status := "PASS"
	details := "All findings have complete audit trail"
	affected := 0
	if traceableCount != totalFindings {
		status = "CRITICAL"
		details = fmt.Sprintf("Findings with incomplete traceability: %d", totalFindings-traceableCount)
		affected = totalFindings - traceableCount
	}

	results = append(results, AuditResult{
		TestName:      "complete_traceability",
		Status:        status,
		Details:       details,
		CountAffected: affected,
		Timestamp:     time.Now(),
	})

	return results
}

// Helper functions
func (v *FindingsValidator) getTotalFindings(ctx context.Context) (int, error) {
	var count int
	err := v.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM findings").Scan(&count)
	return count, err
}

func (v *FindingsValidator) generateSummary(results []AuditResult) Summary {
	var summary Summary
	for _, result := range results {
		switch result.Status {
		case "CRITICAL":
			summary.Critical++
		case "FAIL":
			summary.Fail++
		case "WARNING":
			summary.Warning++
		case "PASS":
			summary.Pass++
		}
	}
	return summary
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	// Database connection string
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password123@localhost:5432/arc_platform?sslmode=disable"
	}

	// Create validator
	validator, err := NewFindingsValidator(dbURL)
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}
	defer validator.db.Close()

	// Run audit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	report, err := validator.RunComprehensiveAudit(ctx)
	if err != nil {
		log.Fatalf("Failed to run audit: %v", err)
	}

	// Output results
	jsonOutput, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal report: %v", err)
	}

	fmt.Println("=== ARC-HAWK FINDINGS INTEGRITY AUDIT REPORT ===")
	fmt.Println(string(jsonOutput))

	// Exit with appropriate code
	if report.Summary.Critical > 0 || report.Summary.Fail > 0 {
		log.Println("❌ Audit failed - Critical issues found")
		os.Exit(1)
	} else if report.Summary.Warning > 0 {
		log.Println("⚠️  Audit completed with warnings")
		os.Exit(2)
	} else {
		log.Println("✅ Audit passed - All checks successful")
		os.Exit(0)
	}
}
