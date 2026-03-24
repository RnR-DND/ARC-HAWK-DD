package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"

	"github.com/arc-platform/backend/modules/lineage/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

// EnrichmentService adds contextual intelligence to raw findings before classification
type EnrichmentService struct {
	repo           *persistence.PostgresRepository
	lineageService *service.SemanticLineageService
}

// NewEnrichmentService creates a new enrichment service
func NewEnrichmentService(repo *persistence.PostgresRepository, lineageService *service.SemanticLineageService) *EnrichmentService {
	return &EnrichmentService{
		repo:           repo,
		lineageService: lineageService,
	}
}

// EnrichmentSignals contains contextual intelligence about a finding
type EnrichmentSignals struct {
	AssetSemantics   float64 `json:"asset_semantics"`   // Score based on path keywords (0.0-1.0)
	Environment      string  `json:"environment"`       // prod, dev, test, staging
	Entropy          float64 `json:"entropy"`           // Shannon entropy of matched value
	CharsetDiversity float64 `json:"charset_diversity"` // Character distribution score
	TokenShape       string  `json:"token_shape"`       // Pattern shape (e.g., "LLLL-dddd-dddd")
	HistoricalCount  int     `json:"historical_count"`  // Times this pattern+value seen before
	ValueHash        string  `json:"value_hash"`        // SHA256 hash of value for deduplication
	EnrichmentFailed bool    `json:"enrichment_failed"` // Track if enrichment had errors
}

// EnrichmentContext contains input data for enrichment
type EnrichmentContext struct {
	FilePath    string
	MatchValue  string
	PatternName string
	AssetType   string
	ColumnName  string // For database assets
}

// Enrich performs contextual enrichment on a finding
func (s *EnrichmentService) Enrich(ctx context.Context, input EnrichmentContext) EnrichmentSignals {
	signals := EnrichmentSignals{
		EnrichmentFailed: false,
	}

	// 1. Asset Semantics - Score based on path keywords
	signals.AssetSemantics = s.calculateAssetSemantics(input.FilePath, input.ColumnName)

	// 2. Environment Detection
	signals.Environment = s.detectEnvironment(input.FilePath)

	// 3. Entropy Calculation
	signals.Entropy = s.calculateEntropy(input.MatchValue)

	// 4. Charset Diversity
	signals.CharsetDiversity = s.calculateCharsetDiversity(input.MatchValue)

	// 5. Token Shape Analysis
	signals.TokenShape = s.analyzeTokenShape(input.MatchValue)

	// 6. Value Hash (for deduplication tracking)
	signals.ValueHash = s.hashValue(input.MatchValue)

	// 7. Historical Count
	// Query DB for how many times we've seen this pattern+hash combo
	// For now, return 0 - will implement after DB schema update
	signals.HistoricalCount = 0

	return signals
}

// calculateAssetSemantics scores the asset path based on high-risk keywords
func (s *EnrichmentService) calculateAssetSemantics(filePath, columnName string) float64 {
	lower := strings.ToLower(filePath + " " + columnName)
	score := 0.0

	// High-risk contexts (boost confidence)
	highRiskKeywords := []string{
		"user", "users", "customer", "customers",
		"auth", "authentication", "login", "password",
		"billing", "payment", "credit", "card",
		"personal", "private", "sensitive", "confidential",
		"prod", "production", "live",
	}

	for _, keyword := range highRiskKeywords {
		if containsWord(lower, keyword) {
			score += 0.15 // Each keyword adds 15%
		}
	}

	// Low-risk contexts (reduce confidence)
	lowRiskKeywords := []string{
		"test", "tests", "testing", "spec", "specs",
		"mock", "mocks", "fixture", "fixtures",
		"example", "examples", "sample", "demo",
		"tmp", "temp", "cache",
	}

	for _, keyword := range lowRiskKeywords {
		if containsWord(lower, keyword) {
			score -= 0.25 // Each keyword reduces 25%
		}
	}

	// Clamp to [0.0, 1.0]
	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// detectEnvironment detects the environment based on path keywords
func (s *EnrichmentService) detectEnvironment(filePath string) string {
	lower := strings.ToLower(filePath)

	if containsWord(lower, "prod") || containsWord(lower, "production") || containsWord(lower, "live") {
		return "production"
	}
	if containsWord(lower, "staging") || containsWord(lower, "stage") {
		return "staging"
	}
	if containsWord(lower, "dev") || containsWord(lower, "development") {
		return "development"
	}
	if containsWord(lower, "test") || containsWord(lower, "testing") {
		return "test"
	}

	return "unknown"
}

// calculateEntropy calculates Shannon entropy of a string
// Higher entropy = more random/secret-like
func (s *EnrichmentService) calculateEntropy(value string) float64 {
	if len(value) == 0 {
		return 0.0
	}

	// Count frequency of each character
	freq := make(map[rune]int)
	for _, c := range value {
		freq[c]++
	}

	// Calculate Shannon entropy
	entropy := 0.0
	length := float64(len(value))

	for _, count := range freq {
		probability := float64(count) / length
		if probability > 0 {
			entropy -= probability * math.Log2(probability)
		}
	}

	return entropy
}

// calculateCharsetDiversity measures how diverse the character set is
// Returns score 0.0-1.0 where 1.0 means highly diverse (numbers, letters, symbols)
func (s *EnrichmentService) calculateCharsetDiversity(value string) float64 {
	if len(value) == 0 {
		return 0.0
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSymbol := false

	for _, c := range value {
		if c >= 'a' && c <= 'z' {
			hasLower = true
		} else if c >= 'A' && c <= 'Z' {
			hasUpper = true
		} else if c >= '0' && c <= '9' {
			hasDigit = true
		} else {
			hasSymbol = true
		}
	}

	// Count how many character classes are present
	diversity := 0
	if hasLower {
		diversity++
	}
	if hasUpper {
		diversity++
	}
	if hasDigit {
		diversity++
	}
	if hasSymbol {
		diversity++
	}

	return float64(diversity) / 4.0
}

// analyzeTokenShape creates a shape representation of the token
// Examples: "AKIA..." -> "LLLLDDDD...", "test@example.com" -> "llll@lllllll.lll"
func (s *EnrichmentService) analyzeTokenShape(value string) string {
	if len(value) == 0 {
		return ""
	}

	shape := strings.Builder{}
	maxLength := 50 // Limit shape length

	for i, c := range value {
		if i >= maxLength {
			shape.WriteString("...")
			break
		}

		if c >= 'a' && c <= 'z' {
			shape.WriteRune('l')
		} else if c >= 'A' && c <= 'Z' {
			shape.WriteRune('L')
		} else if c >= '0' && c <= '9' {
			shape.WriteRune('d')
		} else if c == ' ' {
			shape.WriteRune(' ')
		} else {
			shape.WriteRune(c) // Keep special chars as-is
		}
	}

	return shape.String()
}

// hashValue creates a SHA256 hash of the value for secure deduplication
func (s *EnrichmentService) hashValue(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

// GetEnrichmentScore returns a composite enrichment score (0.0-1.0)
// This is used as the "Context Score" in the multi-signal classification
func (s *EnrichmentService) GetEnrichmentScore(signals EnrichmentSignals) float64 {
	score := 0.0

	// Asset semantics weight: 40%
	score += signals.AssetSemantics * 0.4

	// Environment weight: 30%
	envScore := 0.0
	switch signals.Environment {
	case "production":
		envScore = 1.0 // Highest risk
	case "staging":
		envScore = 0.7
	case "development":
		envScore = 0.3
	case "test":
		envScore = 0.1 // Lowest risk
	default:
		envScore = 0.5 // Unknown = medium risk
	}
	score += envScore * 0.3

	// Entropy weight: 20%
	// Normalize entropy (typical range 0-5, max theoretical ~6.6 for long strings)
	normalizedEntropy := signals.Entropy / 5.0
	if normalizedEntropy > 1.0 {
		normalizedEntropy = 1.0
	}
	score += normalizedEntropy * 0.2

	// Charset diversity weight: 10%
	score += signals.CharsetDiversity * 0.1

	// Clamp to [0.0, 1.0]
	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// containsWord checks if a word exists with boundaries (not part of another word)
func containsWord(text, word string) bool {
	idx := strings.Index(text, word)
	if idx == -1 {
		return false
	}

	// Check preceding character
	if idx > 0 {
		prev := text[idx-1]
		if (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') {
			return false // Part of another word
		}
	}

	// Check succeeding character
	end := idx + len(word)
	if end < len(text) {
		next := text[end]
		if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
			return false // Part of another word
		}
	}

	return true
}
