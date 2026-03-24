package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CreateTemporalExposesRelationship creates a temporal EXPOSES relationship
// This implements the immutable lineage model with exposure windows
func (r *Neo4jRepository) CreateTemporalExposesRelationship(ctx context.Context, assetID, piiType string, findingCount int, avgConfidence float64) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Check if an active EXPOSES edge already exists (until IS NULL)
		checkQuery := `
			MATCH (a:Asset {id: $assetID})-[r:EXPOSES]->(p:PII_Category {pii_type: $piiType})
			WHERE r.until IS NULL
			RETURN r
		`
		checkResult, err := tx.Run(ctx, checkQuery, map[string]interface{}{
			"assetID": assetID,
			"piiType": piiType,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to check existing edge: %w", err)
		}

		// If active edge exists, update its metadata (finding count, confidence)
		if checkResult.Next(ctx) {
			updateQuery := `
				MATCH (a:Asset {id: $assetID})-[r:EXPOSES]->(p:PII_Category {pii_type: $piiType})
				WHERE r.until IS NULL
				SET r.finding_count = $findingCount,
				    r.avg_confidence = $avgConfidence,
				    r.last_updated = datetime()
				RETURN r
			`
			_, err = tx.Run(ctx, updateQuery, map[string]interface{}{
				"assetID":       assetID,
				"piiType":       piiType,
				"findingCount":  findingCount,
				"avgConfidence": avgConfidence,
			})
			return nil, err
		}

		// No active edge exists, create a new one with temporal properties
		createQuery := `
			MATCH (a:Asset {id: $assetID})
			MATCH (p:PII_Category {pii_type: $piiType})
			CREATE (a)-[r:EXPOSES {
				since: datetime(),
				until: null,
				finding_count: $findingCount,
				avg_confidence: $avgConfidence,
				first_detected: datetime(),
				last_updated: datetime()
			}]->(p)
			RETURN r
		`
		_, err = tx.Run(ctx, createQuery, map[string]interface{}{
			"assetID":       assetID,
			"piiType":       piiType,
			"findingCount":  findingCount,
			"avgConfidence": avgConfidence,
		})
		return nil, err
	})

	return err
}

// CloseExposureWindow closes an exposure window by setting the 'until' timestamp
// This is called when PII is no longer detected in an asset
func (r *Neo4jRepository) CloseExposureWindow(ctx context.Context, assetID, piiType string, closedAt time.Time) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (a:Asset {id: $assetID})-[r:EXPOSES]->(p:PII_Category {pii_type: $piiType})
			WHERE r.until IS NULL
			SET r.until = $closedAt
			RETURN r
		`
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"assetID":  assetID,
			"piiType":  piiType,
			"closedAt": closedAt,
		})
		return nil, err
	})

	return err
}
