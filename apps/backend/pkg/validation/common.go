package validation

import "regexp"

// Email validation (RFC 5322 simplified)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail performs basic email format validation
func ValidateEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false // RFC 5321
	}

	return emailRegex.MatchString(email)
}

// Phone validation (international format)
var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{7,14}$`)

// ValidatePhone performs basic phone number format validation
// Expects digits only (no formatting), optionally starting with +
func ValidatePhone(phone string) bool {
	if len(phone) < 8 || len(phone) > 15 {
		return false
	}

	return phoneRegex.MatchString(phone)
}

// Indian phone number format (10 digits starting with 6-9)
var indianPhoneRegex = regexp.MustCompile(`^[6-9]\d{9}$`)

// ValidateIndianPhone validates Indian mobile phone numbers
func ValidateIndianPhone(phone string) bool {
	if len(phone) != 10 {
		return false
	}

	return indianPhoneRegex.MatchString(phone)
}

// US phone number format (10 digits, area code 2-9)
var usPhoneRegex = regexp.MustCompile(`^[2-9]\d{9}$`)

// ValidateUSPhone validates US phone numbers
func ValidateUSPhone(phone string) bool {
	if len(phone) != 10 {
		return false
	}

	return usPhoneRegex.MatchString(phone)
}
