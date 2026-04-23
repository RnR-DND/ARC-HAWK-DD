package classifier

import (
	"math"
	"strings"

	"github.com/arc-platform/go-scanner/internal/classifier/validators"
	"github.com/arc-platform/go-scanner/internal/connectors"
)

// validatorMap maps PIIType to its math validator function.
var validatorMap = map[string]func(string) bool{
	"IN_AADHAAR":         validators.ValidateAadhaar,
	"IN_PAN":             validators.ValidatePAN,
	"CREDIT_CARD":        validators.ValidateCreditCard,
	"EMAIL_ADDRESS":      validators.ValidateEmail,
	"IN_PHONE":           validators.ValidatePhone,
	"IN_PASSPORT":        validators.ValidatePassport,
	"IN_VOTER_ID":        validators.ValidateVoterID,
	"IN_DRIVING_LICENSE": validators.ValidateDrivingLicense,
	"IN_BANK_ACCOUNT":    validators.ValidateBankAccount,
	"IN_IFSC":            validators.ValidateIFSC,
	"IN_GST":             validators.ValidateGST,
	"IN_UPI":             validators.ValidateUPI,
}

// ShannonEntropy calculates the Shannon entropy of a string.
func ShannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	freq := make(map[rune]float64)
	for _, c := range s {
		freq[c]++
	}
	n := float64(len(s))
	h := 0.0
	for _, count := range freq {
		p := count / n
		h -= p * math.Log2(p)
	}
	return h
}

// Score returns confidence score (0-100) and detector type for a matched value.
func Score(value, piiType string, record connectors.FieldRecord, contextKeywords, negKeywords []string) (int, string) {
	lower := strings.ToLower(value)

	// Math validator takes priority; it is the authoritative gatekeeper for
	// types that have one (e.g. EMAIL_ADDRESS already rejects example.com).
	if vfn, ok := validatorMap[piiType]; ok {
		if vfn(value) {
			score := 100
			ctx := strings.ToLower(record.RowContext)
			for _, kw := range contextKeywords {
				if strings.Contains(ctx, strings.ToLower(kw)) {
					if score < 100 {
						score++
					}
				}
			}
			for _, kw := range negKeywords {
				if strings.Contains(ctx, strings.ToLower(kw)) {
					score = int(float64(score) * 0.85)
				}
			}
			return score, "math"
		}
		return 0, "regex"
	}

	// Heuristic for patterns without a math validator.
	// Apply test-data guard here (not before the math-validator block) so that
	// type-specific validators remain the sole authority for their own types.
	for _, w := range []string{"test", "example", "sample", "dummy", "placeholder", "12345", "00000"} {
		if strings.Contains(lower, w) {
			return 0, "regex"
		}
	}

	score := 50
	if ShannonEntropy(value) > 3.0 {
		score += 20
	}

	ctx := strings.ToLower(record.RowContext)
	for _, kw := range contextKeywords {
		if strings.Contains(ctx, strings.ToLower(kw)) {
			if score < 100 {
				score += 5
			}
		}
	}
	for _, kw := range negKeywords {
		if strings.Contains(ctx, strings.ToLower(kw)) {
			score = int(float64(score) * 0.85)
		}
	}
	if score > 100 {
		score = 100
	}

	return score, "regex"
}
