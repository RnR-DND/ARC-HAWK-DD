package service

import (
	"context"
	"fmt"
	"log"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// ConnectionService manages data source connections
type ConnectionService struct {
	pgRepo     *persistence.PostgresRepository
	encryption *encryption.EncryptionService
}

// NewConnectionService creates a new connection service
func NewConnectionService(pgRepo *persistence.PostgresRepository, enc *encryption.EncryptionService) *ConnectionService {
	return &ConnectionService{
		pgRepo:     pgRepo,
		encryption: enc,
	}
}

// AddConnection creates a new connection with encrypted credentials
func (s *ConnectionService) AddConnection(ctx context.Context, sourceType, profileName string, config map[string]interface{}, createdBy string) (*entity.Connection, error) {
	// 1. Encrypt config
	log.Printf("DEBUG CONFIG INPUT: %+v", config)

	configEncrypted, err := s.encryption.Encrypt(config)
        if err != nil {
                return nil, fmt.Errorf("failed to serialize config: %w", err)
        }
	// 2. Create connection entity
	conn := &entity.Connection{
		ID:              uuid.New(),
		SourceType:      sourceType,
		ProfileName:     profileName,
		ConfigEncrypted: configEncrypted,
		CreatedBy:       createdBy,
	}

	// 3. Store in database
	if err := s.pgRepo.CreateConnection(ctx, conn); err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// 4. TODO: Trigger async validation (Phase 3 - Temporal workflow)

	return conn, nil
}

// GetConnections retrieves all connections (without decrypted config for security)
func (s *ConnectionService) GetConnections(ctx context.Context) ([]*entity.Connection, error) {
	return s.pgRepo.ListConnections(ctx)
}

// GetConnectionWithConfig retrieves a connection by ID with decrypted config
// This should only be used internally, never exposed via API
func (s *ConnectionService) GetConnectionWithConfig(ctx context.Context, id uuid.UUID) (*entity.Connection, error) {
	conn, err := s.pgRepo.GetConnection(ctx, id)
	if err != nil {
		return nil, err
	}

	// Decrypt config
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

// DeleteConnection deletes a connection by ID
func (s *ConnectionService) DeleteConnection(ctx context.Context, id uuid.UUID) error {
	return s.pgRepo.DeleteConnection(ctx, id)
}

// UpdateValidationStatus updates the validation status of a connection
// This will be used by the validation Temporal workflow in Phase 3
func (s *ConnectionService) UpdateValidationStatus(ctx context.Context, id uuid.UUID, status string, validationError *string) error {
	return s.pgRepo.UpdateConnectionValidation(ctx, id, status, validationError)
}
