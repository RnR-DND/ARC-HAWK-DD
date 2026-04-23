package validators

import "testing"

// ─── Aadhaar ──────────────────────────────────────────────────────────────────

func TestValidateAadhaar(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid: starts with 2-9, 12 digits, passes Verhoeff
		{"valid_plain", "234123412346", true},
		// Valid with spaces
		{"valid_spaces", "2341 2341 2346", true},
		// Valid with hyphens
		{"valid_hyphens", "2341-2341-2346", true},
		// Invalid: starts with 0
		{"starts_with_0", "034123412346", false},
		// Invalid: starts with 1
		{"starts_with_1", "134123412346", false},
		// Invalid: only 11 digits
		{"too_short", "23412341234", false},
		// Invalid: 13 digits
		{"too_long", "2341234123456", false},
		// Invalid: all zeros (starts with 0)
		{"all_zeros", "000000000000", false},
		// Invalid: letters mixed in
		{"contains_letters", "2341234ABCDE", false},
		// Invalid: wrong Verhoeff check digit
		{"bad_verhoeff", "234123412340", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateAadhaar(tc.input)
			if got != tc.want {
				t.Errorf("ValidateAadhaar(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ─── PAN ─────────────────────────────────────────────────────────────────────

func TestValidatePAN(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid entity codes: P C H F A T B L J G
		{"valid_P", "ABCPE1234F", true},
		{"valid_C", "ABCCE1234F", true},
		{"valid_H", "ABCHE1234F", true},
		{"valid_F", "ABCFE1234F", true},
		{"valid_A", "ABCAE1234F", true},
		{"valid_T", "ABCTE1234F", true},
		// Invalid entity code (4th char D is not in valid set)
		{"invalid_entity_D", "ABCDE1234F", false},
		// Invalid entity code Q
		{"invalid_entity_Q", "ABCQE1234F", false},
		// Wrong length (9 chars)
		{"too_short", "ABCPE123F", false},
		// Wrong length (11 chars)
		{"too_long", "ABCPE12345F", false},
		// All lowercase
		{"lowercase", "abcpe1234f", false},
		// Mixed case
		{"mixed_case", "abcPE1234F", false},
		// Digits where letters expected
		{"wrong_format", "12345E1234F", false},
		// Letters where digits expected
		{"letters_in_digits", "ABCPEABCDF", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidatePAN(tc.input)
			if got != tc.want {
				t.Errorf("ValidatePAN(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ─── Voter ID ─────────────────────────────────────────────────────────────────

func TestValidateVoterID(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid", "ABC1234567", true},
		{"valid_2", "XYZ9876543", true},
		// Only 2 uppercase letters
		{"only_two_letters", "AB1234567", false},
		// 4 letters
		{"four_letters", "ABCD234567", false},
		// Lowercase letters
		{"lowercase", "abc1234567", false},
		// Only 6 digits
		{"six_digits", "ABC123456", false},
		// 8 digits
		{"eight_digits", "ABC12345678", false},
		// Empty
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateVoterID(tc.input)
			if got != tc.want {
				t.Errorf("ValidateVoterID(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ─── Passport ────────────────────────────────────────────────────────────────

func TestValidatePassport(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid_A", "A1234567", true},
		{"valid_Z", "Z9999999", true},
		{"valid_P", "P0000001", true},
		// Lowercase first letter
		{"lowercase_first", "a1234567", false},
		// Two letters
		{"two_letters", "AB123456", false},
		// Only 6 digits
		{"six_digits", "A123456", false},
		// 8 digits
		{"eight_digits", "A12345678", false},
		// Letters in digit section
		{"letters_in_digits", "A123456X", false},
		// Empty
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidatePassport(tc.input)
			if got != tc.want {
				t.Errorf("ValidatePassport(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ─── IFSC ─────────────────────────────────────────────────────────────────────

func TestValidateIFSC(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		// Format: 4 uppercase + '0' + 6 alphanumeric
		{"valid_sbi", "SBIN0001234", true},
		{"valid_hdfc", "HDFC0001234", true},
		{"valid_alpha_suffix", "ICIC0ABCDEF", true},
		// 5th char not '0'
		{"fifth_not_zero", "SBIN1001234", false},
		// Only 3 bank code letters
		{"short_bank_code", "SBI0001234", false},
		// Bank code has digit
		{"digit_in_bank", "SBI10001234", false},
		// Lowercase bank code
		{"lowercase", "sbin0001234", false},
		// Too short
		{"too_short", "SBIN000123", false},
		// Too long
		{"too_long", "SBIN00012345", false},
		// Empty
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateIFSC(tc.input)
			if got != tc.want {
				t.Errorf("ValidateIFSC(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
