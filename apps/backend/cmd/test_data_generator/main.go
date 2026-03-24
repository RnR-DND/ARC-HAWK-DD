package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
)

// TestDataGenerator generates realistic test data for lineage testing
type TestDataGenerator struct {
	rand *rand.Rand
}

// PIIType represents a PII type with its characteristics
type PIIType struct {
	Name            string
	DPDPACategory   string
	RequiresConsent bool
	BaseRisk        string
	SamplePatterns  []string
}

// TestFinding represents a test finding
type TestFinding struct {
	AssetID         uuid.UUID
	AssetName       string
	AssetPath       string
	Host            string
	Environment     string
	PIIType         string
	PatternName     string
	Matches         []string
	Severity        string
	ConfidenceScore float64
	DPDPACategory   string
	RequiresConsent bool
}

var piiTypes = []PIIType{
	{
		Name:            "IN_AADHAAR",
		DPDPACategory:   "Sensitive Personal Data",
		RequiresConsent: true,
		BaseRisk:        "Critical",
		SamplePatterns:  []string{"aadhaar_number", "uid_number"},
	},
	{
		Name:            "IN_PAN",
		DPDPACategory:   "Sensitive Personal Data",
		RequiresConsent: true,
		BaseRisk:        "Critical",
		SamplePatterns:  []string{"pan_number", "permanent_account_number"},
	},
	{
		Name:            "CREDIT_CARD",
		DPDPACategory:   "Sensitive Personal Data",
		RequiresConsent: true,
		BaseRisk:        "Critical",
		SamplePatterns:  []string{"credit_card", "card_number"},
	},
	{
		Name:            "IN_PHONE",
		DPDPACategory:   "Personal Data",
		RequiresConsent: true,
		BaseRisk:        "High",
		SamplePatterns:  []string{"indian_phone", "mobile_number"},
	},
	{
		Name:            "EMAIL_ADDRESS",
		DPDPACategory:   "Personal Data",
		RequiresConsent: true,
		BaseRisk:        "High",
		SamplePatterns:  []string{"email", "email_address"},
	},
	{
		Name:            "IN_PASSPORT",
		DPDPACategory:   "Sensitive Personal Data",
		RequiresConsent: true,
		BaseRisk:        "Critical",
		SamplePatterns:  []string{"passport_number", "indian_passport"},
	},
	{
		Name:            "IN_DRIVING_LICENSE",
		DPDPACategory:   "Sensitive Personal Data",
		RequiresConsent: true,
		BaseRisk:        "High",
		SamplePatterns:  []string{"driving_license", "dl_number"},
	},
}

func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GenerateTestFindings generates realistic test findings
func (g *TestDataGenerator) GenerateTestFindings(numAssets, findingsPerAsset int) []TestFinding {
	findings := []TestFinding{}

	hosts := []string{"prod-db-01.example.com", "staging-db-01.example.com", "analytics-db.example.com"}
	environments := []string{"Production", "Staging", "Development"}

	for i := 0; i < numAssets; i++ {
		assetID := uuid.New()
		host := hosts[g.rand.Intn(len(hosts))]
		env := environments[g.rand.Intn(len(environments))]
		assetName := fmt.Sprintf("users_table_%d", i+1)
		assetPath := fmt.Sprintf("postgresql://%s > public.%s", host, assetName)

		// Generate findings for this asset
		for j := 0; j < findingsPerAsset; j++ {
			piiType := piiTypes[g.rand.Intn(len(piiTypes))]
			pattern := piiType.SamplePatterns[g.rand.Intn(len(piiType.SamplePatterns))]

			// Generate confidence score (biased towards higher values)
			confidence := 0.45 + g.rand.Float64()*0.50 // 0.45 to 0.95

			// Generate matches
			numMatches := 1 + g.rand.Intn(10)
			matches := make([]string, numMatches)
			for k := 0; k < numMatches; k++ {
				matches[k] = fmt.Sprintf("match_%d", k+1)
			}

			finding := TestFinding{
				AssetID:         assetID,
				AssetName:       assetName,
				AssetPath:       assetPath,
				Host:            host,
				Environment:     env,
				PIIType:         piiType.Name,
				PatternName:     pattern,
				Matches:         matches,
				Severity:        piiType.BaseRisk,
				ConfidenceScore: confidence,
				DPDPACategory:   piiType.DPDPACategory,
				RequiresConsent: piiType.RequiresConsent,
			}

			findings = append(findings, finding)
		}
	}

	return findings
}

// ExportToJSON exports findings to JSON file
func (g *TestDataGenerator) ExportToJSON(findings []TestFinding, filename string) error {
	data, err := json.MarshalIndent(findings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// PrintSummary prints a summary of generated findings
func (g *TestDataGenerator) PrintSummary(findings []TestFinding) {
	assetMap := make(map[uuid.UUID]bool)
	piiTypeMap := make(map[string]int)

	for _, f := range findings {
		assetMap[f.AssetID] = true
		piiTypeMap[f.PIIType]++
	}

	fmt.Printf("📊 Test Data Summary:\n")
	fmt.Printf("   - Total Findings: %d\n", len(findings))
	fmt.Printf("   - Unique Assets: %d\n", len(assetMap))
	fmt.Printf("   - PII Type Distribution:\n")
	for piiType, count := range piiTypeMap {
		fmt.Printf("     • %s: %d findings\n", piiType, count)
	}
}

func main() {
	ctx := context.Background()
	_ = ctx // For future use

	generator := NewTestDataGenerator()

	// Generate test data: 10 assets, 5-15 findings per asset
	numAssets := 10
	findingsPerAsset := 5 + rand.Intn(10)

	fmt.Printf("🔧 Generating test data...\n")
	fmt.Printf("   - Assets: %d\n", numAssets)
	fmt.Printf("   - Findings per asset: ~%d\n", findingsPerAsset)

	findings := generator.GenerateTestFindings(numAssets, findingsPerAsset)

	generator.PrintSummary(findings)

	// Export to JSON
	filename := "test_findings.json"
	if err := generator.ExportToJSON(findings, filename); err != nil {
		log.Fatalf("Failed to export test data: %v", err)
	}

	fmt.Printf("\n✅ Test data exported to: %s\n", filename)
	fmt.Printf("\n💡 Next Steps:\n")
	fmt.Printf("   1. Ingest this data using the scanner or ingestion API\n")
	fmt.Printf("   2. Trigger lineage sync: POST /api/v1/lineage/sync\n")
	fmt.Printf("   3. Verify lineage graph: GET /api/v1/lineage\n")
	fmt.Printf("   4. Check frontend visualization at /lineage\n")
}
