package validators

import "strings"

// ValidateBankAccount validates an Indian bank account number.
func ValidateBankAccount(account string) bool {
	cleaned := ""
	for _, c := range account {
		if c >= '0' && c <= '9' {
			cleaned += string(c)
		}
	}
	if len(cleaned) < 9 || len(cleaned) > 18 {
		return false
	}
	// Reject all-same-digit accounts (trivially invalid)
	if strings.Count(cleaned, string(cleaned[0])) == len(cleaned) {
		return false
	}
	return true
}
