package classifier

// Fuzz target for Engine.Classify.
//
// Run with:
//
//	cd apps/goScanner && go test -fuzz=FuzzEngineClassify \
//	    ./internal/classifier/ -fuzztime=60s
//
// The fuzzer drives FieldRecord.Value with arbitrary byte sequences to find:
//   - Panics in regex matching (FindAllString with adversarial input)
//   - Panics inside Dedup (hash collisions, nil findings)
//   - Incorrect behaviour when Value contains null bytes or extremely long strings
//   - ContextExcerpt out-of-bounds slice (excerpt function)

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// FuzzEngineClassify drives random field values through the classification
// engine and asserts no panics occur.
func FuzzEngineClassify(f *testing.F) {
	engine := NewEngine()

	// Seed corpus: real-world PII values and adversarial edge cases.
	seeds := []string{
		// Real PII that should match built-in patterns.
		"alice@example.com",
		"user@subdomain.co.in",
		"9876543210",
		"+91-9876543210",
		"ABCDE1234F",                        // PAN card
		"2345 6789 0123",                    // Aadhaar
		"4111 1111 1111 1111",               // Visa test card
		"alice@okicici",                     // UPI
		"HDFC0001234",                       // IFSC
		"192.168.1.100",                     // IP
		// Edge cases.
		"",
		strings.Repeat("a", 1<<20),          // 1 MB value — no length guard currently
		strings.Repeat("@", 10000),
		"\x00\x01\x02\x03",
		"\n\r\t",
		`{"email":"x@x.com","ssn":"123-45-6789"}`, // JSON blob as a value
		// Boundary: value that looks like PII but just misses.
		"ABCD12345",    // PAN missing final letter
		"987654321",    // 9-digit phone (1 short)
		// Unicode.
		"用户@域名.中国",
		"ℌello wörld",
		// Null byte embedded.
		"test\x00email@example.com",
		// Repetition that stresses excerpt().
		strings.Repeat("alice@example.com ", 1000),
	}

	for _, s := range seeds {
		f.Add(s, "email_field", "schema.table.column")
	}

	// Pre-compile a benign custom pattern used in the fuzz loop.
	customRe := regexp.MustCompile(`TEST-\d{4}`)

	f.Fuzz(func(t *testing.T, value, fieldName, sourcePath string) {
		// We only test valid UTF-8; Go regex operates on runes.
		if !utf8.ValidString(value) {
			t.Skip()
		}

		record := connectors.FieldRecord{
			Value:      value,
			FieldName:  fieldName,
			SourcePath: sourcePath,
		}

		customs := []CustomPattern{
			{
				ID:      "fuzz-custom-1",
				Name:    "FuzzCustom",
				PIIType: "FUZZ_TYPE",
				Regex:   customRe,
				RawRegex: `TEST-\d{4}`,
			},
		}

		// Must not panic under any circumstances.
		findings := engine.Classify(record, customs, nil)

		// Post-condition: every finding must have non-empty PIIType and
		// non-empty MatchedValue (the regex matched something real).
		for i, f := range findings {
			if f.PIIType == "" {
				t.Errorf("finding[%d] has empty PIIType", i)
			}
			if f.MatchedValue == "" {
				t.Errorf("finding[%d] PIIType=%s has empty MatchedValue", i, f.PIIType)
			}
			// Score must be in valid range [0, 100].
			if f.Score < 0 || f.Score > 100 {
				t.Errorf("finding[%d] has out-of-range score %d", i, f.Score)
			}
			// ValueHash must be 64-char hex (SHA-256).
			if len(f.ValueHash) != 64 {
				t.Errorf("finding[%d] ValueHash len=%d, want 64", i, len(f.ValueHash))
			}
		}
	})
}

// FuzzClassifyAllowlist drives the allowedPatterns filter with random pattern
// names to ensure no panic when the allowlist contains unknown names.
func FuzzClassifyAllowlist(f *testing.F) {
	engine := NewEngine()

	f.Add("EMAIL", "PHONE")
	f.Add("", "NONEXISTENT_TYPE")
	f.Add("AADHAAR", "PAN_CARD")

	f.Fuzz(func(t *testing.T, p1, p2 string) {
		if !utf8.ValidString(p1) || !utf8.ValidString(p2) {
			t.Skip()
		}
		allowlist := map[string]struct{}{p1: {}, p2: {}}
		record := connectors.FieldRecord{
			Value:     "alice@example.com 9876543210",
			FieldName: "mixed_field",
		}
		_ = engine.Classify(record, nil, allowlist)
	})
}

// TestClassifyNilCustomRegex ensures Classify skips custom patterns with nil Regex
// (regression for the nil guard at engine.go line 92).
func TestClassifyNilCustomRegex(t *testing.T) {
	engine := NewEngine()
	record := connectors.FieldRecord{Value: "alice@example.com", FieldName: "email"}
	customs := []CustomPattern{
		{ID: "bad", Name: "BadCustom", PIIType: "X", Regex: nil},
	}
	findings := engine.Classify(record, customs, nil)
	// Should still return built-in findings; custom nil-regex entry skipped.
	if len(findings) == 0 {
		t.Error("expected at least one finding for known email value")
	}
}
