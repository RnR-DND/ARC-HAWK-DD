package service

import (
	"context"
	"fmt"
	"time"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/domain/repository"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

// AnalyticsService provides PII heatmap and trend analytics
type AnalyticsService struct {
	pgRepo *persistence.PostgresRepository
}

// PIIHeatmap represents PII distribution across asset types and PII types
type PIIHeatmap struct {
	Rows    []HeatmapRow `json:"rows"`
	Columns []string     `json:"columns"` // 11 PII types
}

// HeatmapRow represents a row in the heatmap (asset type)
type HeatmapRow struct {
	AssetType string        `json:"asset_type"`
	Cells     []HeatmapCell `json:"cells"`
	Total     int           `json:"total"`
}

// HeatmapCell represents a cell in the heatmap
type HeatmapCell struct {
	PIIType      string `json:"pii_type"`
	FindingCount int    `json:"finding_count"`
	RiskLevel    string `json:"risk_level"` // critical, high, medium, low
	Intensity    int    `json:"intensity"`  // 0-100 for color intensity
}

// RiskTrend represents risk trends over time
type RiskTrend struct {
	Timeline         []TimelinePoint `json:"timeline"`
	RiskDistribution map[string]int  `json:"risk_distribution"`
	NewlyExposed     int             `json:"newly_exposed"`
	Resolved         int             `json:"resolved"`
}

// TimelinePoint represents a point in time
type TimelinePoint struct {
	Date        string `json:"date"`
	TotalPII    int    `json:"total_pii"`
	CriticalPII int    `json:"critical_pii"`
	HighPII     int    `json:"high_pii"`
	MediumPII   int    `json:"medium_pii"`
	LowPII      int    `json:"low_pii"`
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(pgRepo *persistence.PostgresRepository) *AnalyticsService {
	return &AnalyticsService{
		pgRepo: pgRepo,
	}
}

// GetPIIHeatmap returns the PII distribution heatmap
func (s *AnalyticsService) GetPIIHeatmap(ctx context.Context) (*PIIHeatmap, error) {
	// Define 11 locked PII types
	piiTypes := []string{
		"IN_AADHAAR", "IN_PAN", "IN_PASSPORT", "CREDIT_CARD",
		"IN_UPI", "IN_IFSC", "IN_BANK_ACCOUNT",
		"IN_PHONE", "EMAIL_ADDRESS",
		"IN_VOTER_ID", "IN_DRIVING_LICENSE",
	}

	// Define asset types
	assetTypes := []string{"file", "database"}

	heatmap := &PIIHeatmap{
		Rows:    []HeatmapRow{},
		Columns: piiTypes,
	}

	// Get all findings
	findings, err := s.pgRepo.ListFindings(ctx, repository.FindingFilters{}, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list findings: %w", err)
	}

	// Build heatmap data
	for _, assetType := range assetTypes {
		row := HeatmapRow{
			AssetType: assetType,
			Cells:     []HeatmapCell{},
			Total:     0,
		}

		// Count findings for each PII type
		piiCounts := make(map[string]int)
		piiRisks := make(map[string]string)
		maxCount := 0

		for _, finding := range findings {
			// Get asset
			asset, err := s.pgRepo.GetAssetByID(ctx, finding.AssetID)
			if err != nil || asset.AssetType != assetType {
				continue
			}

			// Get classification
			classifications, err := s.pgRepo.GetClassificationsByFindingID(ctx, finding.ID)
			if err != nil || len(classifications) == 0 {
				continue
			}

			piiType := classifications[0].SubCategory
			if piiType == "" {
				continue
			}

			piiCounts[piiType]++
			row.Total++

			if piiCounts[piiType] > maxCount {
				maxCount = piiCounts[piiType]
			}

			// Track highest risk level
			if finding.Severity == "Critical" {
				piiRisks[piiType] = "Critical"
			} else if finding.Severity == "High" && piiRisks[piiType] != "Critical" {
				piiRisks[piiType] = "High"
			} else if finding.Severity == "Medium" && piiRisks[piiType] != "Critical" && piiRisks[piiType] != "High" {
				piiRisks[piiType] = "Medium"
			} else if piiRisks[piiType] == "" {
				piiRisks[piiType] = "Low"
			}
		}

		// Create cells for each PII type
		for _, piiType := range piiTypes {
			count := piiCounts[piiType]
			risk := piiRisks[piiType]
			if risk == "" {
				risk = "Low"
			}

			intensity := 0
			if maxCount > 0 {
				intensity = (count * 100) / maxCount
			}

			row.Cells = append(row.Cells, HeatmapCell{
				PIIType:      piiType,
				FindingCount: count,
				RiskLevel:    risk,
				Intensity:    intensity,
			})
		}

		heatmap.Rows = append(heatmap.Rows, row)
	}

	return heatmap, nil
}

// GetRiskTrend returns risk trends over time
func (s *AnalyticsService) GetRiskTrend(ctx context.Context, days int) (*RiskTrend, error) {
	if days <= 0 {
		days = 30 // Default to 30 days
	}

	trend := &RiskTrend{
		Timeline:         []TimelinePoint{},
		RiskDistribution: make(map[string]int),
		NewlyExposed:     0,
		Resolved:         0,
	}

	// Get all findings
	findings, err := s.pgRepo.ListFindings(ctx, repository.FindingFilters{}, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list findings: %w", err)
	}

	// Group findings by date
	findingsByDate := make(map[string][]*entity.Finding)
	for _, finding := range findings {
		date := finding.CreatedAt.Format("2006-01-02")
		findingsByDate[date] = append(findingsByDate[date], finding)
	}

	// Build timeline for last N days
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")

		point := TimelinePoint{
			Date:        date,
			TotalPII:    0,
			CriticalPII: 0,
			HighPII:     0,
			MediumPII:   0,
			LowPII:      0,
		}

		// Count findings for this date
		for _, finding := range findingsByDate[date] {
			point.TotalPII++

			switch finding.Severity {
			case "Critical":
				point.CriticalPII++
				trend.RiskDistribution["Critical"]++
			case "High":
				point.HighPII++
				trend.RiskDistribution["High"]++
			case "Medium":
				point.MediumPII++
				trend.RiskDistribution["Medium"]++
			default:
				point.LowPII++
				trend.RiskDistribution["Low"]++
			}
		}

		trend.Timeline = append(trend.Timeline, point)
	}

	// Calculate newly exposed vs resolved (simplified)
	// In production, this would track asset state changes over time
	if len(trend.Timeline) > 1 {
		lastPoint := trend.Timeline[len(trend.Timeline)-1]
		prevPoint := trend.Timeline[len(trend.Timeline)-2]

		if lastPoint.TotalPII > prevPoint.TotalPII {
			trend.NewlyExposed = lastPoint.TotalPII - prevPoint.TotalPII
		} else {
			trend.Resolved = prevPoint.TotalPII - lastPoint.TotalPII
		}
	}

	return trend, nil
}
