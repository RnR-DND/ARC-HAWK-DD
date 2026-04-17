package presidio

// AdHocRecognizer is one recognizer definition that Presidio accepts on a
// per-request basis via the `ad_hoc_recognizers` field of /analyze. It maps
// directly to the Presidio Python PatternRecognizer class.
type AdHocRecognizer struct {
	Name              string        `json:"name"`
	SupportedLanguage string        `json:"supported_language"`
	SupportedEntity   string        `json:"supported_entity"`
	Patterns          []PatternSpec `json:"patterns"`
	Context           []string      `json:"context,omitempty"`
}

// PatternSpec is one regex pattern inside an AdHocRecognizer.
type PatternSpec struct {
	Name  string  `json:"name"`
	Regex string  `json:"regex"`
	Score float64 `json:"score"`
}

// IndianRecognizers returns the ad-hoc recognizers that teach Presidio about
// India-specific PII (IN_PAN, IN_AADHAAR, IN_GST, IN_VOTER_ID, IN_IFSC,
// IN_DRIVING_LICENSE, IN_PASSPORT, IN_UPI, IN_BANK_ACCOUNT, IN_PHONE).
//
// These are regex-with-context recognizers — the regex is deliberately
// conservative (high-score when the context words appear nearby via Presidio's
// LemmaContextAwareEnhancer, lower baseline otherwise). The Go scanner's local
// regex engine already matches the same formats; the value of sending them to
// Presidio is the context-aware scoring when mode == "contextual".
func IndianRecognizers() []AdHocRecognizer {
	return []AdHocRecognizer{
		{
			Name:              "in_pan_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_PAN",
			Context:           []string{"pan", "permanent", "account", "card", "tax", "income"},
			Patterns: []PatternSpec{
				{Name: "pan_exact", Regex: `\b[A-Z]{5}[0-9]{4}[A-Z]\b`, Score: 0.85},
			},
		},
		{
			Name:              "in_aadhaar_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_AADHAAR",
			Context:           []string{"aadhaar", "aadhar", "uid", "unique", "identification"},
			Patterns: []PatternSpec{
				{Name: "aadhaar_exact", Regex: `\b[2-9]\d{3}[\s-]?\d{4}[\s-]?\d{4}\b`, Score: 0.6},
			},
		},
		{
			Name:              "in_gst_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_GST",
			Context:           []string{"gst", "gstin", "goods", "services", "tax"},
			Patterns: []PatternSpec{
				{Name: "gst_exact", Regex: `\b[0-3][0-9][A-Z]{5}[0-9]{4}[A-Z][0-9A-Z]Z[0-9A-Z]\b`, Score: 0.9},
			},
		},
		{
			Name:              "in_voter_id_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_VOTER_ID",
			Context:           []string{"voter", "epic", "election", "commission"},
			Patterns: []PatternSpec{
				{Name: "voter_exact", Regex: `\b[A-Z]{3}[0-9]{7}\b`, Score: 0.7},
			},
		},
		{
			Name:              "in_ifsc_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_IFSC",
			Context:           []string{"ifsc", "bank", "branch", "code", "rtgs", "neft"},
			Patterns: []PatternSpec{
				{Name: "ifsc_exact", Regex: `\b[A-Z]{4}0[A-Z0-9]{6}\b`, Score: 0.9},
			},
		},
		{
			Name:              "in_driving_license_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_DRIVING_LICENSE",
			Context:           []string{"driving", "license", "licence", "dl", "rto"},
			Patterns: []PatternSpec{
				{Name: "dl_exact", Regex: `\b[A-Z]{2}[- ]?\d{2}[- ]?\d{4,7}\b`, Score: 0.6},
			},
		},
		{
			Name:              "in_passport_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_PASSPORT",
			Context:           []string{"passport", "travel", "document"},
			Patterns: []PatternSpec{
				{Name: "passport_exact", Regex: `\b[A-Z][0-9]{7}\b`, Score: 0.7},
			},
		},
		{
			Name:              "in_upi_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_UPI",
			Context:           []string{"upi", "vpa", "handle", "phonepe", "gpay", "paytm"},
			Patterns: []PatternSpec{
				{Name: "upi_exact", Regex: `\b[a-zA-Z0-9.\-_]{3,}@[a-zA-Z0-9.\-_]+\b`, Score: 0.5},
			},
		},
		{
			Name:              "in_bank_account_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_BANK_ACCOUNT",
			Context:           []string{"account", "bank", "savings", "current", "accnt", "a/c"},
			Patterns: []PatternSpec{
				{Name: "account_exact", Regex: `\b[0-9]{9,18}\b`, Score: 0.4},
			},
		},
		{
			Name:              "in_phone_recognizer",
			SupportedLanguage: "en",
			SupportedEntity:   "IN_PHONE",
			Context:           []string{"phone", "mobile", "contact", "cell", "tel"},
			Patterns: []PatternSpec{
				{Name: "phone_exact", Regex: `(?:\+91|91|0)?[6-9]\d{9}`, Score: 0.7},
			},
		},
	}
}

// IndianEntityNames returns the set of Presidio supported_entity names this
// package teaches Presidio about — used to build the entity allowlist alongside
// Presidio's built-in entities.
func IndianEntityNames() []string {
	out := make([]string, 0, 10)
	for _, r := range IndianRecognizers() {
		out = append(out, r.SupportedEntity)
	}
	return out
}
