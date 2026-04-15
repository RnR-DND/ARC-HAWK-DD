package validators

import "regexp"

var ifscRegex = regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)

// ValidateIFSC validates an IFSC code.
func ValidateIFSC(ifsc string) bool {
	return ifscRegex.MatchString(ifsc)
}
