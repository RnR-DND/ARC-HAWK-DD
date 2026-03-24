package service

import (
	"context"
	"fmt"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

// ClassificationSummaryService handles classification statistics and summaries
type ClassificationSummaryService struct {
	repo *persistence.PostgresRepository
}

// NewClassificationSummaryService creates a new summary service
func NewClassificationSummaryService(repo *persistence.PostgresRepository) *ClassificationSummaryService {
	return &ClassificationSummaryService{repo: repo}
}

// ClassificationSummary represents aggregated classification data
type ClassificationSummary struct {
	Total              int                      `json:"total"`
	ByType             map[string]TypeBreakdown `json:"by_type"`
	BySeverity         map[string]int           `json:"by_severity"`
	HighConfidence     int                      `json:"high_confidence_count"`
	RequiringConsent   int                      `json:"requiring_consent_count"`
	VerifiedCount      int                      `json:"verified_count"`
	FalsePositiveCount int                      `json:"false_positive_count"`
	DPDPACategories    map[string]int           `json:"dpdpa_categories"`
}

// TypeBreakdown represents statistics for a classification type
type TypeBreakdown struct {
	Count           int     `json:"count"`
	AvgConfidence   float64 `json:"avg_confidence"`
	Percentage      float64 `json:"percentage"`
	RequiresConsent int     `json:"requires_consent"`
}

// GetClassificationSummary retrieves aggregated classification statistics
func (s *ClassificationSummaryService) GetClassificationSummary(ctx context.Context) (*ClassificationSummary, error) {
	// Get summary from repository
	rawSummary, err := s.repo.GetClassificationSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get classification summary: %w", err)
	}

	total := rawSummary["total"].(int)
	byTypeRaw := rawSummary["by_type"].(map[string]interface{})

	byType := make(map[string]TypeBreakdown)
	highConfidence := 0
	requiringConsent := 0
	dpdpaCategories := make(map[string]int)

	for typeName, data := range byTypeRaw {
		dataMap := data.(map[string]interface{})
		count := dataMap["count"].(int)
		avgConf := dataMap["avg_confidence"].(float64)

		breakdown := TypeBreakdown{
			Count:         count,
			AvgConfidence: avgConf,
			Percentage:    0,
		}

		if total > 0 {
			breakdown.Percentage = (float64(count) / float64(total)) * 100
		}

		byType[typeName] = breakdown

		if avgConf >= 85.0 {
			highConfidence += count
		}

		// Count DPDPA categories and consent requirements
		switch typeName {
		case "Personal Data", "Sensitive Personal Data":
			requiringConsent += count
			dpdpaCategories[typeName] = count
		case "Secrets":
			dpdpaCategories["N/A"] = count
		}
	}

	// Parse severity breakdown
	bySeverity := make(map[string]int)
	if val, ok := rawSummary["by_severity"].(map[string]int); ok {
		bySeverity = val
	}

	// Parse optional counts (default to 0 if missing)
	verifiedCount := 0
	if val, ok := rawSummary["verified_count"].(int); ok {
		verifiedCount = val
	}

	falsePositiveCount := 0
	if val, ok := rawSummary["false_positive_count"].(int); ok {
		falsePositiveCount = val
	}

	return &ClassificationSummary{
		Total:              total,
		ByType:             byType,
		BySeverity:         bySeverity,
		HighConfidence:     highConfidence,
		RequiringConsent:   requiringConsent,
		VerifiedCount:      verifiedCount,
		FalsePositiveCount: falsePositiveCount,
		DPDPACategories:    dpdpaCategories,
	}, nil
}
