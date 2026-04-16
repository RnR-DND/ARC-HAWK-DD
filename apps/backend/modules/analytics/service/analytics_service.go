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
// Uses a single JOIN query instead of per-finding lookups to avoid N+1 performance issues.
func (s *AnalyticsService) GetPIIHeatmap(ctx context.Context) (*PIIHeatmap, error) {
	piiTypes := []string{
		"IN_AADHAAR", "IN_PAN", "IN_PASSPORT", "CREDIT_CARD",
		"IN_UPI", "IN_IFSC", "IN_BANK_ACCOUNT",
		"IN_PHONE", "EMAIL_ADDRESS",
		"IN_VOTER_ID", "IN_DRIVING_LICENSE",
	}

	heatmap := &PIIHeatmap{
		Rows:    []HeatmapRow{},
		Columns: piiTypes,
	}

	rows, err := s.pgRepo.GetDB().QueryContext(ctx, `
		SELECT a.asset_type, c.sub_category, f.severity, COUNT(*) AS cnt
		FROM findings f
		JOIN assets a ON f.asset_id = a.id
		JOIN classifications c ON c.finding_id = f.id
		WHERE c.sub_category IS NOT NULL AND c.sub_category <> ''
		GROUP BY a.asset_type, c.sub_category, f.severity
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query heatmap data: %w", err)
	}
	defer rows.Close()

	type cellData struct {
		count     int
		riskLevel string
	}
	heatData := make(map[string]map[string]*cellData) // assetType -> piiType -> data
	assetTotals := make(map[string]int)
	var orderedAssetTypes []string

	for rows.Next() {
		var assetType, piiType, severity string
		var count int
		if err := rows.Scan(&assetType, &piiType, &severity, &count); err != nil {
			continue
		}
		if _, ok := heatData[assetType]; !ok {
			heatData[assetType] = make(map[string]*cellData)
			orderedAssetTypes = append(orderedAssetTypes, assetType)
		}
		cd := heatData[assetType][piiType]
		if cd == nil {
			cd = &cellData{riskLevel: "Low"}
			heatData[assetType][piiType] = cd
		}
		cd.count += count
		assetTotals[assetType] += count
		switch severity {
		case "Critical":
			cd.riskLevel = "Critical"
		case "High":
			if cd.riskLevel != "Critical" {
				cd.riskLevel = "High"
			}
		case "Medium":
			if cd.riskLevel == "Low" {
				cd.riskLevel = "Medium"
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("heatmap query error: %w", err)
	}

	for _, assetType := range orderedAssetTypes {
		piiMap := heatData[assetType]
		maxCount := 0
		for _, cd := range piiMap {
			if cd.count > maxCount {
				maxCount = cd.count
			}
		}

		row := HeatmapRow{
			AssetType: assetType,
			Cells:     []HeatmapCell{},
			Total:     assetTotals[assetType],
		}
		for _, piiType := range piiTypes {
			count, risk, intensity := 0, "Low", 0
			if cd, ok := piiMap[piiType]; ok {
				count = cd.count
				risk = cd.riskLevel
				if maxCount > 0 {
					intensity = (count * 100) / maxCount
				}
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

// RiskDistribution contains the count of findings per severity level.
type RiskDistribution struct {
	Distribution map[string]int `json:"distribution"`
	Total        int            `json:"total"`
}

// GetRiskDistribution returns a count of findings grouped by severity.
func (s *AnalyticsService) GetRiskDistribution(ctx context.Context) (*RiskDistribution, error) {
	rows, err := s.pgRepo.GetDB().QueryContext(ctx, `
		SELECT COALESCE(severity, 'Low'), COUNT(*)
		FROM findings
		GROUP BY severity
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query risk distribution: %w", err)
	}
	defer rows.Close()

	dist := &RiskDistribution{
		Distribution: make(map[string]int),
	}
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			continue
		}
		dist.Distribution[severity] = count
		dist.Total += count
	}
	return dist, rows.Err()
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
