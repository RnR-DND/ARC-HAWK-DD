package validators

import (
	"regexp"
	"strings"
)

var upiRegex = regexp.MustCompile(`^[a-zA-Z0-9.\-_]{3,256}@[a-zA-Z0-9.\-_]+$`)
var knownVPAs = []string{
	"okaxis", "okhdfcbank", "okicici", "oksbi", "paytm", "ybl", "ibl",
	"apl", "upi", "axl", "hdfcbank", "sbi", "icici", "kotak",
}

// ValidateUPI validates a UPI VPA.
func ValidateUPI(upi string) bool {
	if !upiRegex.MatchString(upi) {
		return false
	}
	parts := strings.SplitN(upi, "@", 2)
	if len(parts) != 2 {
		return false
	}
	provider := strings.ToLower(parts[1])
	for _, vpa := range knownVPAs {
		if strings.Contains(provider, vpa) {
			return true
		}
	}
	return len(provider) >= 3 && len(provider) <= 20
}
