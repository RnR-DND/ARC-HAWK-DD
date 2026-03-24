package service

import (
	"regexp"
	"strings"
	"unicode"
)

// SimilarityConfig defines thresholds for pattern matching
type SimilarityConfig struct {
	MinSimilarityThreshold float64 // Default 0.85
	FieldPathWeight        float64 // Weight for field path similarity
	ValueWeight            float64 // Weight for value similarity
}

var DefaultSimilarityConfig = SimilarityConfig{
	MinSimilarityThreshold: 0.85,
	FieldPathWeight:        0.3,
	ValueWeight:            0.7,
}

// PatternMatch represents a similarity match result
type PatternMatch struct {
	IsMatch        bool
	Similarity     float64
	MatchedPattern string
	MatchType      string // "exact", "fuzzy", "pattern"
	Confidence     float64
}

// LevenshteinDistance calculates the edit distance between two strings
func LevenshteinDistance(s1, s2 string) int {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// CalculateSimilarity returns a similarity score between 0 and 1
func CalculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	distance := LevenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// GeneratePattern extracts a generalized pattern from a value
// Examples:
//   - "test@example.com" -> "email:*@*.com"
//   - "9876543210" -> "phone:10digits"
//   - "ABCDE1234F" -> "alphanumeric:5alpha4digit1alpha"
func GeneratePattern(value, piiType string) string {
	value = strings.TrimSpace(value)

	switch strings.ToLower(piiType) {
	case "email", "email_address":
		return generateEmailPattern(value)
	case "phone", "phone_number":
		return generatePhonePattern(value)
	case "aadhaar":
		return generateAadhaarPattern(value)
	case "pan":
		return generatePANPattern(value)
	default:
		return generateGenericPattern(value)
	}
}

func generateEmailPattern(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "email:invalid"
	}

	domainParts := strings.Split(parts[1], ".")
	if len(domainParts) < 2 {
		return "email:invalid"
	}

	// Pattern: local_length@domain_tld
	tld := domainParts[len(domainParts)-1]
	return "email:*@*." + tld
}

func generatePhonePattern(phone string) string {
	// Extract only digits
	digits := ""
	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits += string(r)
		}
	}

	return "phone:" + string(rune(len(digits))) + "digits"
}

func generateAadhaarPattern(aadhaar string) string {
	// Aadhaar is 12 digits, possibly with spaces
	digits := ""
	for _, r := range aadhaar {
		if unicode.IsDigit(r) {
			digits += string(r)
		}
	}

	if len(digits) == 12 {
		// Return first 4 digits pattern for grouping
		return "aadhaar:****-****-" + digits[8:]
	}
	return "aadhaar:invalid"
}

func generatePANPattern(pan string) string {
	// PAN format: ABCDE1234F
	pan = strings.ToUpper(pan)
	if len(pan) == 10 {
		// Return type code (4th character) as pattern identifier
		return "pan:" + string(pan[3]) + "-type"
	}
	return "pan:invalid"
}

func generateGenericPattern(value string) string {
	alphaCount := 0
	digitCount := 0
	specialCount := 0

	for _, r := range value {
		if unicode.IsLetter(r) {
			alphaCount++
		} else if unicode.IsDigit(r) {
			digitCount++
		} else if !unicode.IsSpace(r) {
			specialCount++
		}
	}

	return "generic:" + strings.Join([]string{
		string(rune('0'+alphaCount%10)) + "a",
		string(rune('0'+digitCount%10)) + "d",
		string(rune('0'+specialCount%10)) + "s",
	}, "-")
}

// MatchPattern checks if a value matches a stored pattern
func MatchPattern(storedPattern, newValue, piiType string) bool {
	newPattern := GeneratePattern(newValue, piiType)
	return storedPattern == newPattern
}

// CalculateFieldPathSimilarity handles field path comparison with structural awareness
func CalculateFieldPathSimilarity(path1, path2 string) float64 {
	// Normalize paths
	path1 = normalizePath(path1)
	path2 = normalizePath(path2)

	if path1 == path2 {
		return 1.0
	}

	// Split into components
	parts1 := strings.Split(path1, ".")
	parts2 := strings.Split(path2, ".")

	// Check if last components match (field name match is important)
	if len(parts1) > 0 && len(parts2) > 0 {
		if parts1[len(parts1)-1] == parts2[len(parts2)-1] {
			// Same field name - high similarity even if paths differ
			return 0.9
		}
	}

	return CalculateSimilarity(path1, path2)
}

func normalizePath(path string) string {
	// Remove array indices [0], [1], etc.
	re := regexp.MustCompile(`\[\d+\]`)
	path = re.ReplaceAllString(path, "[]")

	// Lowercase for comparison
	return strings.ToLower(path)
}

// ComputeOverallMatch determines if a finding matches a stored FP pattern
func ComputeOverallMatch(
	storedFP *StoredFPPattern,
	newFieldPath, newMatchedValue, piiType string,
	config SimilarityConfig,
) PatternMatch {
	result := PatternMatch{
		IsMatch:    false,
		Similarity: 0.0,
		MatchType:  "none",
		Confidence: 0.0,
	}

	// Exact match check first
	if storedFP.FieldPath == newFieldPath && storedFP.MatchedValue == newMatchedValue {
		return PatternMatch{
			IsMatch:        true,
			Similarity:     1.0,
			MatchedPattern: storedFP.Pattern,
			MatchType:      "exact",
			Confidence:     1.0,
		}
	}

	// Pattern match check
	if storedFP.Pattern != "" {
		newPattern := GeneratePattern(newMatchedValue, piiType)
		if storedFP.Pattern == newPattern {
			return PatternMatch{
				IsMatch:        true,
				Similarity:     0.95,
				MatchedPattern: newPattern,
				MatchType:      "pattern",
				Confidence:     0.95,
			}
		}
	}

	// Fuzzy match check
	pathSim := CalculateFieldPathSimilarity(storedFP.FieldPath, newFieldPath)
	valueSim := CalculateSimilarity(storedFP.MatchedValue, newMatchedValue)

	overallSim := pathSim*config.FieldPathWeight + valueSim*config.ValueWeight

	if overallSim >= config.MinSimilarityThreshold {
		return PatternMatch{
			IsMatch:        true,
			Similarity:     overallSim,
			MatchedPattern: storedFP.Pattern,
			MatchType:      "fuzzy",
			Confidence:     overallSim,
		}
	}

	result.Similarity = overallSim
	return result
}

// StoredFPPattern represents a stored false positive pattern for matching
type StoredFPPattern struct {
	FieldPath    string
	MatchedValue string
	Pattern      string
	PIIType      string
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
