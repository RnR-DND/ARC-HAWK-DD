package service

import (
	"context"
	"fmt"
	"time"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/domain/repository"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// FindingsService handles findings queries
type FindingsService struct {
	repo *persistence.PostgresRepository
}

// NewFindingsService creates a new findings service
func NewFindingsService(repo *persistence.PostgresRepository) *FindingsService {
	return &FindingsService{repo: repo}
}

// FindingsQuery represents query parameters
type FindingsQuery struct {
	ScanRunID   *uuid.UUID
	AssetID     *uuid.UUID
	Severity    string
	PatternName string
	DataSource  string
	Search      string
	Page        int
	PageSize    int
	SortBy      string
	SortOrder   string
}

// FindingsResponse represents paginated findings response
type FindingsResponse struct {
	Findings   []*FindingWithDetails `json:"findings"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

// FindingWithDetails includes finding with asset and classification details
type FindingWithDetails struct {
	*entity.Finding
	AssetName       string                   `json:"asset_name"`
	AssetPath       string                   `json:"asset_path"`
	Environment     string                   `json:"environment"`
	Owner           string                   `json:"owner"`
	SourceSystem    string                   `json:"source_system"`
	Classifications []*entity.Classification `json:"classifications"`
	ReviewStatus    string                   `json:"review_status"`
}

// GetFindings retrieves paginated and filtered findings
func (s *FindingsService) GetFindings(ctx context.Context, query FindingsQuery) (*FindingsResponse, error) {
	// Set defaults
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	offset := (query.Page - 1) * query.PageSize

	// Build filters
	filters := repository.FindingFilters{
		ScanRunID:   query.ScanRunID,
		AssetID:     query.AssetID,
		Severity:    query.Severity,
		PatternName: query.PatternName,
		DataSource:  query.DataSource,
		Search:      query.Search,
	}

	// Get findings
	findings, err := s.repo.ListFindings(ctx, filters, query.PageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list findings: %w", err)
	}

	// Get total count
	total, err := s.repo.CountFindings(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to count findings: %w", err)
	}

	// Enrich findings with details
	enrichedFindings := make([]*FindingWithDetails, 0, len(findings))
	for _, finding := range findings {
		// Get asset details
		asset, err := s.repo.GetAssetByID(ctx, finding.AssetID)
		if err != nil {
			return nil, fmt.Errorf("failed to get asset: %w", err)
		}

		// Get classifications
		classifications, err := s.repo.GetClassificationsByFindingID(ctx, finding.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get classifications: %w", err)
		}

		// Get review status
		reviewState, err := s.repo.GetReviewStateByFindingID(ctx, finding.ID)
		reviewStatus := "pending"
		if err == nil && reviewState != nil {
			reviewStatus = reviewState.Status
		}

		enrichedFindings = append(enrichedFindings, &FindingWithDetails{
			Finding:         finding,
			AssetName:       asset.Name,
			AssetPath:       asset.Path,
			Environment:     asset.Environment,
			Owner:           asset.Owner,
			SourceSystem:    asset.SourceSystem,
			Classifications: classifications,
			ReviewStatus:    reviewStatus,
		})
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize

	return &FindingsResponse{
		Findings:   enrichedFindings,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}

// SubmitFeedback records user feedback for a finding
func (s *FindingsService) SubmitFeedback(ctx context.Context, feedback *entity.FindingFeedback) error {
	// 1. Verify finding exists
	_, err := s.repo.GetFindingByID(ctx, feedback.FindingID)
	if err != nil {
		return fmt.Errorf("finding not found: %w", err)
	}

	// 2. Set defaults
	if feedback.ID == uuid.Nil {
		feedback.ID = uuid.New()
	}
	if feedback.UserID == "" {
		feedback.UserID = "system"
	}

	// Capture original state if not provided (though FE should probably provide it)
	if feedback.OriginalClassification == "" {
		// In a real scenario, we'd fetch the classification entity.
		// For now, we assume the client sends what they saw.
		feedback.OriginalClassification = "Unknown"
	}

	// 3. Save feedback
	if err := s.repo.CreateFeedback(ctx, feedback); err != nil {
		return fmt.Errorf("failed to save feedback: %w", err)
	}

	// 4. IMMEDIATE ACTION: Update Review State based on feedback
	// If User says "False Positive", we should mark it as such.
	reviewStatus := "pending"
	switch feedback.FeedbackType {
	case entity.FeedbackTypeFalsePositive:
		reviewStatus = "false_positive"
	case entity.FeedbackTypeConfirmed:
		reviewStatus = "confirmed"
	}

	// Update or Create Review State
	existingState, err := s.repo.GetReviewStateByFindingID(ctx, feedback.FindingID)
	if err != nil {
		return fmt.Errorf("failed to check review state: %w", err)
	}

	now := time.Now()
	if existingState != nil {
		// Update existing
		existingState.Status = reviewStatus
		existingState.ReviewedBy = feedback.UserID
		existingState.ReviewedAt = &now
		existingState.Comments = feedback.Comments
		if err := s.repo.UpdateReviewState(ctx, existingState); err != nil {
			return fmt.Errorf("failed to update review state: %w", err)
		}
	} else {
		// Create new
		newState := &entity.ReviewState{
			ID:         uuid.New(),
			FindingID:  feedback.FindingID,
			Status:     reviewStatus,
			ReviewedBy: feedback.UserID,
			ReviewedAt: &now,
			Comments:   feedback.Comments,
		}
		if err := s.repo.CreateReviewState(ctx, newState); err != nil {
			return fmt.Errorf("failed to create review state: %w", err)
		}
	}

	return nil
}

// GetFindingsByAsset retrieves all findings for a specific asset
// Implements FindingsProvider interface
func (s *FindingsService) GetFindingsByAsset(ctx context.Context, assetID uuid.UUID, limit, offset int) ([]*entity.Finding, error) {
	filters := repository.FindingFilters{
		AssetID: &assetID,
	}
	return s.repo.ListFindings(ctx, filters, limit, offset)
}

// GetClassificationsByFinding retrieves classifications for a finding
// Implements FindingsProvider interface
func (s *FindingsService) GetClassificationsByFinding(ctx context.Context, findingID uuid.UUID) ([]*entity.Classification, error) {
	return s.repo.GetClassificationsByFindingID(ctx, findingID)
}

// CountFindings returns total findings matching filters
// Implements FindingsProvider interface
func (s *FindingsService) CountFindings(ctx context.Context, filters repository.FindingFilters) (int, error) {
	return s.repo.CountFindings(ctx, filters)
}
