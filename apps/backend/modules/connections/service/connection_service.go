package service

import (
	"context"
	"fmt"
	"log"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/infrastructure/vault"
	"github.com/google/uuid"
)

// ConnectionService manages data source connections
type ConnectionService struct {
	pgRepo     *persistence.PostgresRepository
	encryption *encryption.EncryptionService
	vault      *vault.Client
}

// NewConnectionService creates a new connection service.
// vaultClient may be nil when Vault is not configured.
func NewConnectionService(pgRepo *persistence.PostgresRepository, enc *encryption.EncryptionService, vaultClient *vault.Client) *ConnectionService {
	return &ConnectionService{
		pgRepo:     pgRepo,
		encryption: enc,
		vault:      vaultClient,
	}
}

// AddConnection creates a new connection with encrypted credentials.
// When Vault is enabled, credentials are stored ONLY in Vault KV v2.
// PostgreSQL stores only the connection metadata (no encrypted config).
// When Vault is disabled, credentials are AES-256 encrypted in PostgreSQL.
func (s *ConnectionService) AddConnection(ctx context.Context, sourceType, profileName string, config map[string]interface{}, createdBy string) (*entity.Connection, error) {
	var configEncrypted []byte

	if s.vault != nil && s.vault.IsEnabled() {
		// Vault-only: store credentials exclusively in Vault
		if err := s.vault.WriteConnectionSecret(sourceType, profileName, config); err != nil {
			return nil, fmt.Errorf("vault write failed for %s/%s: %w", sourceType, profileName, err)
		}
		log.Printf("INFO: Credentials for %s/%s stored in Vault (not in PostgreSQL)", sourceType, profileName)
		// configEncrypted stays nil — no credentials in PG
	} else {
		// No Vault: encrypt and store in PostgreSQL
		var err error
		configEncrypted, err = s.encryption.Encrypt(config)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize config: %w", err)
		}
	}

	conn := &entity.Connection{
		ID:              uuid.New(),
		SourceType:      sourceType,
		ProfileName:     profileName,
		ConfigEncrypted: configEncrypted,
		CreatedBy:       createdBy,
	}

	if err := s.pgRepo.CreateConnection(ctx, conn); err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// TODO: Trigger async validation (Phase 3 - Temporal workflow)

	return conn, nil
}

// GetConnections retrieves all connections (without decrypted config for security)
func (s *ConnectionService) GetConnections(ctx context.Context) ([]*entity.Connection, error) {
	return s.pgRepo.ListConnections(ctx)
}

// GetConnectionWithConfig retrieves a connection by ID with decrypted config.
// When Vault is enabled, credentials come exclusively from Vault.
// When Vault is disabled, credentials are decrypted from PostgreSQL.
// This should only be used internally, never exposed via API.
func (s *ConnectionService) GetConnectionWithConfig(ctx context.Context, id uuid.UUID) (*entity.Connection, error) {
	conn, err := s.pgRepo.GetConnection(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.vault != nil && s.vault.IsEnabled() {
		config, vErr := s.vault.ReadConnectionSecret(conn.SourceType, conn.ProfileName)
		if vErr != nil {
			return nil, fmt.Errorf("vault read failed for %s/%s: %w", conn.SourceType, conn.ProfileName, vErr)
		}
		if config == nil {
			return nil, fmt.Errorf("credentials not found in Vault for %s/%s", conn.SourceType, conn.ProfileName)
		}
		conn.Config = config
		return conn, nil
	}

	// Vault disabled: decrypt from PostgreSQL
	var config map[string]interface{}
	if err := s.encryption.Decrypt(conn.ConfigEncrypted, &config); err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}
	conn.Config = config

	return conn, nil
}

// GetConnectionByProfile retrieves a connection by source type and profile name
func (s *ConnectionService) GetConnectionByProfile(ctx context.Context, sourceType, profileName string) (*entity.Connection, error) {
	return s.pgRepo.GetConnectionByProfile(ctx, sourceType, profileName)
}

// DeleteConnection deletes a connection by ID.
// Postgres is deleted first; Vault cleanup follows only on success so that a
// failed Postgres delete never leaves the UI pointing at a missing Vault secret.
func (s *ConnectionService) DeleteConnection(ctx context.Context, id uuid.UUID) error {
	// Capture Vault coords before the row disappears.
	var sourceType, profileName string
	if s.vault != nil && s.vault.IsEnabled() {
		if conn, err := s.pgRepo.GetConnection(ctx, id); err == nil {
			sourceType = conn.SourceType
			profileName = conn.ProfileName
		}
	}

	// Delete from Postgres first — if this fails, the Vault secret is still intact.
	if err := s.pgRepo.DeleteConnection(ctx, id); err != nil {
		return err
	}

	// Postgres row is gone; clean up Vault secret. Non-fatal if it fails.
	if s.vault != nil && s.vault.IsEnabled() && sourceType != "" {
		if vErr := s.vault.DeleteConnectionSecret(sourceType, profileName); vErr != nil {
			log.Printf("WARN: Vault delete failed for %s/%s (Postgres row already deleted): %v", sourceType, profileName, vErr)
		}
	}
	return nil
}

// UpdateValidationStatus updates the validation status of a connection
// This will be used by the validation Temporal workflow in Phase 3
func (s *ConnectionService) UpdateValidationStatus(ctx context.Context, id uuid.UUID, status string, validationError *string) error {
	return s.pgRepo.UpdateConnectionValidation(ctx, id, status, validationError)
}
