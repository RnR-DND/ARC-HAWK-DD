package validators

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var testDomains = []string{"example.com", "test.com", "sample.com", "dummy.com", "placeholder.com"}

// ValidateEmail validates an email address.
func ValidateEmail(email string) bool {
	if !emailRegex.MatchString(email) {
		return false
	}
	lower := strings.ToLower(email)
	for _, d := range testDomains {
		if strings.HasSuffix(lower, "@"+d) {
			return false
		}
	}
	return true
}
