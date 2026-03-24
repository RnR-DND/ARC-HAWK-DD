package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ConsentBasis represents the legal basis for processing personal data
type ConsentBasis string

const (
	ConsentBasisExplicit           ConsentBasis = "explicit"
	ConsentBasisLegitimateInterest ConsentBasis = "legitimate_interest"
	ConsentBasisContractual        ConsentBasis = "contractual"
	ConsentBasisLegalObligation    ConsentBasis = "legal_obligation"
)

// ConsentStatus represents the current status of a consent record
type ConsentStatus string

const (
	ConsentStatusValid        ConsentStatus = "VALID"
	ConsentStatusExpired      ConsentStatus = "EXPIRED"
	ConsentStatusExpiringSoon ConsentStatus = "EXPIRING_SOON"
	ConsentStatusWithdrawn    ConsentStatus = "WITHDRAWN"
)

// ConsentRecord represents a consent record for DPDPA compliance
type ConsentRecord struct {
	ID                    string                 `json:"id"`
	AssetID               string                 `json:"asset_id"`
	PIIType               string                 `json:"pii_type"`
	ConsentObtainedAt     time.Time              `json:"consent_obtained_at"`
	ConsentExpiresAt      *time.Time             `json:"consent_expires_at,omitempty"`
	ConsentWithdrawnAt    *time.Time             `json:"consent_withdrawn_at,omitempty"`
	ConsentBasis          ConsentBasis           `json:"consent_basis"`
	Purpose               string                 `json:"purpose"`
	ObtainedBy            string                 `json:"obtained_by"`
	WithdrawalRequestedBy *string                `json:"withdrawal_requested_by,omitempty"`
	WithdrawalReason      *string                `json:"withdrawal_reason,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
	Status                ConsentStatus          `json:"status"`
}

// ConsentRequest represents a request to record consent
type ConsentRequest struct {
	AssetID           string                 `json:"asset_id" binding:"required"`
	PIIType           string                 `json:"pii_type" binding:"required"`
	ConsentObtainedAt time.Time              `json:"consent_obtained_at" binding:"required"`
	ConsentExpiresAt  *time.Time             `json:"consent_expires_at,omitempty"`
	ConsentBasis      ConsentBasis           `json:"consent_basis" binding:"required"`
	Purpose           string                 `json:"purpose" binding:"required"`
	ObtainedBy        string                 `json:"obtained_by" binding:"required"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ConsentWithdrawalRequest represents a request to withdraw consent
type ConsentWithdrawalRequest struct {
	WithdrawalRequestedBy string `json:"withdrawal_requested_by" binding:"required"`
	WithdrawalReason      string `json:"withdrawal_reason"`
}

// ConsentFilters represents filters for querying consent records
type ConsentFilters struct {
	AssetID string
	PIIType string
	Status  ConsentStatus
	Limit   int
	Offset  int
}

// ConsentService handles consent management operations
type ConsentService struct {
	db *sql.DB
}

// NewConsentService creates a new consent service
func NewConsentService(db *sql.DB) *ConsentService {
	return &ConsentService{db: db}
}

// RecordConsent records a new consent
func (s *ConsentService) RecordConsent(ctx context.Context, req ConsentRequest) (*ConsentRecord, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO consent_records (
			id, asset_id, pii_type, consent_obtained_at, consent_expires_at,
			consent_basis, purpose, obtained_by, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := s.db.QueryRowContext(
		ctx, query,
		id, req.AssetID, req.PIIType, req.ConsentObtainedAt, req.ConsentExpiresAt,
		req.ConsentBasis, req.Purpose, req.ObtainedBy, req.Metadata,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to record consent: %w", err)
	}

	return &ConsentRecord{
		ID:                id,
		AssetID:           req.AssetID,
		PIIType:           req.PIIType,
		ConsentObtainedAt: req.ConsentObtainedAt,
		ConsentExpiresAt:  req.ConsentExpiresAt,
		ConsentBasis:      req.ConsentBasis,
		Purpose:           req.Purpose,
		ObtainedBy:        req.ObtainedBy,
		Metadata:          req.Metadata,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		Status:            ConsentStatusValid,
	}, nil
}

// WithdrawConsent withdraws an existing consent
func (s *ConsentService) WithdrawConsent(ctx context.Context, consentID string, req ConsentWithdrawalRequest) error {
	query := `
		UPDATE consent_records
		SET consent_withdrawn_at = NOW(),
		    withdrawal_requested_by = $1,
		    withdrawal_reason = $2
		WHERE id = $3 AND consent_withdrawn_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, req.WithdrawalRequestedBy, req.WithdrawalReason, consentID)
	if err != nil {
		return fmt.Errorf("failed to withdraw consent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent not found or already withdrawn")
	}

	return nil
}

// GetConsentStatus gets the consent status for a specific asset and PII type
func (s *ConsentService) GetConsentStatus(ctx context.Context, assetID, piiType string) (*ConsentRecord, error) {
	query := `
		SELECT 
			id, asset_id, pii_type, consent_obtained_at, consent_expires_at,
			consent_withdrawn_at, consent_basis, purpose, obtained_by,
			withdrawal_requested_by, withdrawal_reason, metadata,
			created_at, updated_at, status
		FROM consent_status_view
		WHERE asset_id = $1 AND pii_type = $2
		ORDER BY consent_obtained_at DESC
		LIMIT 1
	`

	var record ConsentRecord
	var metadata []byte

	err := s.db.QueryRowContext(ctx, query, assetID, piiType).Scan(
		&record.ID, &record.AssetID, &record.PIIType, &record.ConsentObtainedAt,
		&record.ConsentExpiresAt, &record.ConsentWithdrawnAt, &record.ConsentBasis,
		&record.Purpose, &record.ObtainedBy, &record.WithdrawalRequestedBy,
		&record.WithdrawalReason, &metadata, &record.CreatedAt, &record.UpdatedAt,
		&record.Status,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No consent found
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get consent status: %w", err)
	}

	return &record, nil
}

// ListConsentRecords lists consent records with optional filters
func (s *ConsentService) ListConsentRecords(ctx context.Context, filters ConsentFilters) ([]ConsentRecord, error) {
	query := `
		SELECT 
			id, asset_id, pii_type, consent_obtained_at, consent_expires_at,
			consent_withdrawn_at, consent_basis, purpose, obtained_by,
			withdrawal_requested_by, withdrawal_reason, metadata,
			created_at, updated_at, status
		FROM consent_status_view
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	if filters.AssetID != "" {
		query += fmt.Sprintf(" AND asset_id = $%d", argCount)
		args = append(args, filters.AssetID)
		argCount++
	}

	if filters.PIIType != "" {
		query += fmt.Sprintf(" AND pii_type = $%d", argCount)
		args = append(args, filters.PIIType)
		argCount++
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filters.Status)
		argCount++
	}

	query += " ORDER BY consent_obtained_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)
		argCount++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filters.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list consent records: %w", err)
	}
	defer rows.Close()

	var records []ConsentRecord
	for rows.Next() {
		var record ConsentRecord
		var metadata []byte

		err := rows.Scan(
			&record.ID, &record.AssetID, &record.PIIType, &record.ConsentObtainedAt,
			&record.ConsentExpiresAt, &record.ConsentWithdrawnAt, &record.ConsentBasis,
			&record.Purpose, &record.ObtainedBy, &record.WithdrawalRequestedBy,
			&record.WithdrawalReason, &metadata, &record.CreatedAt, &record.UpdatedAt,
			&record.Status,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan consent record: %w", err)
		}

		records = append(records, record)
	}

	return records, nil
}

// GetConsentViolations returns assets with findings that lack valid consent
func (s *ConsentService) GetConsentViolations(ctx context.Context) ([]ConsentViolation, error) {
	query := `
		SELECT 
			f.asset_id,
			a.name AS asset_name,
			f.pii_type,
			COUNT(f.id) AS finding_count,
			COALESCE(cs.status, 'MISSING') AS consent_status
		FROM findings f
		JOIN assets a ON f.asset_id = a.id
		LEFT JOIN consent_status_view cs ON f.asset_id = cs.asset_id AND f.pii_type = cs.pii_type
		WHERE cs.status IS NULL OR cs.status IN ('EXPIRED', 'WITHDRAWN')
		GROUP BY f.asset_id, a.name, f.pii_type, cs.status
		ORDER BY finding_count DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent violations: %w", err)
	}
	defer rows.Close()

	var violations []ConsentViolation
	for rows.Next() {
		var violation ConsentViolation
		err := rows.Scan(
			&violation.AssetID, &violation.AssetName, &violation.PIIType,
			&violation.FindingCount, &violation.ConsentStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent violation: %w", err)
		}
		violations = append(violations, violation)
	}

	return violations, nil
}

// ConsentViolation represents an asset with consent violations
type ConsentViolation struct {
	AssetID       string `json:"asset_id"`
	AssetName     string `json:"asset_name"`
	PIIType       string `json:"pii_type"`
	FindingCount  int    `json:"finding_count"`
	ConsentStatus string `json:"consent_status"`
}
