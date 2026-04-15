package validators

import "regexp"

var panRegex = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)
var validPANEntities = map[byte]bool{
	'P': true, 'C': true, 'H': true, 'F': true,
	'A': true, 'T': true, 'B': true, 'L': true,
	'J': true, 'G': true,
}

// ValidatePAN validates an Indian PAN card number.
func ValidatePAN(pan string) bool {
	if !panRegex.MatchString(pan) {
		return false
	}
	return validPANEntities[pan[3]]
}
