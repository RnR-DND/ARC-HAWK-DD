package validators

import "regexp"

var phoneRegex = regexp.MustCompile(`^\+?(?:91)?[6-9]\d{9}$`)

// ValidatePhone validates an Indian phone number.
func ValidatePhone(phone string) bool {
	cleaned := ""
	for _, c := range phone {
		if (c >= '0' && c <= '9') || c == '+' {
			cleaned += string(c)
		}
	}
	return phoneRegex.MatchString(cleaned)
}
