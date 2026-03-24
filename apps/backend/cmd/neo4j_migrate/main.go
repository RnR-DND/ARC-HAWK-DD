package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/arc-platform/backend/modules/lineage/migrations"
	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get Neo4j connection details
	neo4jURI := getEnv("NEO4J_URI", "bolt://localhost:7687")
	neo4jUser := getEnv("NEO4J_USER", "neo4j")
	neo4jPassword := getEnv("NEO4J_PASSWORD", "password123")

	// Create Neo4j driver
	driver, err := neo4j.NewDriver(neo4jURI, neo4j.BasicAuth(neo4jUser, neo4jPassword, ""))
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer driver.Close()

	// Verify connection
	if err := driver.VerifyConnectivity(); err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}

	log.Println("Connected to Neo4j successfully")

	// Check command line arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	ctx := context.Background()

	switch command {
	case "migrate":
		log.Println("Running temporal graph migration...")
		if err := migrations.MigrateToTemporalGraph(ctx, driver); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migration completed successfully!")

	case "rollback":
		log.Println("Rolling back temporal graph migration...")
		if err := migrations.RollbackTemporalGraph(ctx, driver); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		log.Println("Rollback completed successfully!")

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run run_migration.go [command]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  migrate   - Run the temporal graph migration")
	fmt.Println("  rollback  - Rollback the temporal graph migration")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  NEO4J_URI      - Neo4j connection URI (default: bolt://localhost:7687)")
	fmt.Println("  NEO4J_USER     - Neo4j username (default: neo4j)")
	fmt.Println("  NEO4J_PASSWORD - Neo4j password (default: password123)")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
