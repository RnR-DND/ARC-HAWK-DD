package utils

// Fuzz targets for ScrubJSONLog and ScrubPII.
//
// Run with:
//
//	cd apps/backend && go test -fuzz=FuzzScrubJSONLog \
//	    ./modules/shared/utils/ -fuzztime=60s
//
//	cd apps/backend && go test -fuzz=FuzzScrubPII \
//	    ./modules/shared/utils/ -fuzztime=60s
//
// Risk surface:
//   - ScrubJSONLog calls json.Unmarshal into interface{} then recurses via
//     scrubValue(). Deeply nested JSON causes stack overflow (no depth limit).
//   - ScrubPII applies 8 regex replacements — adversarial strings can trigger
//     slow backtracking on phonePattern and aadhaarPattern.
//   - Neither function has a size limit on the input string.

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzScrubJSONLog checks that ScrubJSONLog never panics on arbitrary input.
// Key invariants:
//  1. Must return a non-empty string for any input.
//  2. Must not expose values whose keys are in sensitiveKeys.
//  3. Must not panic on deeply-nested or cyclically-invalid JSON.
func FuzzScrubJSONLog(f *testing.F) {
	seeds := []string{
		// Valid JSON with sensitive keys.
		`{"password":"hunter2","email":"alice@example.com"}`,
		`{"token":"eyJhbGci...","user":{"api_key":"sk-1234","data":"ok"}}`,
		// Deeply nested — 100 levels.
		strings.Repeat(`{"a":`, 100) + `"leaf"` + strings.Repeat(`}`, 100),
		// Array of sensitive objects.
		`[{"password":"x"},{"secret":"y"},{"api_key":"z"}]`,
		// Empty / trivial JSON.
		`{}`,
		`[]`,
		`null`,
		`""`,
		`"plain string"`,
		// Invalid JSON — must return "[invalid json]", not panic.
		`{not json`,
		`{"unclosed": [1,2,3`,
		``,
		"\x00",
		// Very large valid JSON (stress memory).
		`{"data":"` + strings.Repeat("x", 1<<16) + `"}`,
		// Numeric/boolean values.
		`{"score":42,"active":true,"ratio":3.14}`,
		// Mixed-case sensitive keys (should still redact).
		`{"Password":"x","TOKEN":"y","API_KEY":"z"}`,
		// Unicode keys and values.
		`{"пароль":"secret","用户":"alice@example.com"}`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip()
		}

		// Cap input to 256 KB to avoid OOM in fuzzing — documents the
		// missing size guard in the production function.
		if len(input) > 256*1024 {
			t.Skip()
		}

		result := ScrubJSONLog(input)

		// Invariant 1: must return non-empty.
		if result == "" {
			t.Error("ScrubJSONLog returned empty string")
		}

		// Invariant 2: result must not contain the literal value of known
		// sensitive keys' values when input was valid JSON with those keys.
		// We only check when the input contained "password" as a key to avoid
		// false positives on non-JSON or edge cases.
		if strings.Contains(strings.ToLower(input), `"password":`) {
			if strings.Contains(result, `"password":"hunter2"`) {
				t.Error("ScrubJSONLog did not redact password field")
			}
		}
	})
}

// FuzzScrubPII checks that ScrubPII never panics and always redacts
// well-known PII formats.
func FuzzScrubPII(f *testing.F) {
	seeds := []string{
		"contact alice@example.com or bob@test.co.in for info",
		"call me at 9876543210 or +91-9876543210",
		"aadhaar: 2345 6789 0123",
		"pan: ABCDE1234F",
		"card: 4111 1111 1111 1111",
		"upi: alice@okicici",
		"ifsc: HDFC0001234",
		"server at 192.168.1.100",
		"",
		strings.Repeat("9", 10000),
		"\x00\x01alice@example.com",
		strings.Repeat("a@b.c ", 10000),
		"no pii here at all",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip()
		}
		if len(input) > 512*1024 {
			t.Skip()
		}

		result := ScrubPII(input, &DefaultScrubConfig)

		if result == "" && input != "" {
			t.Error("ScrubPII returned empty string for non-empty input")
		}

		// Invariant: a known email in input must be replaced in output.
		if strings.Contains(input, "alice@example.com") {
			if strings.Contains(result, "alice@example.com") {
				t.Error("ScrubPII did not redact known email alice@example.com")
			}
		}
	})
}

// TestScrubJSONLog_DeeplyNested is a unit test that documents the stack
// depth limit issue: currently no limit, so 1000-level nesting will
// recurse 1000 levels deep via scrubValue(). This test verifies the
// function at least returns without panicking at moderate depth.
func TestScrubJSONLog_DeeplyNested(t *testing.T) {
	depth := 500
	nested := strings.Repeat(`{"child":`, depth) + `"leaf"` + strings.Repeat(`}`, depth)
	result := ScrubJSONLog(nested)
	if result == "" {
		t.Error("expected non-empty result for deeply nested JSON")
	}
}

// TestScrubJSONLog_SensitiveKeyRedaction verifies each key in sensitiveKeys
// is actually redacted.
func TestScrubJSONLog_SensitiveKeyRedaction(t *testing.T) {
	t.Parallel()
	cases := []struct {
		key   string
		value string
	}{
		{"password", "s3cr3t"},
		{"token", "eyJhbGci"},
		{"api_key", "sk-abc123"},
		{"secret", "topsecret"},
		{"access_token", "bearer-xyz"},
		{"encryption_key", "aes256key"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()
			input := `{"` + tc.key + `":"` + tc.value + `","safe":"visible"}`
			result := ScrubJSONLog(input)
			if strings.Contains(result, tc.value) {
				t.Errorf("key %q value %q not redacted in: %s", tc.key, tc.value, result)
			}
			if !strings.Contains(result, "visible") {
				t.Errorf("non-sensitive key 'safe' incorrectly redacted in: %s", result)
			}
		})
	}
}
