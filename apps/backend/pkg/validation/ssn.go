package validation

import "regexp"

// US SSN format: 9 digits (with or without dashes)
var ssnRegex = regexp.MustCompile(`^\d{9}$`)

// Blacklist of invalid SSN patterns
var ssnBlacklist = map[string]bool{
	// Known invalid patterns
	"000000000": true,
	"111111111": true,
	"222222222": true,
	"333333333": true,
	"444444444": true,
	"555555555": true,
	"666666666": true,
	"777777777": true,
	"888888888": true,
	"999999999": true,
	"123456789": true,
	"987654321": true,
}

// ValidateSSN checks if a string is a valid US SSN
// Expects 9 digits without dashes (normalization should happen before calling this)
func ValidateSSN(ssn string) bool {
	if len(ssn) != 9 {
		return false
	}

	if !ssnRegex.MatchString(ssn) {
		return false
	}

	// Check blacklist
	if ssnBlacklist[ssn] {
		return false
	}

	// SSA (Social Security Administration) rules:
	// - First 3 digits (area number) cannot be 000 or 666
	// - Middle 2 digits (group number) cannot be 00
	// - Last 4 digits (serial number) cannot be 0000

	areaNumber := ssn[0:3]
	groupNumber := ssn[3:5]
	serialNumber := ssn[5:9]

	if areaNumber == "000" || areaNumber == "666" {
		return false
	}

	if groupNumber == "00" {
		return false
	}

	if serialNumber == "0000" {
		return false
	}

	// Area numbers 900-999 are reserved for IRS use (ITIN)
	if ssn[0] == '9' {
		return false
	}

	return true
}
