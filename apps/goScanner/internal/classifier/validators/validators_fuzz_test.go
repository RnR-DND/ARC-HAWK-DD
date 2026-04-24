package validators

import "testing"

// Run with:
//
//	cd apps/goScanner && go test -fuzz=FuzzValidateAadhaar ./internal/classifier/validators/ -fuzztime=30s
//	cd apps/goScanner && go test -fuzz=FuzzValidatePAN ./internal/classifier/validators/ -fuzztime=30s
//	cd apps/goScanner && go test -fuzz=FuzzValidateVoterID ./internal/classifier/validators/ -fuzztime=30s
//	cd apps/goScanner && go test -fuzz=FuzzValidateIFSC ./internal/classifier/validators/ -fuzztime=30s
//	cd apps/goScanner && go test -fuzz=FuzzValidatePassport ./internal/classifier/validators/ -fuzztime=30s

func FuzzValidateAadhaar(f *testing.F) {
	// Valid Aadhaar numbers (Verhoeff checksum correct, starts with 2-9)
	f.Add("234123412346")
	f.Add("234123412346")
	// Invalid: too short, too long, starts with 0/1, non-digits
	f.Add("")
	f.Add("0")
	f.Add("123456789012")
	f.Add("000000000000")
	f.Add("abcdefghijkl")
	f.Add("2341 2341 2346")
	f.Add("99999999999999")
	f.Add("\x00\xff\n\t")

	f.Fuzz(func(t *testing.T, input string) {
		// Must never panic regardless of input.
		_ = ValidateAadhaar(input)
	})
}

func FuzzValidatePAN(f *testing.F) {
	// Valid PAN: 5 alpha + 4 digits + 1 alpha, 4th char is entity code
	f.Add("ABCPE1234F")
	f.Add("AAAPL1234C")
	f.Add("BBBPG5678H")
	// Invalid
	f.Add("")
	f.Add("abcpe1234f")
	f.Add("ABCQE1234F") // invalid entity code at pos 3
	f.Add("ABCPE123")
	f.Add("ABCPE12345F")
	f.Add("12345678901")
	f.Add("\x00\xff")
	f.Add("ABCPE1234\n")

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidatePAN(input)
		// If valid, must be exactly 10 chars and match structure.
		if result && len(input) != 10 {
			t.Errorf("ValidatePAN returned true for len=%d input %q", len(input), input)
		}
	})
}

func FuzzValidateVoterID(f *testing.F) {
	// Valid: 3 uppercase alpha + 7 digits
	f.Add("ABC1234567")
	f.Add("XYZ0000001")
	f.Add("AAA9999999")
	// Invalid
	f.Add("")
	f.Add("abc1234567")
	f.Add("AB1234567")
	f.Add("ABCD234567")
	f.Add("ABC123456")
	f.Add("ABC12345678")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidateVoterID(input)
		if result && len(input) != 10 {
			t.Errorf("ValidateVoterID returned true for len=%d input %q", len(input), input)
		}
	})
}

func FuzzValidateIFSC(f *testing.F) {
	// Valid: 4 uppercase alpha + '0' + 6 alphanumeric
	f.Add("SBIN0001234")
	f.Add("HDFC0000001")
	f.Add("ICIC0AB1234")
	// Invalid
	f.Add("")
	f.Add("sbin0001234")
	f.Add("SBI0001234")
	f.Add("SBIN1001234") // 5th char not '0'
	f.Add("SBIN000123")
	f.Add("SBIN00012345")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidateIFSC(input)
		if result && len(input) != 11 {
			t.Errorf("ValidateIFSC returned true for len=%d input %q", len(input), input)
		}
		if result && input[4] != '0' {
			t.Errorf("ValidateIFSC returned true but 5th char is %q, not '0'", input[4])
		}
	})
}

func FuzzValidatePassport(f *testing.F) {
	// Valid: 1 uppercase alpha + 7 digits
	f.Add("A1234567")
	f.Add("Z9999999")
	f.Add("M0000001")
	// Invalid
	f.Add("")
	f.Add("a1234567")
	f.Add("12345678")
	f.Add("A123456")
	f.Add("A12345678")
	f.Add("AA1234567")
	f.Add("\x00\xff")

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidatePassport(input)
		if result && len(input) != 8 {
			t.Errorf("ValidatePassport returned true for len=%d input %q", len(input), input)
		}
	})
}
