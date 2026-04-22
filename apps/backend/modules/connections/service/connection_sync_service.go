package service

import (
	"context"
	"log"

	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

// ConnectionSyncService was previously responsible for syncing database connections
// to a YAML file consumed by the Python scanner. The Python scanner has been removed;
// the Go scanner reads connection details directly from the database at scan time.
// This service is kept as a stub to avoid breaking callers until they are updated.
type ConnectionSyncService struct {
	repo       *persistence.PostgresRepository
	encryption *encryption.EncryptionService
}

// NewConnectionSyncService creates a new (no-op) connection sync service.
func NewConnectionSyncService(repo *persistence.PostgresRepository, enc *encryption.EncryptionService) *ConnectionSyncService {
	return &ConnectionSyncService{
		repo:       repo,
		encryption: enc,
	}
}

// SyncToYAML is a no-op. The Go scanner reads connections from the database directly.
func (s *ConnectionSyncService) SyncToYAML(ctx context.Context) error {
	log.Printf("INFO: ConnectionSyncService.SyncToYAML called — no-op (Go scanner reads from DB)")
	return nil
}

// SyncSingleConnection is a no-op. The Go scanner reads connections from the database directly.
func (s *ConnectionSyncService) SyncSingleConnection(ctx context.Context, sourceType, profileName string) error {
	log.Printf("INFO: ConnectionSyncService.SyncSingleConnection called — no-op (source=%s profile=%s; Go scanner reads from DB)", sourceType, profileName)
	return nil
}

// ValidateSync always returns true. YAML sync is no longer used.
func (s *ConnectionSyncService) ValidateSync(ctx context.Context) (bool, error) {
	log.Printf("INFO: ConnectionSyncService.ValidateSync called — always true (YAML sync removed)")
	return true, nil
}
