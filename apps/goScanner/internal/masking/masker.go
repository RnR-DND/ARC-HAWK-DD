package masking

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// HashValue returns the SHA-256 hex of a value. Raw PII never leaves the scanner.
func HashValue(v string) string {
	h := sha256.Sum256([]byte(v))
	return fmt.Sprintf("%x", h)
}

// Redact replaces a value with [REDACTED] in context text.
func Redact(context, value string) string {
	return strings.ReplaceAll(context, value, "[REDACTED]")
}
