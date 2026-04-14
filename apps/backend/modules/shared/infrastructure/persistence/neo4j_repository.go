package persistence

import (
	"context"
	"fmt"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Node represents a graph node
type Node struct {
	ID        string                 `json:"id"`
	Label     string                 `json:"label"`
	Type      string                 `json:"type"`
	ParentID  string                 `json:"parent_id,omitempty"`
	RiskScore int                    `json:"risk_score"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Edge represents a graph edge
type Edge struct {
	ID       string                 `json:"id"`
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Type     string                 `json:"type"`
	Label    string                 `json:"label"`
	Metadata map[string]interface{} `json:"metadata"`
}

// LineageGraph represents the complete graph
type LineageGraph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Neo4jRepository handles all Neo4j graph database operations
type Neo4jRepository struct {
	driver neo4j.DriverWithContext
}

// NewNeo4jRepository creates a new Neo4j repository
func NewNeo4jRepository(uri, username, password string) (*Neo4jRepository, error) {
	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.BasicAuth(username, password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	ctx := context.Background()
	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return &Neo4jRepository{
		driver: driver,
	}, nil
}

// GetDriver returns the underlying Neo4j driver
// This is used by Temporal worker to access the driver
func (r *Neo4jRepository) GetDriver() neo4j.DriverWithContext {
	return r.driver
}

// Close closes the Neo4j driver
func (r *Neo4jRepository) Close(ctx context.Context) error {
	return r.driver.Close(ctx)
}

// Ping verifies that the Neo4j server is reachable and responsive.
// L3: Used by the health check endpoint to detect circuit-breaker conditions.
func (r *Neo4jRepository) Ping(ctx context.Context) error {
	if r.driver == nil {
		return fmt.Errorf("neo4j driver not initialized")
	}
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.Run(ctx, "RETURN 1", nil)
	if err != nil {
		return fmt.Errorf("neo4j ping failed: %w", err)
	}
	return nil
}

// === Node Creation Methods ===

// CreateSystemNode creates or updates a system node in Neo4j
func (r *Neo4jRepository) CreateSystemNode(ctx context.Context, systemID, label string, metadata map[string]interface{}) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MERGE (s:System {id: $systemID})
			SET s.label = $label,
			    s.host = $host,
			    s.source_system = $sourceSystem,
			    s.updated_at = datetime()
			RETURN s
		`
		params := map[string]interface{}{
			"systemID":     systemID,
			"label":        label,
			"host":         metadata["host"],
			"sourceSystem": metadata["source_system"],
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// CreateAssetNode creates or updates an asset node in Neo4j
func (r *Neo4jRepository) CreateAssetNode(ctx context.Context, asset *entity.Asset) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MERGE (a:Asset {id: $id})
			SET a.name = $name,
			    a.asset_type = $assetType,
			    a.path = $path,
			    a.data_source = $dataSource,
			    a.host = $host,
			    a.environment = $environment,
			    a.owner = $owner,
			    a.source_system = $sourceSystem,
			    a.risk_score = $riskScore,
			    a.total_findings = $totalFindings,
			    a.parent_id = $parentID,
			    a.updated_at = datetime()
			RETURN a
		`
		// M11: parentID is derived from the asset's SourceSystem (system node ID).
		// The system node ID is set to the SourceSystem value when CreateSystemNode is called.
		parentID := asset.SourceSystem
		params := map[string]interface{}{
			"id":            asset.ID.String(),
			"name":          asset.Name,
			"assetType":     asset.AssetType,
			"path":          asset.Path,
			"dataSource":    asset.DataSource,
			"host":          asset.Host,
			"environment":   asset.Environment,
			"owner":         asset.Owner,
			"sourceSystem":  asset.SourceSystem,
			"riskScore":     asset.RiskScore,
			"totalFindings": asset.TotalFindings,
			"parentID":      parentID,
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// CreateFindingNode creates or updates a finding node in Neo4j
func (r *Neo4jRepository) CreateFindingNode(ctx context.Context, finding *entity.Finding, classification *entity.Classification) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MERGE (f:Finding {id: $id})
			SET f.pattern_name = $patternName,
			    f.severity = $severity,
			    f.matches_count = $matchesCount,
			    f.classification = $classification,
			    f.confidence = $confidence,
			    f.risk_score = $riskScore,
			    f.updated_at = datetime()
			RETURN f
		`

		classificationType := "Unknown"
		confidence := 0.0
		if classification != nil {
			classificationType = classification.ClassificationType
			confidence = classification.ConfidenceScore
		}

		riskScore := calculateFindingRiskScore(finding.Severity)

		params := map[string]interface{}{
			"id":             finding.ID.String(),
			"patternName":    finding.PatternName,
			"severity":       finding.Severity,
			"matchesCount":   len(finding.Matches),
			"classification": classificationType,
			"confidence":     confidence,
			"riskScore":      riskScore,
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// CreateClassificationNode creates or updates a classification node in Neo4j
func (r *Neo4jRepository) CreateClassificationNode(ctx context.Context, classification *entity.Classification) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MERGE (c:Classification {type: $type})
			SET c.dpdpa_category = $dpdpaCategory,
			    c.requires_consent = $requiresConsent,
			    c.updated_at = datetime()
			RETURN c
		`
		params := map[string]interface{}{
			"type":            classification.ClassificationType,
			"dpdpaCategory":   classification.DPDPACategory,
			"requiresConsent": classification.RequiresConsent,
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// === Relationship Creation Methods ===

// CreateExposesRelationship creates an EXPOSES relationship (Asset -> Finding)
func (r *Neo4jRepository) CreateExposesRelationship(ctx context.Context, assetID, findingID string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (a:Asset {id: $assetID})
			MATCH (f:Finding {id: $findingID})
			MERGE (a)-[r:EXPOSES]->(f)
			RETURN r
		`
		params := map[string]interface{}{
			"assetID":   assetID,
			"findingID": findingID,
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// CreateClassifiedAsRelationship creates a CLASSIFIED_AS relationship (Finding -> Classification)
func (r *Neo4jRepository) CreateClassifiedAsRelationship(ctx context.Context, findingID, classificationType string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (f:Finding {id: $findingID})
			MATCH (c:Classification {type: $classificationType})
			MERGE (f)-[r:CLASSIFIED_AS]->(c)
			RETURN r
		`
		params := map[string]interface{}{
			"findingID":          findingID,
			"classificationType": classificationType,
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// === Query Methods ===

// GetLineageGraph retrieves the complete lineage graph from Neo4j
func (r *Neo4jRepository) GetLineageGraph(ctx context.Context) (*LineageGraph, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	nodes := []Node{}
	edges := []Edge{}
	nodeMap := make(map[string]bool)
	edgeMap := make(map[string]bool)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Query to get all nodes and relationships
		query := `
			MATCH (s:System)
			OPTIONAL MATCH (s)-[r1:CONTAINS]->(a:Asset)
			OPTIONAL MATCH (a)-[r2:EXPOSES]->(f:Finding)
			OPTIONAL MATCH (f)-[r3:CLASSIFIED_AS]->(c:Classification)
			WHERE c.type IS NULL OR c.type <> 'Non-PII'
			RETURN s, a, f, c, r1, r2, r3
		`

		result, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}

		records, err := result.Collect(ctx)
		if err != nil {
			return nil, err
		}

		for _, record := range records {
			// Process System node
			if systemNode, ok := record.Get("s"); ok && systemNode != nil {
				if node, ok := systemNode.(neo4j.Node); ok {
					id, _ := node.Props["id"].(string)
					label, _ := node.Props["label"].(string)
					if id != "" && !nodeMap[id] {
						nodes = append(nodes, Node{
							ID:        id,
							Label:     label,
							Type:      "system",
							RiskScore: 0,
							Metadata: map[string]interface{}{
								"host":          node.Props["host"],
								"source_system": node.Props["source_system"],
							},
						})
						nodeMap[id] = true
					}
				}
			}

			// Process Asset node
			if assetNode, ok := record.Get("a"); ok && assetNode != nil {
				if node, ok := assetNode.(neo4j.Node); ok {
					id, _ := node.Props["id"].(string)
					name, _ := node.Props["name"].(string)
					assetType, _ := node.Props["asset_type"].(string)
					riskScore, _ := node.Props["risk_score"].(int64)
					// M11: populate ParentID from system node or stored parent_id
					parentID, _ := node.Props["parent_id"].(string)
					if parentID == "" {
						// Derive from system node in this record
						if sysNode, ok := record.Get("s"); ok && sysNode != nil {
							if sNode, ok := sysNode.(neo4j.Node); ok {
								parentID, _ = sNode.Props["id"].(string)
							}
						}
					}
					if id != "" && !nodeMap[id] {
						nodes = append(nodes, Node{
							ID:        id,
							Label:     name,
							Type:      assetType,
							ParentID:  parentID,
							RiskScore: int(riskScore),
							Metadata: map[string]interface{}{
								"path":           node.Props["path"],
								"data_source":    node.Props["data_source"],
								"environment":    node.Props["environment"],
								"owner":          node.Props["owner"],
								"total_findings": node.Props["total_findings"],
							},
						})
						nodeMap[id] = true
					}
				}
			}

			// Process Finding node
			if findingNode, ok := record.Get("f"); ok && findingNode != nil {
				if node, ok := findingNode.(neo4j.Node); ok {
					id, _ := node.Props["id"].(string)
					patternName, _ := node.Props["pattern_name"].(string)
					riskScore, _ := node.Props["risk_score"].(int64)
					if id != "" && !nodeMap[id] {
						nodes = append(nodes, Node{
							ID:        id,
							Label:     patternName,
							Type:      "finding",
							RiskScore: int(riskScore),
							Metadata: map[string]interface{}{
								"severity":       node.Props["severity"],
								"matches_count":  node.Props["matches_count"],
								"classification": node.Props["classification"],
								"confidence":     node.Props["confidence"],
							},
						})
						nodeMap[id] = true
					}
				}
			}

			// Process Classification node
			if classNode, ok := record.Get("c"); ok && classNode != nil {
				if node, ok := classNode.(neo4j.Node); ok {
					classType, _ := node.Props["type"].(string)
					if classType != "" && !nodeMap[classType] {
						nodes = append(nodes, Node{
							ID:        classType,
							Label:     classType,
							Type:      "classification",
							RiskScore: 0,
							Metadata: map[string]interface{}{
								"dpdpa_category":   node.Props["dpdpa_category"],
								"requires_consent": node.Props["requires_consent"],
							},
						})
						nodeMap[classType] = true
					}
				}
			}

			// Process relationships
			processRelationship := func(relKey string, relType string) {
				if rel, ok := record.Get(relKey); ok && rel != nil {
					if relationship, ok := rel.(neo4j.Relationship); ok {
						edgeID := fmt.Sprintf("%s-%s-%s", relationship.StartElementId, relType, relationship.EndElementId)
						if !edgeMap[edgeID] {
							// Simply create edges - detailed mapping would require tracking element IDs
							edges = append(edges, Edge{
								ID:     edgeID,
								Source: "", // Would need element ID to node ID mapping
								Target: "",
								Type:   relType,
								Label:  relType,
							})
							edgeMap[edgeID] = true
						}
					}
				}
			}

			processRelationship("r1", "CONTAINS")
			processRelationship("r2", "EXPOSES")
			processRelationship("r3", "CLASSIFIED_AS")
		}

		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	_ = result // result is nil but returned for interface compatibility

	return &LineageGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// Helper function
func calculateFindingRiskScore(severity string) int {
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
