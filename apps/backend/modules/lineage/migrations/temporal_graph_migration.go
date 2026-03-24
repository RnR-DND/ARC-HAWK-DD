package migrations

import (
	"context"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// MigrateToTemporalGraph migrates existing static graph to temporal model
// This adds time-bound properties to edges for exposure window tracking
func MigrateToTemporalGraph(ctx context.Context, driver neo4j.Driver) error {
	session := driver.NewSession(neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close()

	log.Println("Starting temporal graph migration...")

	// Step 1: Add temporal properties to existing ASSET_CONTAINS_PII edges
	log.Println("Step 1: Adding temporal properties to ASSET_CONTAINS_PII edges...")
	_, err := session.Run(`
		MATCH (a:Asset)-[r:ASSET_CONTAINS_PII]->(p:PII_Category)
		WHERE r.since IS NULL
		SET r.since = datetime(),
			r.until = null,
			r.first_scan_id = 'migration',
			r.last_scan_id = 'migration'
		RETURN count(r) as updated_count
	`, nil)
	if err != nil {
		return fmt.Errorf("failed to add temporal properties: %w", err)
	}

	// Step 2: Rename ASSET_CONTAINS_PII to EXPOSES
	log.Println("Step 2: Renaming ASSET_CONTAINS_PII to EXPOSES...")
	result, err := session.Run(`
		MATCH (a:Asset)-[r:ASSET_CONTAINS_PII]->(p:PII_Category)
		WITH a, r, p, properties(r) as props
		CREATE (a)-[r2:EXPOSES]->(p)
		SET r2 = props
		DELETE r
		RETURN count(r2) as renamed_count
	`, nil)
	if err != nil {
		return fmt.Errorf("failed to rename edges: %w", err)
	}

	if result.Next() {
		count := result.Record().Values[0]
		log.Printf("Renamed %v edges from ASSET_CONTAINS_PII to EXPOSES\n", count)
	}

	// Step 3: Create indexes for temporal queries
	log.Println("Step 3: Creating indexes for temporal queries...")
	_, err = session.Run(`
		CREATE INDEX exposes_since IF NOT EXISTS
		FOR ()-[r:EXPOSES]-()
		ON (r.since)
	`, nil)
	if err != nil {
		return fmt.Errorf("failed to create since index: %w", err)
	}

	_, err = session.Run(`
		CREATE INDEX exposes_until IF NOT EXISTS
		FOR ()-[r:EXPOSES]-()
		ON (r.until)
	`, nil)
	if err != nil {
		return fmt.Errorf("failed to create until index: %w", err)
	}

	// Step 4: Add temporal properties to System and Asset nodes
	log.Println("Step 4: Adding created_at to System and Asset nodes...")
	_, err = session.Run(`
		MATCH (s:System)
		WHERE s.created_at IS NULL
		SET s.created_at = datetime()
	`, nil)
	if err != nil {
		return fmt.Errorf("failed to add created_at to System nodes: %w", err)
	}

	_, err = session.Run(`
		MATCH (a:Asset)
		WHERE a.created_at IS NULL
		SET a.created_at = datetime()
	`, nil)
	if err != nil {
		return fmt.Errorf("failed to add created_at to Asset nodes: %w", err)
	}

	log.Println("Temporal graph migration completed successfully!")
	return nil
}

// RollbackTemporalGraph rolls back the temporal graph migration
func RollbackTemporalGraph(ctx context.Context, driver neo4j.Driver) error {
	session := driver.NewSession(neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close()

	log.Println("Rolling back temporal graph migration...")

	// Rename EXPOSES back to ASSET_CONTAINS_PII
	_, err := session.Run(`
		MATCH (a:Asset)-[r:EXPOSES]->(p:PII_Category)
		WITH a, r, p, properties(r) as props
		CREATE (a)-[r2:ASSET_CONTAINS_PII]->(p)
		SET r2 = props
		DELETE r
		RETURN count(r2) as rolled_back_count
	`, nil)
	if err != nil {
		return fmt.Errorf("failed to rollback edges: %w", err)
	}

	// Drop indexes
	_, err = session.Run(`DROP INDEX exposes_since IF EXISTS`, nil)
	if err != nil {
		log.Printf("Warning: failed to drop exposes_since index: %v\n", err)
	}

	_, err = session.Run(`DROP INDEX exposes_until IF EXISTS`, nil)
	if err != nil {
		log.Printf("Warning: failed to drop exposes_until index: %v\n", err)
	}

	log.Println("Temporal graph migration rolled back successfully!")
	return nil
}
