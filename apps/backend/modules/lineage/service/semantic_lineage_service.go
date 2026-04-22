package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/google/uuid"
	neo4jDriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// getConfidenceThreshold reads LINEAGE_CONFIDENCE_THRESHOLD from env, defaults to 0.45
func getConfidenceThreshold() float64 {
	if val := os.Getenv("LINEAGE_CONFIDENCE_THRESHOLD"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return 0.45
}

// SemanticLineageService builds aggregated semantic lineage graphs
// Implements LineageSync interface
type SemanticLineageService struct {
	neo4jRepo        *persistence.Neo4jRepository
	pgRepo           *persistence.PostgresRepository
	findingsProvider interfaces.FindingsProvider
}

// NewSemanticLineageService creates a new semantic lineage service
func NewSemanticLineageService(
	neo4jRepo *persistence.Neo4jRepository,
	pgRepo *persistence.PostgresRepository,
	findingsProvider interfaces.FindingsProvider,
) *SemanticLineageService {
	return &SemanticLineageService{
		neo4jRepo:        neo4jRepo,
		pgRepo:           pgRepo,
		findingsProvider: findingsProvider,
	}
}

// IsAvailable returns true if Neo4j is configured
// Implements LineageSync interface
func (s *SemanticLineageService) IsAvailable() bool {
	return s.neo4jRepo != nil
}

// SyncAllAssets triggers full lineage synchronization
// Implements LineageSync interface
func (s *SemanticLineageService) SyncAllAssets(ctx context.Context) error {
	return s.SyncLineage(ctx)
}

// SemanticNode represents a node in the semantic graph
type SemanticNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // system, asset, data_category, finding
	Label    string                 `json:"label"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SemanticEdge represents a relationship in the semantic graph
type SemanticEdge struct {
	ID       string                 `json:"id"`
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Type     string                 `json:"type"` // SYSTEM_OWNS_ASSET, EXPOSES
	Metadata map[string]interface{} `json:"metadata"`
}

// SemanticGraph represents the aggregated graph
type SemanticGraph struct {
	Nodes []SemanticNode `json:"nodes"`
	Edges []SemanticEdge `json:"edges"`
}

// SyncAssetToNeo4j syncs an asset and its findings to Neo4j (3-level hierarchy - Frozen Semantic Contract)
// Creates: System → Asset → PII_Category (specific PII types like IN_AADHAAR, CREDIT_CARD)
// NO DataCategory abstraction layer - direct mapping to PII types
func (s *SemanticLineageService) SyncAssetToNeo4j(ctx context.Context, assetID uuid.UUID) error {
	log.Printf("[SYNC] Starting SyncAssetToNeo4j for asset: %s", assetID)

	// Skip if Neo4j is not available
	if s.neo4jRepo == nil {
		log.Printf("[SYNC] Neo4j repository not configured - skipping sync for asset: %s", assetID)
		return nil
	}

	// Get asset from PostgreSQL
	asset, err := s.pgRepo.GetAssetByID(ctx, assetID)
	if err != nil {
		log.Printf("❌ [SYNC] Failed to get asset %s from PostgreSQL: %v\n", assetID, err)
		return fmt.Errorf("failed to get asset: %w", err)
	}
	log.Printf("✅ [SYNC] Retrieved asset from PostgreSQL: %s (Host: %s, Path: %s)\n",
		asset.Name, asset.Host, asset.Path)

	// 1. Create/Update System node
	systemID := fmt.Sprintf("system-%s", asset.Host)
	systemMetadata := map[string]interface{}{
		"host":          asset.Host,
		"source_system": asset.SourceSystem,
		"environment":   asset.Environment,
	}
	if err := s.neo4jRepo.CreateSystemNode(ctx, systemID, asset.Host, systemMetadata); err != nil {
		log.Printf("❌ [SYNC] Failed to create System node: %s - %v\n", systemID, err)
		return fmt.Errorf("failed to create system node: %w", err)
	}
	log.Printf("✅ [SYNC] Created/Updated System node: %s\n", systemID)

	// 2. Create/Update Asset node
	if err := s.neo4jRepo.CreateAssetNode(ctx, asset); err != nil {
		log.Printf("❌ [SYNC] Failed to create Asset node: %s - %v\n", asset.ID, err)
		return fmt.Errorf("failed to create asset node: %w", err)
	}
	log.Printf("✅ [SYNC] Created/Updated Asset node: %s\n", asset.ID)

	// 3. Create SYSTEM_OWNS_ASSET relationship (Frozen Semantic Contract)
	if err := s.neo4jRepo.CreateHierarchyRelationship(ctx, systemID, asset.ID.String(), "SYSTEM_OWNS_ASSET"); err != nil {
		log.Printf("❌ [SYNC] Failed to create SYSTEM_OWNS_ASSET relationship: %s → %s - %v\n",
			systemID, asset.ID, err)
		return fmt.Errorf("failed to create system-asset relationship: %w", err)
	}
	log.Printf("✅ [SYNC] Created SYSTEM_OWNS_ASSET: %s → %s\n", systemID, asset.ID)

	// 4. Get findings for this asset using FindingsProvider
	findings, err := s.findingsProvider.GetFindingsByAsset(ctx, assetID, 1000, 0)
	if err != nil {
		log.Printf("❌ [SYNC] Failed to get findings for asset %s: %v\n", assetID, err)
		return fmt.Errorf("failed to get findings: %w", err)
	}
	log.Printf("📊 [SYNC] Retrieved %d findings from FindingsProvider for asset: %s\n", len(findings), assetID)

	// 5. Aggregate findings by PII TYPE (not classification type)
	// Frozen Semantic Contract: PII_Category = specific PII types (IN_AADHAAR, CREDIT_CARD, etc.)
	piiCategoryMap := make(map[string]*PIICategoryAggregate)
	skippedCount := 0
	lowConfidenceCount := 0
	missingPIITypeCount := 0

	for _, finding := range findings {
		// Get classification using FindingsProvider
		classifications, err := s.findingsProvider.GetClassificationsByFinding(ctx, finding.ID)
		if err != nil || len(classifications) == 0 {
			skippedCount++
			continue
		}

		classification := classifications[0]

		// Filter low-confidence findings (configurable via LINEAGE_CONFIDENCE_THRESHOLD)
		if classification.ConfidenceScore < getConfidenceThreshold() {
			lowConfidenceCount++
			continue
		}

		// Extract PII type from SubCategory (e.g., "IN_AADHAAR", "CREDIT_CARD")
		piiType := classification.SubCategory
		if piiType == "" {
			missingPIITypeCount++
			continue
		}

		if _, exists := piiCategoryMap[piiType]; !exists {
			piiCategoryMap[piiType] = &PIICategoryAggregate{
				PIIType:         piiType,
				DPDPACategory:   classification.DPDPACategory,
				RequiresConsent: classification.RequiresConsent,
				ConfidenceScore: classification.ConfidenceScore,
			}
		}

		agg := piiCategoryMap[piiType]
		// Track highest individual confidence seen for this PII type. Previously
		// a "MMR dedup" step skipped lower-confidence findings here, which made
		// FindingCount undercount (e.g. asset reported 7 PANs, lineage showed 1).
		// FindingCount must reflect every finding row; confidence averaging
		// already handles low-quality findings via TotalConfidence / FindingCount.
		if classification.ConfidenceScore > agg.ConfidenceScore {
			agg.ConfidenceScore = classification.ConfidenceScore
		}
		agg.FindingCount++
		agg.TotalConfidence += classification.ConfidenceScore

		findingAgg := FindingAggregate{
			PatternName: finding.PatternName,
			Severity:    finding.Severity,
			Count:       len(finding.Matches),
		}
		agg.Findings = append(agg.Findings, findingAgg)
	}

	log.Printf("📊 [SYNC] Aggregation Summary:\n")
	log.Printf("   - Total findings processed: %d\n", len(findings))
	log.Printf("   - Unique PII types found: %d\n", len(piiCategoryMap))
	log.Printf("   - Skipped (no classification): %d\n", skippedCount)
	log.Printf("   - Skipped (low confidence <0.45): %d\n", lowConfidenceCount)
	log.Printf("   - Skipped (missing PII type): %d\n", missingPIITypeCount)

	// 6. Create PII_Category nodes (3-level hierarchy - Frozen Semantic Contract)
	// Each PII_Category represents a specific PII type (IN_AADHAAR, CREDIT_CARD, etc.)
	piiNodesCreated := 0
	for piiType, agg := range piiCategoryMap {
		avgConfidence := agg.TotalConfidence / float64(agg.FindingCount)

		// Aggregate pattern statistics for metadata
		patternCounts := make(map[string]int)
		severityCounts := make(map[string]int)
		for _, findingAgg := range agg.Findings {
			patternCounts[findingAgg.PatternName] += findingAgg.Count
			severityCounts[findingAgg.Severity]++
		}

		// Determine risk level based on PII type and confidence
		riskLevel := getRiskLevelForPIIType(piiType, avgConfidence)

		piiCategoryMetadata := map[string]interface{}{
			"pii_type":           piiType,
			"dpdpa_category":     agg.DPDPACategory,
			"requires_consent":   agg.RequiresConsent,
			"finding_count":      agg.FindingCount,
			"avg_confidence":     avgConfidence,
			"risk_level":         riskLevel,
			"pattern_diversity":  len(patternCounts),
			"pattern_counts":     patternCounts,
			"severity_breakdown": severityCounts,
		}

		// Create PII_Category node in Neo4j — continue on failure to avoid aborting entire sync
		if err := s.neo4jRepo.CreatePIICategoryNode(ctx, piiType, piiCategoryMetadata); err != nil {
			log.Printf("[SYNC] Failed to create PII_Category node %s, continuing: %v", piiType, err)
			continue
		}

		// Create EXPOSES relationship with temporal properties (Immutable Lineage)
		if err := s.neo4jRepo.CreateTemporalExposesRelationship(ctx, asset.ID.String(), piiType, agg.FindingCount, avgConfidence); err != nil {
			log.Printf("[SYNC] Failed to create EXPOSES relationship %s → %s, continuing: %v",
				asset.ID, piiType, err)
			continue
		}

		log.Printf("[SYNC] Created PII_Category: %s (Count: %d, Avg Confidence: %.2f, Risk: %s)",
			piiType, agg.FindingCount, avgConfidence, riskLevel)
		piiNodesCreated++
	}

	log.Printf("🎉 [SYNC] Successfully synced asset %s to Neo4j:\n", assetID)
	log.Printf("   - System node: %s\n", systemID)
	log.Printf("   - Asset node: %s\n", asset.ID)
	log.Printf("   - PII_Category nodes: %d\n", piiNodesCreated)
	log.Printf("   - Total relationships: %d (1 SYSTEM_OWNS_ASSET + %d EXPOSES)\n",
		1+piiNodesCreated, piiNodesCreated)

	return nil
}

// getRiskLevelForPIIType determines risk level based on specific PII type and confidence
// Frozen Semantic Contract: Risk is based on the PII type itself, not abstracted classification
func getRiskLevelForPIIType(piiType string, avgConfidence float64) string {
	// Base risk by PII type (India-specific PII is critical)
	baseRisk := map[string]int{
		"IN_AADHAAR":         3, // Critical - Government ID
		"IN_PAN":             3, // Critical - Financial ID
		"IN_PASSPORT":        3, // Critical - Government ID
		"CREDIT_CARD":        3, // Critical - Financial Data
		"IN_BANK_ACCOUNT":    3, // Critical - Financial Data
		"IN_DRIVING_LICENSE": 2, // High - Government ID
		"IN_VOTER_ID":        2, // High - Government ID
		"IN_UPI":             2, // High - Financial
		"IN_IFSC":            1, // Medium - Institutional
		"IN_PHONE":           2, // High - Personal Contact
		"EMAIL_ADDRESS":      2, // High - Personal Contact
	}

	risk, exists := baseRisk[piiType]
	if !exists {
		risk = 1 // Medium for unknown types
	}

	// Adjust based on confidence
	if avgConfidence < 0.65 {
		risk-- // Lower risk if low confidence
	} else if avgConfidence > 0.85 {
		risk++ // Higher risk if very confident
	}

	// Map to risk level strings
	switch {
	case risk >= 3:
		return "Critical"
	case risk == 2:
		return "High"
	case risk == 1:
		return "Medium"
	default:
		return "Low"
	}
}

// PIICategoryAggregate represents aggregated findings by specific PII type
// Frozen Semantic Contract: Aggregate by PII type (IN_AADHAAR, CREDIT_CARD), not classification type
type PIICategoryAggregate struct {
	PIIType         string
	DPDPACategory   string
	RequiresConsent bool
	FindingCount    int
	TotalConfidence float64
	ConfidenceScore float64 // Highest individual recorded confidence
	Findings        []FindingAggregate
}

// FindingAggregate represents aggregated findings by pattern
type FindingAggregate struct {
	PatternName string
	Severity    string
	Count       int
}

// GetSemanticGraph retrieves the semantic lineage graph
// Uses ONLY Neo4j with 3-level frozen hierarchy: System → Asset → PII_Category
func (s *SemanticLineageService) GetSemanticGraph(ctx context.Context, filters SemanticGraphFilters) (*SemanticGraph, error) {
	// Neo4j is MANDATORY - no PostgreSQL fallback
	if s.neo4jRepo == nil {
		return nil, fmt.Errorf("neo4j repository not configured - semantic lineage unavailable")
	}

	// Get graph from Neo4j (3-level hierarchy ONLY)
	// Note: neo4jRepo expects separate string params, not a struct
	nodes, edges, err := s.neo4jRepo.GetSemanticGraph(ctx, filters.SystemID, filters.RiskLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to get semantic graph from neo4j: %w", err)
	}

	// Convert Neo4j types to semantic types
	semanticNodes := []SemanticNode{}
	semanticEdges := []SemanticEdge{}

	for _, node := range nodes {
		semanticNodes = append(semanticNodes, SemanticNode{
			ID:       node.ID,
			Type:     node.Type,
			Label:    node.Label,
			Metadata: node.Metadata,
		})
	}

	for _, edge := range edges {
		semanticEdges = append(semanticEdges, SemanticEdge{
			ID:       edge.ID,
			Source:   edge.Source,
			Target:   edge.Target,
			Type:     edge.Type,
			Metadata: edge.Metadata,
		})
	}

	return &SemanticGraph{
		Nodes: semanticNodes,
		Edges: semanticEdges,
	}, nil
}

// SemanticGraphFilters contains filtering options
type SemanticGraphFilters struct {
	SystemID  string
	RiskLevel string // high, medium, low
	Category  string // PII category filter
}

// SyncLineage triggers a full synchronization of all assets to Neo4j.
// Uses paginated batches of 500 to handle arbitrarily large asset counts.
func (s *SemanticLineageService) SyncLineage(ctx context.Context) error {
	log.Printf("[FULL-SYNC] Starting full lineage synchronization...")

	if s.neo4jRepo == nil {
		return fmt.Errorf("neo4j repository not configured")
	}

	const batchSize = 500
	offset := 0
	totalSynced := 0
	totalErrors := 0

	for {
		assets, err := s.pgRepo.ListAssets(ctx, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to list assets (offset %d): %w", offset, err)
		}

		if len(assets) == 0 {
			break
		}

		log.Printf("[FULL-SYNC] Processing batch of %d assets (offset %d)", len(assets), offset)

		for _, asset := range assets {
			if err := s.SyncAssetToNeo4j(ctx, asset.ID); err != nil {
				log.Printf("[FULL-SYNC] Error syncing asset %s: %v", asset.Name, err)
				totalErrors++
			} else {
				totalSynced++
			}
		}

		offset += len(assets)
		if len(assets) < batchSize {
			break // Last batch
		}
	}

	log.Printf("[FULL-SYNC] Sync completed: %d assets synced, %d failed", totalSynced, totalErrors)
	return nil
}

// DetectLineageCycles runs a Neo4j Cypher query to find cycles in the EXPOSES
// relationship graph. A cycle would mean an asset indirectly exposes itself —
// which indicates a data-flow loop or a Neo4j sync bug.
//
// Phase 5 requirement: flag cycles with risk +10 in caller.
func (s *SemanticLineageService) DetectLineageCycles(ctx context.Context) ([][]string, error) {
	if s.neo4jRepo == nil {
		return nil, nil
	}

	// Cypher: find paths where an Asset node reaches itself via EXPOSES edges.
	// Limit to 20 to avoid overwhelming the graph for large installations.
	query := `
		MATCH path = (a:Asset)-[:EXPOSES*2..10]->(b)
		WHERE a = b
		RETURN [node IN nodes(path) | COALESCE(node.id, node.name, toString(id(node)))] AS cycle_ids
		LIMIT 20
	`
	session := s.neo4jRepo.GetDriver().NewSession(ctx, neo4jDriver.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("cycle detection query: %w", err)
	}

	var cycles [][]string
	for result.Next(ctx) {
		record := result.Record()
		raw, ok := record.Get("cycle_ids")
		if !ok {
			continue
		}
		ids, ok := raw.([]interface{})
		if !ok {
			continue
		}
		var cycle []string
		for _, id := range ids {
			cycle = append(cycle, fmt.Sprintf("%v", id))
		}
		if len(cycle) > 0 {
			cycles = append(cycles, cycle)
		}
	}
	return cycles, result.Err()
}

// TagOrphanAssets marks assets in PostgreSQL that have had no scan findings
// for more than staleAfterDays days. Orphan assets get a +10 risk bonus
// (applied by the caller when scoring). Returns the count of newly tagged assets.
//
// Phase 5 requirement.
func (s *SemanticLineageService) TagOrphanAssets(ctx context.Context, staleAfterDays int) (int, error) {
	if staleAfterDays <= 0 {
		staleAfterDays = 90
	}

	// An asset is an orphan if its most recent finding was created more than
	// staleAfterDays ago OR it has never had a finding.
	res, err := s.pgRepo.GetDB().ExecContext(ctx, `
		UPDATE assets
		   SET is_orphan        = TRUE,
		       orphan_since     = NOW(),
		       orphan_risk_bonus = 10,
		       updated_at       = NOW()
		 WHERE (
		     -- No findings ever
		     id NOT IN (SELECT DISTINCT asset_id FROM findings WHERE asset_id IS NOT NULL)
		     OR
		     -- Last finding older than threshold
		     id IN (
		         SELECT asset_id
		           FROM findings
		       GROUP BY asset_id
		         HAVING MAX(created_at) < NOW() - ($1 || ' days')::INTERVAL
		     )
		 )
		   AND is_orphan = FALSE
	`, staleAfterDays)
	if err != nil {
		return 0, fmt.Errorf("tag orphan assets: %w", err)
	}
	n, _ := res.RowsAffected()

	// Clear orphan flag for assets that have been scanned recently
	_, err = s.pgRepo.GetDB().ExecContext(ctx, `
		UPDATE assets
		   SET is_orphan         = FALSE,
		       orphan_since      = NULL,
		       orphan_risk_bonus = 0,
		       updated_at        = NOW()
		 WHERE is_orphan = TRUE
		   AND id IN (
		       SELECT asset_id
		         FROM findings
		     GROUP BY asset_id
		       HAVING MAX(created_at) >= NOW() - ($1 || ' days')::INTERVAL
		   )
	`, staleAfterDays)
	if err != nil {
		log.Printf("[LINEAGE] orphan clear-flag error (non-fatal): %v", err)
	}

	if n > 0 {
		log.Printf("[LINEAGE] Tagged %d new orphan assets (stale after %d days)", n, staleAfterDays)
	}
	return int(n), nil
}
