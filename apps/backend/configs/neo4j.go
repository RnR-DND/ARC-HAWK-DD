package configs

import "os"

// Neo4jConfig holds Neo4j database configuration
type Neo4jConfig struct {
	URI      string
	Username string
	Password string
	Database string
}

// LoadNeo4jConfig loads Neo4j configuration from environment variables
func LoadNeo4jConfig() *Neo4jConfig {
	return &Neo4jConfig{
		URI:      getEnvWithDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username: getEnvWithDefault("NEO4J_USERNAME", "neo4j"),
		Password: getEnvWithDefault("NEO4J_PASSWORD", ""),
		Database: getEnvWithDefault("NEO4J_DATABASE", "neo4j"),
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
