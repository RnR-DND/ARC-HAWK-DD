package validators

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var testDomains = []string{"example.com", "test.com", "sample.com", "dummy.com", "placeholder.com"}

// testLocalParts are local-part values that indicate synthetic/test data.
var testLocalParts = []string{"test", "example", "sample", "dummy", "placeholder"}

// ValidateEmail validates an email address.
func ValidateEmail(email string) bool {
	if !emailRegex.MatchString(email) {
		return false
	}
	lower := strings.ToLower(email)
	// Reject well-known test domains.
	for _, d := range testDomains {
		if strings.HasSuffix(lower, "@"+d) {
			return false
		}
	}
	// Reject emails whose local part is a generic test-data placeholder.
	atIdx := strings.Index(lower, "@")
	if atIdx >= 0 {
		localPart := lower[:atIdx]
		for _, w := range testLocalParts {
			if localPart == w {
				return false
			}
		}
	}
	return true
}
