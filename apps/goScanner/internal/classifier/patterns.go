package classifier

import (
	"log"
	"regexp"
)

// Pattern represents a compiled PII detection pattern.
type Pattern struct {
	Name            string
	PIIType         string
	Regex           *regexp.Regexp
	ContextKeywords []string // if non-empty, at least one must appear in field/table name or score is penalised
}

// CustomPattern is a user-defined pattern with context keyword scoring.
// RawRegex preserves the original source expression so downstream components
// (e.g. the Presidio ad-hoc-recognizer bridge) can forward the regex without
// re-deriving it from the compiled *regexp.Regexp.
type CustomPattern struct {
	ID               string
	Name             string
	PIIType          string
	Regex            *regexp.Regexp
	RawRegex         string
	ContextKeywords  []string
	NegativeKeywords []string
}

// AllPatterns returns all built-in PII patterns.
func AllPatterns() []Pattern {
	defs := []struct {
		name, piiType, regex string
		contextKeywords      []string
	}{
		// Boundary classes explicitly exclude hex letters (A-Fa-f) so the pattern
		// does NOT match digit runs embedded inside SHA-256 hashes or other
		// hex-encoded values (previously produced hundreds of false positives
		// when scanning audit_log/findings tables with hex columns).
		{"Aadhaar", "IN_AADHAAR", `(?:^|[^0-9A-Za-z])([2-9]\d{3}[\s-]?\d{4}[\s-]?\d{4})(?:[^0-9A-Za-z]|$)`, nil},
		{"PAN", "IN_PAN", `\b[A-Z]{5}[0-9]{4}[A-Z]\b`, nil},
		{"Credit Card Visa", "CREDIT_CARD", `\b4[0-9]{12}(?:[0-9]{3})?\b`, nil},
		{"Credit Card MC", "CREDIT_CARD", `\b5[1-5][0-9]{14}\b`, nil},
		{"Credit Card Amex", "CREDIT_CARD", `\b3[47][0-9]{13}\b`, nil},
		{"Credit Card Discover", "CREDIT_CARD", `\b6(?:011|5[0-9]{2})[0-9]{12}\b`, nil},
		{"Email", "EMAIL_ADDRESS", `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`, nil},
		// \b on both sides prevents matching 10-digit substrings inside hex
		// hashes (hex letters are word chars in RE2, so \b will not fire
		// between `d` and `8` etc.). Note: a leading `+91` before a whitespace
		// boundary will not match here; add a separate pattern if needed.
		{"Phone India", "IN_PHONE", `\b(?:91|0)?[6-9]\d{9}\b`, nil},
		{"Phone India +91", "IN_PHONE", `\+91[\s-]?[6-9]\d{9}\b`, nil},
		{"GST", "IN_GST", `[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][0-9A-Z]Z[0-9A-Z]`, nil},
		{"UPI", "IN_UPI", `[a-zA-Z0-9.\-_]{3,}@[a-zA-Z0-9.\-_]+`, nil},
		{"IFSC", "IN_IFSC", `[A-Z]{4}0[A-Z0-9]{6}`, nil},
		{"Passport India", "IN_PASSPORT", `\b[A-Z][0-9]{7}\b`, nil},
		{"Voter ID", "IN_VOTER_ID", `\b[A-Z]{3}[0-9]{7}\b`, nil},
		{"Driving License", "IN_DRIVING_LICENSE", `[A-Z]{2}[- ]?\d{2}[- ]?\d{4,7}`, nil},
		// IN_BANK_ACCOUNT regex is intentionally permissive; context keywords gate
		// false positives from timestamps, employee IDs, and similar digit sequences.
		{"Bank Account", "IN_BANK_ACCOUNT", `\b[0-9]{9,18}\b`, []string{"account", "acct", "bank", "ifsc", "savings", "current", "routing"}},
		{"AWS Access Key", "AWS_ACCESS_KEY", `AKIA[0-9A-Z]{16}`, nil},
		{"Private Key", "PRIVATE_KEY", `-----BEGIN [A-Z ]+ PRIVATE KEY-----`, nil},
		{"Generic API Key", "GENERIC_API_KEY", `(?i)api[_-]?key["':\s]+[A-Za-z0-9_\-]{20,}`, nil},
		{"Generic Secret", "GENERIC_SECRET", `(?i)secret["':\s]+[A-Za-z0-9_\-]{16,}`, nil},
		{"JWT Token", "JWT_TOKEN", `eyJ[A-Za-z0-9\-_]+\.eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`, nil},
		{"Bearer Token", "BEARER_TOKEN", `(?i)Bearer\s+[A-Za-z0-9\-_.~+/]+=*`, nil},
		{"IP Address", "IP_ADDRESS", `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`, nil},
		{"MAC Address", "MAC_ADDRESS", `[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}`, nil},
		{"Date of Birth", "DATE_OF_BIRTH", `\b(?:0[1-9]|[12][0-9]|3[01])[/\-.](?:0[1-9]|1[012])[/\-.](?:19|20)\d{2}\b`, nil},
		{"EPF UAN", "IN_EPF_UAN", `\b\d{12}\b`, nil},
		{"ESIC", "IN_ESIC", `\b\d{17}\b`, nil},
		{"Health Record", "HEALTH_RECORD", `(?i)(?:blood\s*group|diagnosis|prescription|icd[-\s]?10|uhid|abha)\s*:?\s*[A-Z0-9\-+]+`, nil},
	}

	patterns := make([]Pattern, 0, len(defs))
	for _, d := range defs {
		r, err := regexp.Compile(d.regex)
		if err != nil {
			log.Printf("classifier: failed to compile pattern %q (%s): %v", d.name, d.piiType, err)
			continue
		}
		patterns = append(patterns, Pattern{Name: d.name, PIIType: d.piiType, Regex: r, ContextKeywords: d.contextKeywords})
	}
	return patterns
}
