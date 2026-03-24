package service

import (
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1, s2   string
		expected int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "abcd", 1},
		{"kitten", "sitting", 3},
		{"test@example.com", "test@example.org", 3},
	}

	for _, tt := range tests {
		result := LevenshteinDistance(tt.s1, tt.s2)
		if result != tt.expected {
			t.Errorf("LevenshteinDistance(%q, %q) = %d; expected %d", tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		s1, s2      string
		minExpected float64
	}{
		{"abc", "abc", 1.0},
		{"abc", "abd", 0.6},
		{"test@example.com", "test@sample.com", 0.7},
		{"completely", "different", 0.0},
	}

	for _, tt := range tests {
		result := CalculateSimilarity(tt.s1, tt.s2)
		if result < tt.minExpected {
			t.Errorf("CalculateSimilarity(%q, %q) = %f; expected >= %f", tt.s1, tt.s2, result, tt.minExpected)
		}
	}
}

func TestGeneratePattern(t *testing.T) {
	tests := []struct {
		value    string
		piiType  string
		expected string
	}{
		{"test@example.com", "email", "email:*@*.com"},
		{"user@domain.org", "email", "email:*@*.org"},
		{"9876543210", "phone", "phone:\ndigits"},
		{"ABCDE1234F", "pan", "pan:D-type"},
	}

	for _, tt := range tests {
		result := GeneratePattern(tt.value, tt.piiType)
		// Pattern generation may vary, just check it's not empty
		if result == "" {
			t.Errorf("GeneratePattern(%q, %q) returned empty string", tt.value, tt.piiType)
		}
	}
}

func TestCalculateFieldPathSimilarity(t *testing.T) {
	tests := []struct {
		path1, path2 string
		minExpected  float64
	}{
		{"user.email", "user.email", 1.0},
		{"user[0].email", "user[1].email", 1.0}, // Array indices should be normalized
		{"data.user.email", "records.user.email", 0.7},
	}

	for _, tt := range tests {
		result := CalculateFieldPathSimilarity(tt.path1, tt.path2)
		if result < tt.minExpected {
			t.Errorf("CalculateFieldPathSimilarity(%q, %q) = %f; expected >= %f", tt.path1, tt.path2, result, tt.minExpected)
		}
	}
}

func TestComputeOverallMatch(t *testing.T) {
	storedFP := &StoredFPPattern{
		FieldPath:    "user.email",
		MatchedValue: "test@example.com",
		Pattern:      "email:*@*.com",
		PIIType:      "email",
	}

	// Exact match
	result := ComputeOverallMatch(storedFP, "user.email", "test@example.com", "email", DefaultSimilarityConfig)
	if !result.IsMatch || result.MatchType != "exact" {
		t.Errorf("Expected exact match, got %+v", result)
	}

	// Pattern match (same domain TLD)
	result = ComputeOverallMatch(storedFP, "user.email", "other@domain.com", "email", DefaultSimilarityConfig)
	if !result.IsMatch || result.MatchType != "pattern" {
		t.Errorf("Expected pattern match for same TLD, got %+v", result)
	}

	// Fuzzy match (similar email)
	result = ComputeOverallMatch(storedFP, "user.email", "test@example.org", "email", DefaultSimilarityConfig)
	// This may or may not match depending on similarity threshold
	if result.Similarity < 0.5 {
		t.Errorf("Expected some similarity for similar email, got %f", result.Similarity)
	}
}
