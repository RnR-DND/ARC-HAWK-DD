package saas

import "testing"

func TestValidSOQLIdent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"simple field", "Email", true},
		{"with underscore", "Custom_Field__c", true},
		{"starts with underscore", "_internal", true},
		{"digits after first char", "Field123", true},
		{"all letters mixed case", "MailingCity", true},

		{"empty string", "", false},
		{"starts with digit", "1Field", false},
		{"contains space", "First Name", false},
		{"contains quote", "Name'", false},
		{"contains semicolon", "Name;DROP", false},
		{"contains hyphen", "Full-Name", false},
		{"contains paren", "Name(1)", false},
		{"contains equals", "Name=X", false},
		{"too long (65 chars)", string(make([]byte, 65)), false},
	}

	// the all-zero-byte too-long case above would be rejected on "starts with digit/control"
	// so use a real long alphanumeric identifier to exercise the length branch.
	long := make([]byte, 65)
	for i := range long {
		long[i] = 'A'
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validSOQLIdent(tc.in); got != tc.want {
				t.Errorf("validSOQLIdent(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}

	if validSOQLIdent(string(long)) {
		t.Error("65-char all-letter identifier must be rejected by length gate")
	}
	ok := validSOQLIdent(string(long[:64]))
	if !ok {
		t.Error("64-char all-letter identifier should be accepted")
	}
}
