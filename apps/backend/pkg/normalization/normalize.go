package normalization

import (
	"strings"
	"unicode"
)

// NormalizeForDedup creates canonical form for deduplication
// This ensures that "john@example.com" and " john@example.com " are treated as duplicates
func NormalizeForDedup(value string) string {
	// Remove all whitespace
	value = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)

	// Remove common delimiters that don't change semantic meaning
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, ".", "")

	// Lowercase for case-insensitive comparison
	return strings.ToLower(value)
}

// Normalize converts value to canonical form for Presidio analysis
// This ensures Presidio sees the cleaned value without extra formatting
func Normalize(value string) string {
	// Trim leading/trailing whitespace
	value = strings.TrimSpace(value)

	// Collapse multiple spaces to single space
	value = strings.Join(strings.Fields(value), " ")

	return value
}

// ExtractDigits removes all non-digit characters
// Used for validating numeric patterns like credit cards, SSNs, etc.
func ExtractDigits(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, value)
}

// NormalizeEmail removes dots before @ and lowercases
// Gmail treats "john.doe@gmail.com" and "johndoe@gmail.com" as identical
func NormalizeEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return strings.ToLower(email) // Invalid email, just lowercase
	}

	localPart := strings.ReplaceAll(parts[0], ".", "")
	domain := parts[1]

	return strings.ToLower(localPart + "@" + domain)
}
