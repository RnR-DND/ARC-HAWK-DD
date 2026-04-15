package validators

import "regexp"

var voterRegex = regexp.MustCompile(`^[A-Z]{3}[0-9]{7}$`)

// ValidateVoterID validates an Indian voter ID.
func ValidateVoterID(id string) bool {
	return voterRegex.MatchString(id)
}
