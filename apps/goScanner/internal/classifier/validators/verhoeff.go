package validators

var verhoeffD = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 2, 3, 4, 0, 6, 7, 8, 9, 5},
	{2, 3, 4, 0, 1, 7, 8, 9, 5, 6},
	{3, 4, 0, 1, 2, 8, 9, 5, 6, 7},
	{4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
	{5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
	{6, 5, 9, 8, 7, 1, 0, 4, 3, 2},
	{7, 6, 5, 9, 8, 2, 1, 0, 4, 3},
	{8, 7, 6, 5, 9, 3, 2, 1, 0, 4},
	{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

var verhoeffP = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
	{5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
	{8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
	{9, 4, 5, 3, 1, 2, 6, 8, 7, 0},
	{4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
	{2, 7, 9, 3, 8, 0, 6, 4, 1, 5},
	{7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}

// ValidateAadhaar validates an Aadhaar number using the Verhoeff algorithm.
func ValidateAadhaar(number string) bool {
	cleaned := ""
	for _, c := range number {
		if c >= '0' && c <= '9' {
			cleaned += string(c)
		}
	}
	if len(cleaned) != 12 {
		return false
	}
	if cleaned[0] < '2' {
		return false
	}
	c := 0
	for i := len(cleaned) - 1; i >= 0; i-- {
		digit := int(cleaned[i] - '0')
		pos := (len(cleaned) - 1 - i) % 8
		c = verhoeffD[c][verhoeffP[pos][digit]]
	}
	return c == 0
}
