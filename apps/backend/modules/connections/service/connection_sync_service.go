package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"gopkg.in/yaml.v3"
)

// ConnectionSyncService syncs database connections to scanner YAML file
type ConnectionSyncService struct {
	repo       *persistence.PostgresRepository
	encryption *encryption.EncryptionService
	yamlPath   string
}

// NewConnectionSyncService creates a new connection sync service
func NewConnectionSyncService(repo *persistence.PostgresRepository, enc *encryption.EncryptionService) *ConnectionSyncService {
	// Determine YAML path
	yamlPath := os.Getenv("SCANNER_CONFIG_PATH")
	if yamlPath == "" {
		// Default to scanner config directory
		workDir := os.Getenv("ARC_HAWK_ROOT")
		if workDir == "" {
			workDir = "."
		}
		yamlPath = filepath.Join(workDir, "apps/scanner/config/connection.yml")
	}

	return &ConnectionSyncService{
		repo:       repo,
		encryption: enc,
		yamlPath:   yamlPath,
	}
}

// ScannerConfig represents the scanner YAML configuration format
type ScannerConfig struct {
	Sources map[string]map[string]interface{} `yaml:"sources"`
}

// SyncToYAML syncs all database connections to the scanner YAML file
func (s *ConnectionSyncService) SyncToYAML(ctx context.Context) error {
	log.Printf("INFO: Starting connection sync to %s", s.yamlPath)

	// Get all connections from database
	connections, err := s.repo.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	if len(connections) == 0 {
		log.Printf("INFO: No connections to sync")
		return nil
	}

	// Build scanner config structure
	scannerConfig := ScannerConfig{
		Sources: make(map[string]map[string]interface{}),
	}

	for _, conn := range connections {
		log.Printf("INFO: Syncing connection SourceType=%s Profile=%s", conn.SourceType, conn.ProfileName)

		var config map[string]interface{}
		err := s.encryption.Decrypt(conn.ConfigEncrypted, &config)
		if err != nil {
			log.Printf("WARN: Decrypt failed for %s/%s: %v", conn.SourceType, conn.ProfileName, err)
			continue
		}

		// C-6: Write only non-secret metadata to YAML; passwords are NOT written to disk
		safeConfig := make(map[string]interface{})
		for k, v := range config {
			switch k {
			case "password", "secret", "access_key", "secret_key", "token", "api_key":
				continue
			default:
				safeConfig[k] = v
			}
		}

		if scannerConfig.Sources[conn.SourceType] == nil {
			scannerConfig.Sources[conn.SourceType] = make(map[string]interface{})
		}
		scannerConfig.Sources[conn.SourceType][conn.ProfileName] = safeConfig
	}
	// Marshal to YAML
	yamlData, err := yaml.Marshal(&scannerConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(s.yamlPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(s.yamlPath, yamlData, 0600); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	log.Printf("SUCCESS: Synced %d connections to %s", len(connections), s.yamlPath)
	return nil
}

// SyncSingleConnection syncs a single connection to YAML
// This is more efficient than full sync when only one connection changes
func (s *ConnectionSyncService) SyncSingleConnection(ctx context.Context, sourceType, profileName string) error {
	// For simplicity, we'll do a full sync
	// In production, you might want to read existing YAML, update one entry, and write back
	return s.SyncToYAML(ctx)
}

// ValidateSync verifies that YAML file matches database state
func (s *ConnectionSyncService) ValidateSync(ctx context.Context) (bool, error) {
	// Read current YAML
	yamlData, err := os.ReadFile(s.yamlPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, needs sync
			return false, nil
		}
		return false, fmt.Errorf("failed to read YAML: %w", err)
	}

	var currentConfig ScannerConfig
	if err := yaml.Unmarshal(yamlData, &currentConfig); err != nil {
		return false, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Get connections from database
	connections, err := s.repo.ListConnections(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list connections: %w", err)
	}

	// Count connections in YAML
	yamlCount := 0
	for _, profiles := range currentConfig.Sources {
		yamlCount += len(profiles)
	}

	// Simple validation: count should match
	if yamlCount != len(connections) {
		log.Printf("INFO: Sync validation failed: YAML has %d connections, DB has %d", yamlCount, len(connections))
		return false, nil
	}

	return true, nil
}
