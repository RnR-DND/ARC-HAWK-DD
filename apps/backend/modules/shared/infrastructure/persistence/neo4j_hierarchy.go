package persistence

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// === Frozen Semantic Contract: 3-Level Hierarchy ===
// Node Types: System → Asset → PII_Category
// Edge Types: SYSTEM_OWNS_ASSET, EXPOSES
// NO transformation edges - only risk associations

// CreatePIICategoryNode creates or updates a PII_Category node
// PII_Category represents specific PII types (IN_AADHAAR, CREDIT_CARD, etc.)
func (r *Neo4jRepository) CreatePIICategoryNode(ctx context.Context, piiType string, metadata map[string]interface{}) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MERGE (pii:PII_Category {type: $type})
			SET pii.pii_type = $type,
			    pii.dpdpa_category = $dpdpa_category,
			    pii.requires_consent = $requires_consent,
			    pii.finding_count = $finding_count,
			    pii.avg_confidence = $avg_confidence,
			    pii.risk_level = $risk_level,
			    pii.updated_at = datetime()
			RETURN pii
		`
		params := map[string]interface{}{
			"type":             piiType,
			"dpdpa_category":   metadata["dpdpa_category"],
			"requires_consent": metadata["requires_consent"],
			"finding_count":    metadata["finding_count"],
			"avg_confidence":   metadata["avg_confidence"],
			"risk_level":       metadata["risk_level"],
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// CreateHierarchyRelationship creates relationships using frozen semantic contract
// Allowed edge types: SYSTEM_OWNS_ASSET, EXPOSES
func (r *Neo4jRepository) CreateHierarchyRelationship(ctx context.Context, parentID, childID, relType string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		var query string

		switch relType {
		case "SYSTEM_OWNS_ASSET": // System → Asset
			query = `
				MATCH (sys:System {id: $parentID})
				MATCH (asset:Asset {id: $childID})
				MERGE (sys)-[r:SYSTEM_OWNS_ASSET]->(asset)
				SET r.updated_at = datetime()
				RETURN r
			`
		case "EXPOSES": // Asset → PII_Category
			query = `
				MATCH (asset:Asset {id: $parentID})
				MATCH (pii:PII_Category {type: $childID})
				MERGE (asset)-[r:EXPOSES]->(pii)
				SET r.updated_at = datetime()
				RETURN r
			`
		default:
			return nil, fmt.Errorf("unknown relationship type: %s (allowed: SYSTEM_OWNS_ASSET, EXPOSES)", relType)
		}

		params := map[string]interface{}{
			"parentID": parentID,
			"childID":  childID,
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

// GetSemanticGraph retrieves the 3-level hierarchy from Neo4j
func (r *Neo4jRepository) GetSemanticGraph(ctx context.Context, systemFilter, riskFilter string) ([]Node, []Edge, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	nodes := []Node{}
	edges := []Edge{}
	nodeMap := make(map[string]bool)
	edgeMap := make(map[string]bool)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Frozen Semantic Contract: 3-level hierarchy query
		// System → Asset → PII_Category (no intermediate DataCategory)
		query := `
			MATCH (sys:System)
			OPTIONAL MATCH (sys)-[:SYSTEM_OWNS_ASSET]->(asset:Asset)
			OPTIONAL MATCH (asset)-[:EXPOSES]->(pii:PII_Category)
			WHERE ($systemFilter = '' OR sys.host = $systemFilter)
			  AND ($riskFilter = '' OR pii.risk_level IS NULL OR pii.risk_level = $riskFilter)
			RETURN sys, asset, pii
			ORDER BY sys.host, asset.name
			LIMIT 1000
		`
		params := map[string]interface{}{
			"systemFilter": systemFilter,
			"riskFilter":   riskFilter,
		}

		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		records, err := result.Collect(ctx)
		if err != nil {
			return nil, err
		}

		for _, record := range records {
			// Get values from record (3-level hierarchy)
			sysVal, _ := record.Get("sys")
			assetVal, _ := record.Get("asset")
			piiVal, _ := record.Get("pii")

			// Process System node
			if sysVal != nil {
				if node, ok := sysVal.(neo4j.Node); ok {
					id, _ := node.Props["id"].(string)
					host, _ := node.Props["host"].(string)
					// Use host as label for System nodes
					label := host
					if label == "" {
						label = id // Fallback to ID if host is empty
					}
					if id != "" && !nodeMap[id] {
						nodes = append(nodes, Node{
							ID:    id,
							Label: label,
							Type:  "system",
							Metadata: map[string]interface{}{
								"host": host,
							},
						})
						nodeMap[id] = true
					}
				}
			}

			// Process Asset node
			if assetVal != nil {
				if node, ok := assetVal.(neo4j.Node); ok {
					id, _ := node.Props["id"].(string)
					name, _ := node.Props["name"].(string)
					path, _ := node.Props["path"].(string)
					// Use name if available, otherwise path, otherwise ID
					label := name
					if label == "" {
						label = path
					}
					if label == "" {
						label = id
					}
					if id != "" && !nodeMap[id] {
						nodes = append(nodes, Node{
							ID:    id,
							Label: label,
							Type:  "asset",
							Metadata: map[string]interface{}{
								"path":        path,
								"environment": node.Props["environment"],
							},
						})
						nodeMap[id] = true
					}
				}
			}

			// Process PII_Category node (replaces old DataCategory + PIIType)
			if piiVal != nil {
				if node, ok := piiVal.(neo4j.Node); ok {
					piiType, _ := node.Props["type"].(string)
					if piiType != "" && !nodeMap[piiType] {
						nodes = append(nodes, Node{
							ID:    piiType,
							Label: piiType,
							Type:  "pii_category",
							Metadata: map[string]interface{}{
								"pii_type":       piiType,
								"finding_count":  node.Props["finding_count"],
								"risk_level":     node.Props["risk_level"],
								"avg_confidence": node.Props["avg_confidence"],
								"dpdpa_category": node.Props["dpdpa_category"],
							},
						})
						nodeMap[piiType] = true
					}
				}
			}

			// Build edges from 3-level hierarchy
			// System -> Asset (SYSTEM_OWNS_ASSET)
			if sysVal != nil && assetVal != nil {
				if sysNode, ok := sysVal.(neo4j.Node); ok {
					if assetNode, ok := assetVal.(neo4j.Node); ok {
						sysID, _ := sysNode.Props["id"].(string)
						assetID, _ := assetNode.Props["id"].(string)
						if sysID != "" && assetID != "" {
							edgeID := fmt.Sprintf("%s-SYSTEM_OWNS_ASSET-%s", sysID, assetID)
							if !edgeMap[edgeID] {
								edges = append(edges, Edge{
									ID:     edgeID,
									Source: sysID,
									Target: assetID,
									Type:   "SYSTEM_OWNS_ASSET",
									Label:  "owns",
								})
								edgeMap[edgeID] = true
							}
						}
					}
				}
			}

			// Asset -> PII_Category (EXPOSES)
			if assetVal != nil && piiVal != nil {
				if assetNode, ok := assetVal.(neo4j.Node); ok {
					if piiNode, ok := piiVal.(neo4j.Node); ok {
						assetID, _ := assetNode.Props["id"].(string)
						piiType, _ := piiNode.Props["type"].(string)
						if assetID != "" && piiType != "" {
							edgeID := fmt.Sprintf("%s-EXPOSES-%s", assetID, piiType)
							if !edgeMap[edgeID] {
								edges = append(edges, Edge{
									ID:     edgeID,
									Source: assetID,
									Target: piiType,
									Type:   "EXPOSES",
									Label:  "contains",
								})
								edgeMap[edgeID] = true
							}
						}
					}
				}
			}
		}

		return nil, nil
	})

	if err != nil {
		return nil, nil, err
	}

	_ = result

	return nodes, edges, nil
}

// GetPIIAggregations returns aggregated PII type statistics
func (r *Neo4jRepository) GetPIIAggregations(ctx context.Context) ([]map[string]interface{}, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// FROZEN SEMANTIC CONTRACT: 3-level hierarchy only
		// System -[:SYSTEM_OWNS_ASSET]-> Asset -[:EXPOSES]-> PII_Category
		query := `
			MATCH (pii:PII_Category)
			OPTIONAL MATCH (asset:Asset)-[:EXPOSES]->(pii)
			OPTIONAL MATCH (sys:System)-[:SYSTEM_OWNS_ASSET]->(asset)
			RETURN 
			  pii.type as pii_type,
			  pii.finding_count as total_findings,
			  pii.risk_level as risk_level,
			  pii.avg_confidence as confidence,
			  COUNT(DISTINCT asset) as affected_assets,
			  COUNT(DISTINCT sys) as affected_systems
			ORDER BY total_findings DESC
		`

		result, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}

		records, err := result.Collect(ctx)
		if err != nil {
			return nil, err
		}

		aggregations := []map[string]interface{}{}
		for _, record := range records {
			agg := map[string]interface{}{
				"pii_type":         record.Values[0],
				"total_findings":   record.Values[1],
				"risk_level":       record.Values[2],
				"confidence":       record.Values[3],
				"affected_assets":  record.Values[4],
				"affected_systems": record.Values[5],
				// categories removed - not in frozen contract
			}
			aggregations = append(aggregations, agg)
		}

		return aggregations, nil
	})

	if err != nil {
		return nil, err
	}

	if aggs, ok := result.([]map[string]interface{}); ok {
		return aggs, nil
	}

	return []map[string]interface{}{}, nil
}
