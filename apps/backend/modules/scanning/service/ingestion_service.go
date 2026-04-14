package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/domain/repository"
	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/arc-platform/backend/modules/shared/scoring"
	"github.com/arc-platform/backend/pkg/normalization"
	"github.com/arc-platform/backend/pkg/validators"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var findingsIngestedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "findings_ingested_total",
		Help: "Total findings ingested by detector type",
	},
	[]string{"detector_type"},
)

// IngestionService handles scan ingestion and normalization
type IngestionService struct {
	repo         *persistence.PostgresRepository
	classifier   *ClassificationService
	enrichment   *EnrichmentService
	assetManager interfaces.AssetManager
	encryptor    *encryption.EncryptionService // nil when ENCRYPTION_KEY not set (dev fallback)
	neo4jRepo    *persistence.Neo4jRepository  // nil when Neo4j is unavailable; writes PII_Category edges
}

// NewIngestionService creates a new ingestion service
func NewIngestionService(
	repo *persistence.PostgresRepository,
	classifier *ClassificationService,
	enrichment *EnrichmentService,
	assetManager interfaces.AssetManager,
	encryptor *encryption.EncryptionService,
) *IngestionService {
	return &IngestionService{
		repo:         repo,
		classifier:   classifier,
		enrichment:   enrichment,
		assetManager: assetManager,
		encryptor:    encryptor,
	}
}

// WithNeo4jRepo wires the Neo4j repository for PII_Category edge syncing (C4/C5).
func (s *IngestionService) WithNeo4jRepo(neo4jRepo *persistence.Neo4jRepository) *IngestionService {
	s.neo4jRepo = neo4jRepo
	return s
}

// HawkeyeScanInput represents the Hawk-eye scanner JSON format.
// Sources holds findings keyed by connector name (e.g. "fs", "postgresql", "bigquery").
// Custom UnmarshalJSON captures any JSON key that isn't "scan_id" into Sources,
// so new connectors never require a struct change.
type HawkeyeScanInput struct {
	ScanID  string                      `json:"scan_id"`
	Sources map[string][]HawkeyeFinding `json:"-"`
}

// UnmarshalJSON populates ScanID and routes all other keys into Sources.
func (h *HawkeyeScanInput) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if v, ok := raw["scan_id"]; ok {
		if err := json.Unmarshal(v, &h.ScanID); err != nil {
			return err
		}
	}
	h.Sources = make(map[string][]HawkeyeFinding)
	for key, val := range raw {
		if key == "scan_id" {
			continue
		}
		var findings []HawkeyeFinding
		if err := json.Unmarshal(val, &findings); err != nil {
			continue // non-finding keys (future metadata) — skip gracefully
		}
		if len(findings) > 0 {
			h.Sources[key] = findings
		}
	}
	return nil
}

// HawkeyeFinding represents a single finding from Hawk-eye
type HawkeyeFinding struct {
	Host                string         `json:"host"`
	FilePath            string         `json:"file_path"`
	PatternName         string         `json:"pattern_name"`
	Matches             []string       `json:"matches"`
	SampleText          string         `json:"sample_text"`
	Profile             string         `json:"profile"`
	DataSource          string         `json:"data_source"`
	FileData            map[string]any `json:"file_data"`
	Severity            string         `json:"severity"`
	SeverityDescription string         `json:"severity_description"`
	DetectorType        string         `json:"detector_type"`
}

// IngestScanResult represents the result of ingestion
type IngestScanResult struct {
	ScanRunID     uuid.UUID `json:"scan_run_id"`
	TotalFindings int       `json:"total_findings"`
	TotalAssets   int       `json:"total_assets"`
	AssetsCreated int       `json:"assets_created"`
	PatternsFound int       `json:"patterns_found"`
}

// normalizeSeverity maps scanner severity terms to canonical internal terms.
func normalizeSeverity(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "highest", "critical":
		return "Critical"
	case "high":
		return "High"
	case "medium", "moderate":
		return "Medium"
	case "low":
		return "Low"
	default:
		return s
	}
}

// IngestScan processes Hawk-eye scan output and normalizes it into the database
func (s *IngestionService) IngestScan(ctx context.Context, input *HawkeyeScanInput) (retResult *IngestScanResult, retErr error) {
	if len(input.Sources) == 0 {
		return nil, fmt.Errorf("no findings in scan input")
	}

	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant ID: %w", err)
	}

	// BEGIN TRANSACTION - Critical fix for ING-001
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback on panic — use named returns so recover can propagate the error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("PANIC during ingestion, transaction rolled back: %v", r)
			retErr = fmt.Errorf("panic during ingestion: %v", r)
		}
	}()

	// Combine findings from all data sources — use fresh slice to avoid mutating caller's backing array
	totalLen := 0
	for _, findings := range input.Sources {
		totalLen += len(findings)
	}
	allFindings := make([]HawkeyeFinding, 0, totalLen)
	for _, findings := range input.Sources {
		allFindings = append(allFindings, findings...)
	}

	// Try to link to existing ScanRun if ScanID is provided in input
	// (Check first finding's custom field or top-level metadata if available)
	var scanRun *entity.ScanRun

	// Check if scan_id is in the input (we added this field to struct)
	if input.ScanID != "" {
		if id, err := uuid.Parse(input.ScanID); err == nil {
			scanRun, err = s.repo.GetScanRunByID(ctx, id)
			if err != nil {
				log.Printf("WARNING: specific scan_id %s not found, creating new", input.ScanID)
			}
		}
	}

	// Also check first finding's profile or metadata if available
	if scanRun == nil && len(allFindings) > 0 {
		// Fallback logic could go here
	}

	if scanRun == nil {
		profileName := allFindings[0].Profile
		if profileName == "" {
			profileName = "default"
		}

		scanRun = &entity.ScanRun{
			ID:              uuid.New(),
			ProfileName:     profileName,
			ScanStartedAt:   time.Now().Add(-5 * time.Minute), // Approximate
			ScanCompletedAt: time.Now(),
			Host:            allFindings[0].Host,
			Status:          "completed",
			Metadata:        map[string]any{},
		}

		if err := tx.CreateScanRun(ctx, scanRun); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create scan run: %w", err)
		}
	} else {
		// Update existing scan run
		scanRun.Status = "completed"
		scanRun.ScanCompletedAt = time.Now()
		if scanRun.Metadata == nil {
			scanRun.Metadata = make(map[string]any)
		}

		if err := tx.UpdateScanRun(ctx, scanRun); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update scan run: %w", err)
		}
	}

	// Track created assets and patterns
	assetMap := make(map[string]uuid.UUID)                // stableID -> UUID
	patternMap := make(map[string]uuid.UUID)              // pattern name -> UUID (for getOrCreatePattern cache)
	assetPIIMap := make(map[uuid.UUID]map[string]int)    // C4/C5: assetID -> piiType -> count for Neo4j PII_Category sync
	assetsCreated := 0
	uniquePatternCount := 0
	seenPatterns := make(map[string]bool)

	// Process each finding
	for _, hawkeyeFinding := range allFindings {
		// Build asset from finding data
		asset := s.buildAssetFromFinding(&hawkeyeFinding, scanRun)

		// Delegate asset creation to AssetManager (single source of truth)
		assetID, isNew, err := s.assetManager.CreateOrUpdateAsset(ctx, asset)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create/update asset: %w", err)
		}

		assetMap[asset.StableID] = assetID
		if isNew {
			assetsCreated++
		}

		// Get or create pattern
		patternID, err := s.getOrCreatePattern(ctx, &hawkeyeFinding, patternMap)
		if err != nil {
			return nil, fmt.Errorf("failed to get/create pattern: %w", err)
		}
		if !seenPatterns[hawkeyeFinding.PatternName] {
			seenPatterns[hawkeyeFinding.PatternName] = true
			uniquePatternCount++
		}

		// ENRICHMENT LAYER - Add contextual intelligence
		// Extract column name if this is a database finding
		columnName := ""
		if colVal, ok := hawkeyeFinding.FileData["column_name"]; ok {
			if colStr, ok := colVal.(string); ok {
				columnName = colStr
			}
		}

		matchSample := ""
		if len(hawkeyeFinding.Matches) > 0 {
			matchSample = hawkeyeFinding.Matches[0]
		}

		// CRITICAL FIX #3: Normalize before classification
		normalizedMatch := normalization.Normalize(matchSample)

		// Format validation — reject false positives before expensive classification
		piiType := validators.MapPatternToPIIType(hawkeyeFinding.PatternName)
		if piiType != "" && !validators.Validate(piiType, normalizedMatch) {
			log.Printf("DEBUG: Format validation rejected %s (%s): %q",
				hawkeyeFinding.PatternName, piiType, normalizedMatch)
			continue
		}

		// Perform enrichment
		enrichmentSignals := s.enrichment.Enrich(ctx, EnrichmentContext{
			FilePath:    hawkeyeFinding.FilePath,
			MatchValue:  normalizedMatch, // Use normalized value
			PatternName: hawkeyeFinding.PatternName,
			AssetType:   "file",
			ColumnName:  columnName,
		})

		// Calculate enrichment score (this becomes the Context Score in multi-signal)
		enrichmentScore := s.enrichment.GetEnrichmentScore(enrichmentSignals)

		// Classify finding using multi-signal engine
		multiSignalInput := MultiSignalInput{
			PatternName:       hawkeyeFinding.PatternName,
			FilePath:          hawkeyeFinding.FilePath,
			MatchValue:        normalizedMatch,
			ColumnName:        columnName,
			FileData:          hawkeyeFinding.FileData,
			EnrichmentScore:   enrichmentScore,
			EnrichmentSignals: enrichmentSignals,
		}

		decision, err := s.classifier.ClassifyMultiSignal(ctx, multiSignalInput)
		if err != nil {
			log.Printf("WARN: classification failed for %s: %v — skipping finding", hawkeyeFinding.PatternName, err)
			continue
		}

		// Filter Non-PII at ingestion time (60-80% DB size reduction)
		// Only store findings that are confirmed PII with sufficient confidence
		if decision.Classification == "Non-PII" || decision.FinalScore < 0.45 {
			// Skip low-confidence and Non-PII findings to prevent database bloat
			continue
		}

		// Sanitize inputs for Postgres (remove null bytes) with logging
		sanitizedMatches := make([]string, len(hawkeyeFinding.Matches))
		sanitizationCount := 0
		for i, m := range hawkeyeFinding.Matches {
			if strings.Contains(m, "\u0000") {
				sanitizationCount++
				log.Printf("WARNING: Null byte detected in finding %s at %s (removed)",
					hawkeyeFinding.PatternName, hawkeyeFinding.FilePath)
			}
			sanitizedMatches[i] = strings.ReplaceAll(m, "\u0000", "")
		}
		sanitizedSample := strings.ReplaceAll(hawkeyeFinding.SampleText, "\u0000", "")

		// Encrypt PII sample value at rest (AES-256-GCM) — DPDPA security requirement.
		// If encryptor is unavailable (dev/test without ENCRYPTION_KEY), store as-is with a warning.
		storedSample := sanitizedSample
		if s.encryptor != nil && sanitizedSample != "" {
			encrypted, encErr := s.encryptor.EncryptString(sanitizedSample)
			if encErr != nil {
				log.Printf("WARNING: Failed to encrypt sample text for %s: %v — storing redacted",
					hawkeyeFinding.PatternName, encErr)
				storedSample = "[ENCRYPTION_FAILED]"
			} else {
				storedSample = encrypted
			}
		} else if s.encryptor == nil {
			log.Printf("WARNING: EncryptionService unavailable — PII sample stored unencrypted for %s",
				hawkeyeFinding.PatternName)
		}

		// Track sanitization in scan metadata
		if sanitizationCount > 0 {
			if scanRun.Metadata == nil {
				scanRun.Metadata = make(map[string]any)
			}
			if existingCount, ok := scanRun.Metadata["sanitized_findings"].(int); ok {
				scanRun.Metadata["sanitized_findings"] = existingCount + sanitizationCount
			} else {
				scanRun.Metadata["sanitized_findings"] = sanitizationCount
			}
		}

		// Convert enrichment signals to map for storage
		enrichmentMap := map[string]any{
			"asset_semantics":   enrichmentSignals.AssetSemantics,
			"environment":       enrichmentSignals.Environment,
			"entropy":           enrichmentSignals.Entropy,
			"charset_diversity": enrichmentSignals.CharsetDiversity,
			"token_shape":       enrichmentSignals.TokenShape,
			"value_hash":        enrichmentSignals.ValueHash,
			"historical_count":  enrichmentSignals.HistoricalCount,
		}

		// Generate normalized hash for deduplication
		// Use pkg/normalization when available, inline implementation for now
		normalizedValue := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(matchSample, " ", ""), "-", ""))
		hash := sha256.Sum256([]byte(normalizedValue))
		valueHash := hex.EncodeToString(hash[:])

		// Check for duplicates (same asset, pattern, and value hash in this scan)
		// Note: This requires adding GetFindingByHash to repository interface
		// For now, we'll just add the hash and rely on the unique index to prevent duplication
		// The database migration 000003_add_deduplication.up.sql adds:
		// CREATE UNIQUE INDEX idx_findings_unique ON findings(asset_id, pattern_name, normalized_value_hash, scan_run_id)

		// Calculate dynamic severity based on classification, confidence, and context
		dynamicSeverity := calculateDynamicSeverity(
			decision.Classification,
			decision.ConfidenceLevel,
			hawkeyeFinding.FileData,
		)

		// Calculate risk score for prioritization (0-100) via the canonical shared scorer.
		// This replaces the local formula to ensure consistency with hawk/backend and
		// discovery/service/risk_engine.go.
		accessExposure := 0.5 // default: internal
		if isProductionEnvironment(hawkeyeFinding.FileData) {
			accessExposure = 1.0
		}
		riskScore := int(scoring.ComputeRiskScore(scoring.RiskScoreParams{
			PIIType:        decision.Classification,
			Confidence:     decision.FinalScore,
			PIIDensity:     enrichmentScore, // enrichment score proxies density context
			AccessExposure: accessExposure,
		}))

		// Classification: Test vs Prod
		environment := "PROD"
		status := "pending"

		if isTestArtifact(hawkeyeFinding.FilePath) || isSemanticTestData(hawkeyeFinding.SampleText) {
			environment = "TEST"
			status = "ignored"
		}

		// Create finding with deduplication hash
		finding := &entity.Finding{
			ID:                  uuid.New(),
			TenantID:            tenantID,
			NormalizedValueHash: valueHash,
			ScanRunID:           scanRun.ID,
			AssetID:             assetID,
			PatternID:           &patternID,
			PatternName:         hawkeyeFinding.PatternName,
			Matches:             sanitizedMatches,
			SampleText:          storedSample,
			Severity:            dynamicSeverity, // Now calculated from classification+confidence+context
			SeverityDescription: fmt.Sprintf("Risk Score: %d/100 | %s", riskScore, decision.Justification),
			ConfidenceScore:     &decision.FinalScore,
			Environment:         environment,
			Context:             decision.SignalBreakdown,
			EnrichmentSignals:   enrichmentMap,
			EnrichmentScore:     &enrichmentScore,
			EnrichmentFailed:    enrichmentSignals.EnrichmentFailed,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}

		if err := tx.CreateFinding(ctx, finding); err != nil {
			// Check if error is due to duplicate (unique constraint violation)
			if strings.Contains(err.Error(), "idx_findings_unique") {
				// Duplicate detected - skip silently or log
				log.Printf("DEBUG: Duplicate finding skipped for %s at %s", hawkeyeFinding.PatternName, hawkeyeFinding.FilePath)
				continue
			}
			tx.Rollback()
			return nil, fmt.Errorf("failed to create finding: %w", err)
		}

		// Increment ingestion counter by detector type
		detectorType := hawkeyeFinding.DetectorType
		if detectorType == "" {
			detectorType = "regex"
		}
		findingsIngestedTotal.WithLabelValues(detectorType).Inc()

		// Save Classification
		classification := &entity.Classification{
			ID:                 uuid.New(),
			FindingID:          finding.ID,
			ClassificationType: decision.Classification,
			SubCategory:        decision.SubCategory,
			ConfidenceScore:    decision.FinalScore,
			Justification:      decision.Justification,
			DPDPACategory:      decision.DPDPACategory,
			RequiresConsent:    decision.RequiresConsent,
		}

		if err := tx.CreateClassification(ctx, classification); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create classification: %w", err)
		}

		// Create review state (Logic moved upstream)
		reviewState := &entity.ReviewState{
			ID:        uuid.New(),
			FindingID: finding.ID,
			Status:    status,
		}

		if err := tx.CreateReviewState(ctx, reviewState); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create review state: %w", err)
		}

		// C4/C5: Accumulate PII type counts per asset for Neo4j PII_Category sync after commit
		piiTypeName := hawkeyeFinding.PatternName
		if _, ok := assetPIIMap[assetID]; !ok {
			assetPIIMap[assetID] = make(map[string]int)
		}
		assetPIIMap[assetID][piiTypeName]++
	}

	// Collect asset IDs for post-commit risk recalculation
	assetIDs := make([]uuid.UUID, 0, len(assetMap))
	for _, assetID := range assetMap {
		assetIDs = append(assetIDs, assetID)
	}

	// Update scan run totals
	scanRun.TotalFindings = len(allFindings)
	scanRun.TotalAssets = len(assetMap)
	if err := tx.UpdateScanRun(ctx, scanRun); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update scan run: %w", err)
	}

	// CRITICAL FIX: Commit the transaction to persist all changes
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Recalculate asset risk AFTER commit so CountFindings sees committed rows
	for _, assetID := range assetIDs {
		if err := s.recalculateAssetRisk(ctx, assetID); err != nil {
			log.Printf("Error recalculating risk for asset %s: %v", assetID, err)
		}
	}

	// C4/C5: Sync PII categories to Neo4j for lineage graph (3-level contract: System → Asset → PII_Category)
	// This runs after commit to avoid blocking the transaction. Failures are non-fatal (logged only).
	if s.neo4jRepo != nil {
		for assetID, piiTypes := range assetPIIMap {
			if err := s.neo4jRepo.SyncFindingsToPIICategories(ctx, assetID.String(), piiTypes); err != nil {
				log.Printf("WARN: neo4j PII category sync failed for asset %s: %v", assetID, err)
			}
		}
	}

	return &IngestScanResult{
		ScanRunID:     scanRun.ID,
		TotalFindings: scanRun.TotalFindings,
		TotalAssets:   scanRun.TotalAssets,
		AssetsCreated: assetsCreated,
		PatternsFound: uniquePatternCount,
	}, nil
}

// recalculateAssetRisk derives the risk score from findings severity and count
func (s *IngestionService) recalculateAssetRisk(ctx context.Context, assetID uuid.UUID) error {
	// 1. Get total findings count
	// We could use CountFindings, but we need max severity too.
	// Let's rely on the repository to give us stats or query findings.

	// For now, simpler: Get ALL findings for this asset (lightweight if paginated/limited, but potentially heavy)
	// BETTER: Add a method to repo: GetAssetRiskData(assetID) -> (count, maxSeverity)
	// Since I can't easily modify the repo interface without touching multiple files,
	// I will use ListFindings logic with a limit, or just count.

	// Actually, I can use CountFindings for count.
	count, err := s.repo.CountFindings(ctx, repository.FindingFilters{
		AssetID: &assetID,
	})
	if err != nil {
		return err
	}

	// 2. Determine Max Severity
	// We verify if there are ANY 'Critical' or 'High' findings.
	hasCritical, err := s.hasFindingWithSeverity(ctx, assetID, "Critical")
	if err != nil {
		return err
	}

	hasHigh, err := s.hasFindingWithSeverity(ctx, assetID, "High")
	if err != nil {
		return err
	}

	// 3. Calculate Base Score
	baseScore := 10
	if hasCritical {
		baseScore = 95
	} else if hasHigh {
		if count > 3 {
			baseScore = 85 // High volume of High severity
		} else {
			baseScore = 75
		}
	} else if count > 0 {
		// Medium/Low
		if count > 10 {
			baseScore = 60
		} else {
			baseScore = 40
		}
	}

	// 4. Update Asset
	return s.repo.UpdateAssetStats(ctx, assetID, baseScore, count)
}

func (s *IngestionService) hasFindingWithSeverity(ctx context.Context, assetID uuid.UUID, severity string) (bool, error) {
	// Normalize to canonical term before querying
	canonical := normalizeSeverity(severity)

	count, err := s.repo.CountFindings(ctx, repository.FindingFilters{
		AssetID:  &assetID,
		Severity: canonical,
	})

	return count > 0, err
}

// getOrCreatePattern gets existing pattern or creates new one
func (s *IngestionService) getOrCreatePattern(ctx context.Context, finding *HawkeyeFinding, patternMap map[string]uuid.UUID) (uuid.UUID, error) {
	// Check cache
	if id, exists := patternMap[finding.PatternName]; exists {
		return id, nil
	}

	// Check database
	existingPattern, err := s.repo.GetPatternByName(ctx, finding.PatternName)
	if err != nil {
		return uuid.Nil, err
	}

	if existingPattern != nil {
		patternMap[finding.PatternName] = existingPattern.ID
		return existingPattern.ID, nil
	}

	// Create new pattern
	pattern := &entity.Pattern{
		ID:                uuid.New(),
		Name:              finding.PatternName,
		PatternType:       "regex",
		Category:          categorizePattern(finding.PatternName),
		Description:       fmt.Sprintf("Pattern for detecting: %s", finding.PatternName),
		PatternDefinition: "",
		IsActive:          true,
	}

	if err := s.repo.CreatePattern(ctx, pattern); err != nil {
		return uuid.Nil, err
	}

	patternMap[finding.PatternName] = pattern.ID
	return pattern.ID, nil
}


// generateStableID creates a stable identifier from asset identifier
// getFileName extracts filename from path
func getFileName(path string) string {
	// Simple extraction - in production use filepath.Base
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// calculateRiskScore converts severity to numeric risk score
func calculateRiskScore(severity string) int {
	switch severity {
	case "Critical":
		return 95
	case "High":
		return 80
	case "Medium":
		return 60
	case "Low":
		return 30
	default:
		return 10
	}
}

// contains checks if string contains any of the substrings
func contains(str string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(str, substr) {
			return true
		}
	}
	return false
}

// categorizePattern determines pattern category
func categorizePattern(patternName string) string {
	lowerName := strings.ToLower(patternName)

	if contains(lowerName, []string{"email", "phone", "ssn", "passport", "license", "aadhaar", "aadhar", "pan"}) {
		return "pii"
	}
	if contains(lowerName, []string{"key", "token", "secret", "password", "api", "aws"}) {
		return "secret"
	}
	if contains(lowerName, []string{"credit", "card", "account", "bank"}) {
		return "financial"
	}

	return "other"
}

// buildAssetFromFinding constructs an asset entity from finding data
func (s *IngestionService) buildAssetFromFinding(finding *HawkeyeFinding, scanRun *entity.ScanRun) *entity.Asset {
	// Extract owner from file data if available
	owner := "Platform Team"
	if val, ok := finding.FileData["owner"].(string); ok {
		owner = val
	}

	// Map profile to environment
	env := "Production"
	if scanRun.ProfileName == "test_scan" || scanRun.ProfileName == "dev" {
		env = "Development"
	}

	return &entity.Asset{
		AssetType:    "file",
		Name:         getFileName(finding.FilePath),
		Path:         finding.FilePath,
		DataSource:   finding.DataSource,
		Host:         finding.Host,
		Environment:  env,
		Owner:        owner,
		SourceSystem: fmt.Sprintf("%s://%s", finding.DataSource, finding.Host),
		FileMetadata: finding.FileData,
		RiskScore:    calculateRiskScore(finding.Severity),
		// StableID will be generated by AssetService
	}
}

// Ensure interface compatibility
var _ json.Marshaler = (*HawkeyeScanInput)(nil)

func (h *HawkeyeScanInput) MarshalJSON() ([]byte, error) {
	type Alias HawkeyeScanInput
	return json.Marshal(&struct{ *Alias }{Alias: (*Alias)(h)})
}

// GetLatestScan returns the most recent scan run
func (s *IngestionService) GetLatestScan(ctx context.Context) (*entity.ScanRun, error) {
	return s.repo.GetLatestScanRun(ctx)
}

// calculateDynamicSeverity determines severity based on classification, confidence, and environment
// This creates coherence between severity, classification, and confidence for better interpretability
func calculateDynamicSeverity(classification, confidence string, fileData map[string]any) string {
	// Determine if this is production environment
	isProduction := isProductionEnvironment(fileData)

	// Apply decision matrix: Classification + Confidence + Context = Severity
	switch classification {
	case "Sensitive Personal Data":
		// SSN, PAN, Aadhaar, Credit Cards, etc.
		if confidence == "CONFIRMED" && isProduction {
			return "Critical"
		}
		if confidence == "CONFIRMED" || (confidence == "HIGH_CONFIDENCE" && isProduction) {
			return "High"
		}
		if isProduction {
			return "High"
		}
		return "Medium"

	case "Personal Data":
		// Email, Phone, etc.
		if confidence == "CONFIRMED" && isProduction {
			return "Medium"
		}
		if isProduction {
			return "Low"
		}
		return "Low"

	case "Secrets":
		// API Keys, AWS Keys, etc.
		if confidence == "CONFIRMED" && isProduction {
			return "Critical"
		}
		if isProduction {
			return "High"
		}
		return "Medium"

	default:
		// Non-PII or unknown
		return "Info"
	}
}

// isProductionEnvironment determines if data is from production environment
func isProductionEnvironment(fileData map[string]any) bool {
	if fileData == nil {
		return true // Default to production if unknown (safer)
	}

	// Check environment field
	if env, ok := fileData["environment"].(string); ok {
		envLower := strings.ToLower(env)
		// Non-production indicators
		if strings.Contains(envLower, "test") ||
			strings.Contains(envLower, "dev") ||
			strings.Contains(envLower, "staging") ||
			strings.Contains(envLower, "qa") ||
			strings.Contains(envLower, "sandbox") {
			return false
		}
	}

	// Check database/schema names for test indicators
	if dbName, ok := fileData["database"].(string); ok {
		dbLower := strings.ToLower(dbName)
		if strings.Contains(dbLower, "test") || strings.Contains(dbLower, "dev") {
			return false
		}
	}

	// Default to production
	return true
}

// calculateComprehensiveRiskScore is SUPERSEDED by scoring.ComputeRiskScore().
// Kept here temporarily so git history shows the migration path.
// It is no longer called from IngestScan — use the canonical scorer instead.
//
// Deprecated: Use scoring.ComputeRiskScore(scoring.RiskScoreParams{...}) from
// github.com/arc-platform/backend/modules/shared/scoring.
func calculateComprehensiveRiskScore(classification, confidence string, fileData map[string]any) int {
	// Base weights for classification types
	var classificationWeight float64
	switch classification {
	case "Sensitive Personal Data":
		classificationWeight = 100.0
	case "Secrets":
		classificationWeight = 90.0
	case "Personal Data":
		classificationWeight = 50.0
	default:
		classificationWeight = 10.0
	}

	// Confidence multiplier
	var confidenceMultiplier float64
	switch confidence {
	case "CONFIRMED":
		confidenceMultiplier = 1.0
	case "HIGH_CONFIDENCE":
		confidenceMultiplier = 0.75
	case "VALIDATED":
		confidenceMultiplier = 0.5
	default:
		confidenceMultiplier = 0.3
	}

	// Environment context multiplier
	contextMultiplier := 1.0
	if !isProductionEnvironment(fileData) {
		contextMultiplier = 0.3 // Test/dev data is 70% less critical
	}

	// Calculate weighted score
	// Formula: (ClassWeight * 0.6) + (Confidence * 20) + (Context * 20)
	// This ensures classification type dominates, but confidence/context can adjust prioritization

	baseScore := classificationWeight * 0.6

	//     environment = "TEST"
	// }
	confidenceScore := (confidenceMultiplier * 100) * 0.2
	contextScore := (contextMultiplier * 100) * 0.2

	totalScore := int(baseScore + confidenceScore + contextScore)

	// Ensure bounds 0-100
	if totalScore > 100 {
		return 100
	}
	if totalScore < 0 {
		return 0
	}

	return totalScore
}

// ClearAllScanData deletes all previous scan data for clean scan-replace workflow
func (s *IngestionService) ClearAllScanData(ctx context.Context) error {
	log.Println("Clearing all previous scan data...")
	_, err := s.repo.GetDB().ExecContext(ctx, `
		TRUNCATE findings, assets, classifications, 
		asset_relationships, review_states, scan_runs, finding_feedback 
		CASCADE
	`)
	if err != nil {
		return fmt.Errorf("failed to clear scan data: %w", err)
	}

	log.Println("✅ All previous scan data cleared successfully")
	return nil
}

// isTestArtifact checks if the file path indicates a test or mock file
func isTestArtifact(path string) bool {
	lowerPath := strings.ToLower(path)

	// Check for common test directories
	if strings.Contains(lowerPath, "/test/") ||
		strings.Contains(lowerPath, "/tests/") ||
		strings.Contains(lowerPath, "/mock/") ||
		strings.Contains(lowerPath, "/mocks/") ||
		strings.Contains(lowerPath, "/fixture/") ||
		strings.Contains(lowerPath, "/fixtures/") ||
		strings.Contains(lowerPath, "/spec/") ||
		strings.Contains(lowerPath, "/specs/") {
		return true
	}

	// Check for common test file extensions/suffixes
	if strings.HasSuffix(lowerPath, "_test.go") ||
		strings.HasSuffix(lowerPath, ".test.ts") ||
		strings.HasSuffix(lowerPath, ".spec.ts") ||
		strings.HasSuffix(lowerPath, ".test.js") ||
		strings.HasSuffix(lowerPath, ".spec.js") ||
		strings.HasSuffix(lowerPath, ".mock.json") ||
		strings.HasSuffix(lowerPath, ".test.json") {
		return true
	}

	return false
}

// isSemanticTestData checks if the sample text contains common test data patterns
func isSemanticTestData(sampleText string) bool {
	lower := strings.ToLower(sampleText)
	if strings.Contains(lower, "john doe") ||
		strings.Contains(lower, "example") ||
		strings.Contains(lower, "test@") ||
		strings.Contains(sampleText, "1234567890") {
		return true
	}
	return false
}
