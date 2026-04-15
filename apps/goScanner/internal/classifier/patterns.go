package classifier

import "regexp"

// Pattern represents a compiled PII detection pattern.
type Pattern struct {
	Name    string
	PIIType string
	Regex   *regexp.Regexp
}

// CustomPattern is a user-defined pattern with context keyword scoring.
type CustomPattern struct {
	ID               string
	Name             string
	PIIType          string
	Regex            *regexp.Regexp
	ContextKeywords  []string
	NegativeKeywords []string
}

// AllPatterns returns all built-in PII patterns.
func AllPatterns() []Pattern {
	defs := []struct{ name, piiType, regex string }{
		{"Aadhaar", "IN_AADHAAR", `(?:^|[^0-9])([2-9]\d{3}[\s-]?\d{4}[\s-]?\d{4})(?:[^0-9]|$)`},
		{"PAN", "IN_PAN", `\b[A-Z]{5}[0-9]{4}[A-Z]\b`},
		{"Credit Card Visa", "CREDIT_CARD", `\b4[0-9]{12}(?:[0-9]{3})?\b`},
		{"Credit Card MC", "CREDIT_CARD", `\b5[1-5][0-9]{14}\b`},
		{"Credit Card Amex", "CREDIT_CARD", `\b3[47][0-9]{13}\b`},
		{"Credit Card Discover", "CREDIT_CARD", `\b6(?:011|5[0-9]{2})[0-9]{12}\b`},
		{"Email", "EMAIL_ADDRESS", `[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`},
		{"Phone India", "IN_PHONE", `(?:\+91|91|0)?[6-9]\d{9}`},
		{"GST", "IN_GST", `[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][0-9A-Z]Z[0-9A-Z]`},
		{"UPI", "IN_UPI", `[a-zA-Z0-9.\-_]{3,}@[a-zA-Z0-9.\-_]+`},
		{"IFSC", "IN_IFSC", `[A-Z]{4}0[A-Z0-9]{6}`},
		{"Passport India", "IN_PASSPORT", `\b[A-Z][0-9]{7}\b`},
		{"Voter ID", "IN_VOTER_ID", `\b[A-Z]{3}[0-9]{7}\b`},
		{"Driving License", "IN_DRIVING_LICENSE", `[A-Z]{2}[- ]?\d{2}[- ]?\d{4,7}`},
		{"Bank Account", "IN_BANK_ACCOUNT", `\b[0-9]{9,18}\b`},
		{"AWS Access Key", "AWS_ACCESS_KEY", `AKIA[0-9A-Z]{16}`},
		{"Private Key", "PRIVATE_KEY", `-----BEGIN [A-Z ]+ PRIVATE KEY-----`},
		{"Generic API Key", "GENERIC_API_KEY", `(?i)api[_-]?key["':\s]+[A-Za-z0-9_\-]{20,}`},
		{"Generic Secret", "GENERIC_SECRET", `(?i)secret["':\s]+[A-Za-z0-9_\-]{16,}`},
		{"JWT Token", "JWT_TOKEN", `eyJ[A-Za-z0-9\-_]+\.eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`},
		{"Bearer Token", "BEARER_TOKEN", `(?i)Bearer\s+[A-Za-z0-9\-_.~+/]+=*`},
		{"IP Address", "IP_ADDRESS", `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`},
		{"MAC Address", "MAC_ADDRESS", `[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}`},
		{"Date of Birth", "DATE_OF_BIRTH", `\b(?:0[1-9]|[12][0-9]|3[01])[/\-.](?:0[1-9]|1[012])[/\-.](?:19|20)\d{2}\b`},
		{"EPF UAN", "IN_EPF_UAN", `\b\d{12}\b`},
		{"ESIC", "IN_ESIC", `\b\d{17}\b`},
		{"Health Record", "HEALTH_RECORD", `(?i)(?:blood\s*group|diagnosis|prescription|icd[-\s]?10|uhid|abha)\s*:?\s*[A-Z0-9\-+]+`},
	}

	patterns := make([]Pattern, 0, len(defs))
	for _, d := range defs {
		r, err := regexp.Compile(d.regex)
		if err != nil {
			continue
		}
		patterns = append(patterns, Pattern{Name: d.name, PIIType: d.piiType, Regex: r})
	}
	return patterns
}
