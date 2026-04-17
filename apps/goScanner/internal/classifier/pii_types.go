package classifier

// frontendToPatternNames maps frontend-selected PII type names to the
// internal Pattern.Name values compiled in patterns.go.
// Mirrors apps/scanner/sdk/pii_type_mapping.py from the deleted Python scanner.
var frontendToPatternNames = map[string][]string{
	"PAN":              {"PAN"},
	"AADHAAR":          {"Aadhaar"},
	"EMAIL":            {"Email"},
	"PHONE":            {"Phone India"},
	"CREDIT_CARD":      {"Credit Card Visa", "Credit Card MC", "Credit Card Amex", "Credit Card Discover"},
	"PASSPORT":         {"Passport India"},
	"UPI_ID":           {"UPI"},
	"UPI":              {"UPI"},
	"IFSC":             {"IFSC"},
	"BANK_ACCOUNT":     {"Bank Account"},
	"GST":              {"GST"},
	"VOTER_ID":         {"Voter ID"},
	"DRIVING_LICENSE":  {"Driving License"},
	"AWS_ACCESS_KEY":   {"AWS Access Key"},
	"PRIVATE_KEY":      {"Private Key"},
	"GENERIC_API_KEY":  {"Generic API Key"},
	"GENERIC_SECRET":   {"Generic Secret"},
	"JWT_TOKEN":        {"JWT Token"},
	"BEARER_TOKEN":     {"Bearer Token"},
	"IP_ADDRESS":       {"IP Address"},
	"MAC_ADDRESS":      {"MAC Address"},
	"DATE_OF_BIRTH":    {"Date of Birth"},
	"EPF_UAN":          {"EPF UAN"},
	"ESIC":             {"ESIC"},
	"HEALTH_RECORD":    {"Health Record"},
}

// frontendToPresidioEntities maps frontend-selected PII names to Presidio
// entity types. Kept for parity with the Python scanner; the regex-only Go
// engine currently does not forward to Presidio but downstream code may use
// the mapping to tag findings with canonical ML entity names.
var frontendToPresidioEntities = map[string][]string{
	"PAN":             {"IN_PAN"},
	"AADHAAR":         {"IN_AADHAAR"},
	"EMAIL":           {"EMAIL_ADDRESS"},
	"PHONE":           {"IN_PHONE"},
	"CREDIT_CARD":     {"CREDIT_CARD"},
	"DRIVING_LICENSE": {"IN_DRIVING_LICENSE"},
	"VOTER_ID":        {"IN_VOTER_ID"},
	"PASSPORT":        {"IN_PASSPORT"},
	"GST":             {"IN_GST"},
}

// AllowedPatternSet returns the set of internal Pattern.Name values that
// correspond to the given frontend PII type names. An empty or nil input
// returns an empty set (caller should treat "no filter" as "allow all").
func AllowedPatternSet(frontendPIITypes []string) map[string]struct{} {
	out := make(map[string]struct{}, len(frontendPIITypes)*2)
	for _, t := range frontendPIITypes {
		for _, name := range frontendToPatternNames[t] {
			out[name] = struct{}{}
		}
	}
	return out
}

// PresidioEntitiesFor returns the Presidio entity names for the given
// frontend PII type selection. Returns nil when the input is empty, which
// means "all entities" in Python-parity terminology.
func PresidioEntitiesFor(frontendPIITypes []string) []string {
	if len(frontendPIITypes) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(frontendPIITypes))
	for _, t := range frontendPIITypes {
		for _, e := range frontendToPresidioEntities[t] {
			if _, ok := seen[e]; ok {
				continue
			}
			seen[e] = struct{}{}
			out = append(out, e)
		}
	}
	return out
}
