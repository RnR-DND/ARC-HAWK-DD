package validators

import "regexp"

var passportRegex = regexp.MustCompile(`^[A-Z][0-9]{7}$`)

// ValidatePassport validates an Indian passport number.
func ValidatePassport(passport string) bool {
	return passportRegex.MatchString(passport)
}
