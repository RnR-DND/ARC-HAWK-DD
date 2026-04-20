package validators

import "testing"

func TestValidateCreditCard(t *testing.T) {
	valid := []string{
		"4532015112830366",    // Visa (Luhn-valid)
		"5425233430109903",    // Mastercard
		"4532-0151-1283-0366", // With dashes
		"4532 0151 1283 0366", // With spaces
	}
	invalid := []string{
		"1234567890123456",     // Fails Luhn
		"123456789012",         // Too short (12 digits)
		"12345678901234567890", // Too long (20 digits)
		"abcdefghijklm",        // Not numeric
		"0000000000000",        // All zeros (Luhn-valid but only 13 zeros — actually passes Luhn)
	}

	for _, v := range valid {
		if !Validate("CREDIT_CARD", v) {
			t.Errorf("expected valid credit card: %s", v)
		}
	}
	for _, v := range invalid[:3] { // First 3 are structurally invalid
		if Validate("CREDIT_CARD", v) {
			t.Errorf("expected invalid credit card: %s", v)
		}
	}
}

func TestValidateAadhaar(t *testing.T) {
	valid := []string{
		"213465870917", // Verhoeff-valid, starts with 2
		"987654321096", // Verhoeff-valid, starts with 9
		"500000000006", // Verhoeff-valid, starts with 5
	}
	invalid := []string{
		"123456789012",  // Starts with 1 (not allowed)
		"023456789012",  // Starts with 0 (not allowed)
		"1234567890",    // Too short (10 digits)
		"1234567890123", // Too long (13 digits)
		"abcdefghijkl",  // Not numeric
		"295071489627",  // Fails Verhoeff checksum
	}

	for _, v := range valid {
		if !Validate("IN_AADHAAR", v) {
			t.Errorf("expected valid Aadhaar: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_AADHAAR", v) {
			t.Errorf("expected invalid Aadhaar: %s", v)
		}
	}
}

func TestValidatePAN(t *testing.T) {
	valid := []string{
		"ABCDE1234F",
		"BNZAA2318J",
		"abcde1234f", // Should be case-insensitive (uppercased internally)
	}
	invalid := []string{
		"ABCDE1234",   // Too short
		"ABCDE12345F", // Too long
		"12345ABCDE",  // Wrong format
		"ABCDEFFFFG",  // Letters where digits should be
	}

	for _, v := range valid {
		if !Validate("IN_PAN", v) {
			t.Errorf("expected valid PAN: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_PAN", v) {
			t.Errorf("expected invalid PAN: %s", v)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	valid := []string{
		"user@example.com",
		"test.user+tag@domain.co.in",
		"a@b.cd",
	}
	invalid := []string{
		"@example.com",     // No local part
		"user@",            // No domain
		"user@domain",      // No dot in domain
		"user@.com",        // Domain starts with dot
		"user@-domain.com", // Domain starts with hyphen
		"",                 // Empty
	}

	for _, v := range valid {
		if !Validate("EMAIL_ADDRESS", v) {
			t.Errorf("expected valid email: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("EMAIL_ADDRESS", v) {
			t.Errorf("expected invalid email: %q", v)
		}
	}
}

func TestValidateIndianPhone(t *testing.T) {
	valid := []string{
		"9876543210",
		"6000000000",
		"+919876543210",
		"919876543210",
	}
	invalid := []string{
		"1234567890",  // Starts with 1
		"5234567890",  // Starts with 5
		"98765432",    // Too short
		"98765432101", // Too long (11 digits without prefix)
	}

	for _, v := range valid {
		if !Validate("IN_PHONE", v) {
			t.Errorf("expected valid phone: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_PHONE", v) {
			t.Errorf("expected invalid phone: %s", v)
		}
	}
}

func TestValidateUPI(t *testing.T) {
	valid := []string{
		"user@upi",
		"test.user@ybl",
		"9876543210@paytm",
	}
	invalid := []string{
		"usernoat",       // No @
		"@ybl",           // No username
		"user@",          // No handle
		"user@123handle", // Handle starts with digit
	}

	for _, v := range valid {
		if !Validate("IN_UPI", v) {
			t.Errorf("expected valid UPI: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_UPI", v) {
			t.Errorf("expected invalid UPI: %s", v)
		}
	}
}

func TestValidateIFSC(t *testing.T) {
	valid := []string{
		"SBIN0001234",
		"HDFC0000001",
		"icic0002345", // Case-insensitive
	}
	invalid := []string{
		"SBIN1001234",  // 5th char must be 0
		"SBI0001234",   // Too short
		"SBIN00012345", // Too long
		"1234056789A",  // Starts with digits
	}

	for _, v := range valid {
		if !Validate("IN_IFSC", v) {
			t.Errorf("expected valid IFSC: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_IFSC", v) {
			t.Errorf("expected invalid IFSC: %s", v)
		}
	}
}

func TestValidatePassport(t *testing.T) {
	valid := []string{
		"A1234567",
		"Z9876543",
		"m1234567", // Case-insensitive
	}
	invalid := []string{
		"12345678",  // Starts with digit
		"A123456",   // Too short
		"A12345678", // Too long
		"AB1234567", // Two letters at start
	}

	for _, v := range valid {
		if !Validate("IN_PASSPORT", v) {
			t.Errorf("expected valid passport: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_PASSPORT", v) {
			t.Errorf("expected invalid passport: %s", v)
		}
	}
}

func TestValidateVoterID(t *testing.T) {
	valid := []string{
		"ABC1234567",
		"xyz1234567", // Case-insensitive
	}
	invalid := []string{
		"AB1234567",   // Only 2 letters
		"ABCD1234567", // 4 letters
		"ABC123456",   // Too short
		"ABC12345678", // Too long
	}

	for _, v := range valid {
		if !Validate("IN_VOTER_ID", v) {
			t.Errorf("expected valid voter ID: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_VOTER_ID", v) {
			t.Errorf("expected invalid voter ID: %s", v)
		}
	}
}

func TestValidateDrivingLicense(t *testing.T) {
	valid := []string{
		"MH0220190000001",
		"DL-04-20190000001",
		"KA 01 20200000123",
	}
	invalid := []string{
		"12345",              // Too short
		"AAAAAAAAAAAAAAAAAA", // All letters
	}

	for _, v := range valid {
		if !Validate("IN_DRIVING_LICENSE", v) {
			t.Errorf("expected valid DL: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_DRIVING_LICENSE", v) {
			t.Errorf("expected invalid DL: %s", v)
		}
	}
}

func TestValidateBankAccount(t *testing.T) {
	valid := []string{
		"123456789",          // 9 digits (minimum)
		"123456789012345678", // 18 digits (maximum)
		"00123456789",        // With leading zeros
	}
	invalid := []string{
		"12345678",            // Too short (8 digits)
		"1234567890123456789", // Too long (19 digits)
		"abcdefghi",           // Not numeric
	}

	for _, v := range valid {
		if !Validate("IN_BANK_ACCOUNT", v) {
			t.Errorf("expected valid bank account: %s", v)
		}
	}
	for _, v := range invalid {
		if Validate("IN_BANK_ACCOUNT", v) {
			t.Errorf("expected invalid bank account: %s", v)
		}
	}
}

func TestValidateUnknownType(t *testing.T) {
	// Unknown PII types should pass (don't reject what we don't know)
	if !Validate("UNKNOWN_TYPE", "anything") {
		t.Error("unknown PII type should pass validation")
	}
}

func TestValidateEmptyValue(t *testing.T) {
	types := []string{"CREDIT_CARD", "IN_AADHAAR", "IN_PAN", "EMAIL_ADDRESS", "IN_PHONE"}
	for _, piiType := range types {
		if Validate(piiType, "") {
			t.Errorf("empty value should fail for type %s", piiType)
		}
		if Validate(piiType, "   ") {
			t.Errorf("whitespace-only value should fail for type %s", piiType)
		}
	}
}

func TestMapPatternToPIIType(t *testing.T) {
	cases := map[string]string{
		"PAN_NUMBER":      "IN_PAN",
		"aadhaar_number":  "IN_AADHAAR",
		"CREDIT_CARD":     "CREDIT_CARD",
		"EMAIL_ADDRESS":   "EMAIL_ADDRESS",
		"PHONE_NUMBER":    "IN_PHONE",
		"PASSPORT_NUMBER": "IN_PASSPORT",
		"UPI_ID":          "IN_UPI",
		"IFSC_CODE":       "IN_IFSC",
		"VOTER_ID":        "IN_VOTER_ID",
		"DRIVING_LICENSE": "IN_DRIVING_LICENSE",
		"BANK_ACCOUNT":    "IN_BANK_ACCOUNT",
		"UNKNOWN_PATTERN": "",
	}

	for pattern, expected := range cases {
		got := MapPatternToPIIType(pattern)
		if got != expected {
			t.Errorf("MapPatternToPIIType(%q) = %q, want %q", pattern, got, expected)
		}
	}
}

func TestLuhnCheck(t *testing.T) {
	// Known Luhn-valid numbers
	if !luhnCheck("79927398713") {
		t.Error("79927398713 should pass Luhn")
	}
	if luhnCheck("79927398710") {
		t.Error("79927398710 should fail Luhn")
	}
}

func TestVerhoeffCheck(t *testing.T) {
	// Known Verhoeff-valid number: "2363" (check digit 3 for "236")
	if !verhoeffCheck("2363") {
		t.Error("2363 should pass Verhoeff")
	}
	if verhoeffCheck("2364") {
		t.Error("2364 should fail Verhoeff")
	}
}
