package validators

// ValidateCreditCard validates using the Luhn algorithm.
func ValidateCreditCard(number string) bool {
	cleaned := ""
	for _, c := range number {
		if c >= '0' && c <= '9' {
			cleaned += string(c)
		}
	}
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false
	}
	sum := 0
	nDigits := len(cleaned)
	parity := nDigits % 2
	for i := 0; i < nDigits; i++ {
		digit := int(cleaned[i] - '0')
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}
