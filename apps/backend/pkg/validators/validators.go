package validators

import (
	"regexp"
	"strings"
	"unicode"
)

// Validate checks whether value passes format validation for the given PII type.
// Returns true if the value looks structurally valid (not necessarily real).
// Unknown PII types pass by default (no false rejection).
func Validate(piiType, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	switch piiType {
	case "CREDIT_CARD":
		return validateCreditCard(value)
	case "IN_AADHAAR":
		return validateAadhaar(value)
	case "IN_PAN":
		return validatePAN(value)
	case "EMAIL_ADDRESS":
		return validateEmail(value)
	case "IN_PHONE":
		return validateIndianPhone(value)
	case "IN_UPI":
		return validateUPI(value)
	case "IN_IFSC":
		return validateIFSC(value)
	case "IN_PASSPORT":
		return validatePassport(value)
	case "IN_VOTER_ID":
		return validateVoterID(value)
	case "IN_DRIVING_LICENSE":
		return validateDrivingLicense(value)
	case "IN_BANK_ACCOUNT":
		return validateBankAccount(value)
	default:
		return true // Unknown type — don't reject
	}
}

// MapPatternToPIIType converts scanner pattern names to canonical PII type strings.
// Returns empty string for unrecognized patterns (caller should keep the finding).
func MapPatternToPIIType(patternName string) string {
	name := strings.ToLower(patternName)
	switch {
	case strings.Contains(name, "aadhaar") || strings.Contains(name, "aadhar"):
		return "IN_AADHAAR"
	case strings.Contains(name, "pan"):
		return "IN_PAN"
	case strings.Contains(name, "credit"):
		return "CREDIT_CARD"
	case strings.Contains(name, "email"):
		return "EMAIL_ADDRESS"
	case strings.Contains(name, "phone"):
		return "IN_PHONE"
	case strings.Contains(name, "passport"):
		return "IN_PASSPORT"
	case strings.Contains(name, "upi"):
		return "IN_UPI"
	case strings.Contains(name, "ifsc"):
		return "IN_IFSC"
	case strings.Contains(name, "voter"):
		return "IN_VOTER_ID"
	case strings.Contains(name, "driving"), strings.Contains(name, "license"), strings.Contains(name, "licence"):
		return "IN_DRIVING_LICENSE"
	case strings.Contains(name, "bank"), strings.Contains(name, "account"):
		return "IN_BANK_ACCOUNT"
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// Credit Card — Luhn algorithm, 13-19 digits
// ---------------------------------------------------------------------------

func validateCreditCard(value string) bool {
	digits := extractDigits(value)
	n := len(digits)
	if n < 13 || n > 19 {
		return false
	}
	return luhnCheck(digits)
}

// luhnCheck implements the Luhn mod-10 checksum.
func luhnCheck(digits string) bool {
	sum := 0
	n := len(digits)
	parity := n % 2

	for i := 0; i < n; i++ {
		d := int(digits[i] - '0')
		if i%2 == parity {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

// ---------------------------------------------------------------------------
// Aadhaar — exactly 12 digits, Verhoeff checksum
// ---------------------------------------------------------------------------

func validateAadhaar(value string) bool {
	digits := extractDigits(value)
	if len(digits) != 12 {
		return false
	}
	// Aadhaar cannot start with 0 or 1
	if digits[0] == '0' || digits[0] == '1' {
		return false
	}
	return verhoeffCheck(digits)
}

// Verhoeff multiplication table
var verhoeffD = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 2, 3, 4, 0, 6, 7, 8, 9, 5},
	{2, 3, 4, 0, 1, 7, 8, 9, 5, 6},
	{3, 4, 0, 1, 2, 8, 9, 5, 6, 7},
	{4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
	{5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
	{6, 5, 9, 8, 7, 1, 0, 4, 3, 2},
	{7, 6, 5, 9, 8, 2, 1, 0, 4, 3},
	{8, 7, 6, 5, 9, 3, 2, 1, 0, 4},
	{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

// Verhoeff permutation table
var verhoeffP = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
	{5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
	{8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
	{9, 4, 5, 3, 1, 2, 6, 8, 7, 0},
	{4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
	{2, 7, 9, 3, 8, 0, 6, 4, 1, 5},
	{7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}

// verhoeffCheck validates a numeric string using the Verhoeff algorithm.
func verhoeffCheck(digits string) bool {
	c := 0
	n := len(digits)
	for i := n - 1; i >= 0; i-- {
		pos := n - 1 - i
		d := int(digits[i] - '0')
		c = verhoeffD[c][verhoeffP[pos%8][d]]
	}
	return c == 0
}

// ---------------------------------------------------------------------------
// PAN — [A-Z]{5}[0-9]{4}[A-Z], exactly 10 characters
// ---------------------------------------------------------------------------

var panRegex = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)

func validatePAN(value string) bool {
	v := strings.ToUpper(strings.TrimSpace(value))
	if len(v) != 10 {
		return false
	}
	return panRegex.MatchString(v)
}

// ---------------------------------------------------------------------------
// Email — basic RFC-style validation
// ---------------------------------------------------------------------------

func validateEmail(value string) bool {
	v := strings.TrimSpace(value)
	at := strings.LastIndex(v, "@")
	if at < 1 || at >= len(v)-1 {
		return false
	}
	local := v[:at]
	domain := v[at+1:]

	if len(local) == 0 || len(local) > 64 {
		return false
	}
	if len(domain) < 3 || !strings.Contains(domain, ".") {
		return false
	}
	// Domain must not start/end with dot or hyphen
	if domain[0] == '.' || domain[0] == '-' || domain[len(domain)-1] == '.' || domain[len(domain)-1] == '-' {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Indian Phone — 10 digits starting with 6-9
// ---------------------------------------------------------------------------

func validateIndianPhone(value string) bool {
	digits := extractDigits(value)
	// Handle +91 prefix
	if strings.HasPrefix(digits, "91") && len(digits) == 12 {
		digits = digits[2:]
	}
	if len(digits) != 10 {
		return false
	}
	return digits[0] >= '6' && digits[0] <= '9'
}

// ---------------------------------------------------------------------------
// UPI ID — username@bankhandle
// ---------------------------------------------------------------------------

var upiRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+@[a-zA-Z][a-zA-Z0-9]*$`)

func validateUPI(value string) bool {
	v := strings.TrimSpace(value)
	if !strings.Contains(v, "@") {
		return false
	}
	return upiRegex.MatchString(v)
}

// ---------------------------------------------------------------------------
// IFSC — [A-Z]{4}0[A-Z0-9]{6}, exactly 11 characters
// ---------------------------------------------------------------------------

var ifscRegex = regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)

func validateIFSC(value string) bool {
	v := strings.ToUpper(strings.TrimSpace(value))
	if len(v) != 11 {
		return false
	}
	return ifscRegex.MatchString(v)
}

// ---------------------------------------------------------------------------
// Passport — [A-Z][0-9]{7}, exactly 8 characters
// ---------------------------------------------------------------------------

var passportRegex = regexp.MustCompile(`^[A-Z][0-9]{7}$`)

func validatePassport(value string) bool {
	v := strings.ToUpper(strings.TrimSpace(value))
	if len(v) != 8 {
		return false
	}
	return passportRegex.MatchString(v)
}

// ---------------------------------------------------------------------------
// Voter ID — [A-Z]{3}[0-9]{7}, exactly 10 characters
// ---------------------------------------------------------------------------

var voterIDRegex = regexp.MustCompile(`^[A-Z]{3}[0-9]{7}$`)

func validateVoterID(value string) bool {
	v := strings.ToUpper(strings.TrimSpace(value))
	if len(v) != 10 {
		return false
	}
	return voterIDRegex.MatchString(v)
}

// ---------------------------------------------------------------------------
// Driving License — 2-letter state code + hyphen/space (optional) + 13 digits/chars
// Total 15-16 chars. Format varies by state but always starts with state code.
// ---------------------------------------------------------------------------

var dlRegex = regexp.MustCompile(`^[A-Z]{2}[\s-]?\d{2}[\s-]?\d{4}[\s-]?\d{7}$`)

func validateDrivingLicense(value string) bool {
	v := strings.ToUpper(strings.TrimSpace(value))
	// Remove common separators for length check
	cleaned := strings.NewReplacer("-", "", " ", "").Replace(v)
	if len(cleaned) < 13 || len(cleaned) > 16 {
		return false
	}
	return dlRegex.MatchString(v)
}

// ---------------------------------------------------------------------------
// Bank Account — 9-18 digits
// ---------------------------------------------------------------------------

func validateBankAccount(value string) bool {
	digits := extractDigits(value)
	n := len(digits)
	return n >= 9 && n <= 18
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func extractDigits(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
