package validation

import "regexp"

// PAN (Permanent Account Number - India) format: AAAAA9999A
// 5 letters + 4 digits + 1 letter
var panRegex = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]{1}$`)

// ValidatePAN checks if a string is a valid Indian PAN format
func ValidatePAN(pan string) bool {
	if len(pan) != 10 {
		return false
	}

	return panRegex.MatchString(pan)
}

// ValidatePANWithDetails performs additional semantic validation
// Fourth character should be 'P' for individual, 'C' for company, etc.
func ValidatePANWithDetails(pan string) (bool, string) {
	if !ValidatePAN(pan) {
		return false, "Invalid PAN format"
	}

	// Fourth character indicates entity type
	fourthChar := pan[3]
	switch fourthChar {
	case 'P':
		return true, "Individual"
	case 'C':
		return true, "Company"
	case 'H':
		return true, "HUF (Hindu Undivided Family)"
	case 'A':
		return true, "AOP (Association of Persons)"
	case 'B':
		return true, "BOI (Body of Individuals)"
	case 'G':
		return true, "Government"
	case 'J':
		return true, "Artificial Juridical Person"
	case 'L':
		return true, "Local Authority"
	case 'F':
		return true, "Firm"
	case 'T':
		return true, "Trust"
	default:
		return false, "Unknown entity type"
	}
}
