package validation

// ValidateLuhn checks if a number passes the Luhn algorithm (mod-10 checksum)
// Used for credit card validation
// Reference: https://en.wikipedia.org/wiki/Luhn_algorithm
func ValidateLuhn(number string) bool {
	if len(number) == 0 {
		return false
	}

	var sum int
	parity := len(number) % 2

	for i, digit := range number {
		if digit < '0' || digit > '9' {
			return false // Invalid character
		}

		d := int(digit - '0')

		// Double every second digit from right
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
