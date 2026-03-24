package service

import (
	"context"
	"fmt"
	"time"

	"github.com/arc-platform/backend/modules/fplearning/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

type FPLearningService struct {
	repo *persistence.PostgresRepository
}

func NewFPLearningService(repo *persistence.PostgresRepository) *FPLearningService {
	return &FPLearningService{repo: repo}
}

func (s *FPLearningService) CreateFalsePositive(
	ctx context.Context,
	tenantID, userID, assetID uuid.UUID,
	patternName, piiType, fieldName, fieldPath, matchedValue, justification string,
	sourceFindingID, scanRunID *uuid.UUID,
) (*entity.FPLearning, error) {
	fp := &entity.FPLearning{
		ID:              uuid.New(),
		TenantID:        tenantID,
		UserID:          userID,
		AssetID:         assetID,
		PatternName:     patternName,
		PIIType:         piiType,
		FieldName:       fieldName,
		FieldPath:       fieldPath,
		MatchedValue:    matchedValue,
		LearningType:    entity.FPLearningTypeFalsePositive,
		Version:         1,
		Justification:   justification,
		SourceFindingID: sourceFindingID,
		ScanRunID:       scanRunID,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateFPLearning(ctx, fp); err != nil {
		return nil, fmt.Errorf("failed to create FP learning: %w", err)
	}

	return fp, nil
}

func (s *FPLearningService) CreateConfirmed(
	ctx context.Context,
	tenantID, userID, assetID uuid.UUID,
	patternName, piiType, fieldName, fieldPath, matchedValue string,
	sourceFindingID, scanRunID *uuid.UUID,
) (*entity.FPLearning, error) {
	fp := &entity.FPLearning{
		ID:              uuid.New(),
		TenantID:        tenantID,
		UserID:          userID,
		AssetID:         assetID,
		PatternName:     patternName,
		PIIType:         piiType,
		FieldName:       fieldName,
		FieldPath:       fieldPath,
		MatchedValue:    matchedValue,
		LearningType:    entity.FPLearningTypeConfirmed,
		Version:         1,
		Justification:   "Confirmed by user",
		SourceFindingID: sourceFindingID,
		ScanRunID:       scanRunID,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateFPLearning(ctx, fp); err != nil {
		return nil, fmt.Errorf("failed to create confirmed learning: %w", err)
	}

	return fp, nil
}

func (s *FPLearningService) CheckFalsePositive(
	ctx context.Context,
	tenantID, assetID uuid.UUID,
	patternName, piiType, fieldPath, matchedValue string,
) (*entity.FPLearning, bool, error) {
	falsePositiveType := entity.FPLearningTypeFalsePositive
	filter := entity.FPLearningFilter{
		TenantID:     tenantID,
		AssetID:      &assetID,
		PatternName:  patternName,
		PIIType:      piiType,
		LearningType: &falsePositiveType,
		IsActive:     boolPtr(true),
	}

	learning, err := s.repo.GetFPLearningByFilter(ctx, filter)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check FP learning: %w", err)
	}

	if learning == nil {
		return nil, false, nil
	}

	matched := s.matchFP(learning, fieldPath, matchedValue)
	return learning, matched, nil
}

func (s *FPLearningService) matchFP(fp *entity.FPLearning, fieldPath, matchedValue string) bool {
	// Create stored pattern for comparison
	storedPattern := &StoredFPPattern{
		FieldPath:    fp.FieldPath,
		MatchedValue: fp.MatchedValue,
		Pattern:      GeneratePattern(fp.MatchedValue, fp.PIIType),
		PIIType:      fp.PIIType,
	}

	// Use ML-based matching with default config
	match := ComputeOverallMatch(storedPattern, fieldPath, matchedValue, fp.PIIType, DefaultSimilarityConfig)
	return match.IsMatch
}

func (s *FPLearningService) GetFPLearnings(
	ctx context.Context,
	tenantID uuid.UUID,
	filter entity.FPLearningFilter,
	page, pageSize int,
) ([]*entity.FPLearning, int, error) {
	filter.TenantID = tenantID
	return s.repo.GetFPLearnings(ctx, filter, page, pageSize)
}

func (s *FPLearningService) GetFPLearningByID(ctx context.Context, id uuid.UUID) (*entity.FPLearning, error) {
	return s.repo.GetFPLearningByID(ctx, id)
}

func (s *FPLearningService) DeactivateFPLearning(ctx context.Context, id, userID uuid.UUID) error {
	fp, err := s.repo.GetFPLearningByID(ctx, id)
	if err != nil {
		return fmt.Errorf("FP learning not found: %w", err)
	}

	fp.IsActive = false
	fp.Version++
	fp.UpdatedAt = time.Now()

	return s.repo.UpdateFPLearning(ctx, fp)
}

func (s *FPLearningService) GetStats(ctx context.Context, tenantID uuid.UUID) (*entity.FPLearningStats, error) {
	filter := entity.FPLearningFilter{
		TenantID: tenantID,
		IsActive: boolPtr(true),
	}

	fps, err := s.repo.GetAllFPLearnings(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get FP learnings: %w", err)
	}

	stats := &entity.FPLearningStats{
		TotalPatterns:  len(fps),
		FalsePositives: 0,
		Confirmed:      0,
		ByPIIType:      make(map[string]int),
		ByAsset:        make(map[string]int),
	}

	var latestAt *time.Time
	for _, fp := range fps {
		if fp.LearningType == entity.FPLearningTypeFalsePositive {
			stats.FalsePositives++
		} else if fp.LearningType == entity.FPLearningTypeConfirmed {
			stats.Confirmed++
		}

		stats.ByPIIType[fp.PIIType]++
		stats.ByAsset[fp.AssetID.String()]++

		if latestAt == nil || fp.CreatedAt.After(*latestAt) {
			latestAt = &fp.CreatedAt
		}
	}

	stats.LatestLearningAt = latestAt
	return stats, nil
}

func (s *FPLearningService) CheckAndSuppressFinding(
	ctx context.Context,
	tenantID, userID, assetID uuid.UUID,
	patternName, piiType, fieldPath, matchedValue string,
) (bool, string, error) {
	fp, isMatch, err := s.CheckFalsePositive(ctx, tenantID, assetID, patternName, piiType, fieldPath, matchedValue)
	if err != nil {
		return false, "", err
	}

	if isMatch && fp != nil {
		return true, fp.ID.String(), nil
	}

	return false, "", nil
}

func boolPtr(b bool) *bool {
	return &b
}
