package service

// Fuzz target for validateRegexSafety.
//
// Run with:
//
//	cd apps/backend && go test -fuzz=FuzzValidateRegexSafety \
//	    ./modules/scanning/service/ -fuzztime=60s
//
// Findings: the function must never panic and must return in <3s for any input.
// ReDoS corpus seeds cover catastrophic backtracking patterns known to defeat
// NFA-based engines (Go uses RE2 so pure CPU exhaustion is the real risk here).

import (
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzValidateRegexSafety checks that validateRegexSafety never panics and
// always returns within 3 seconds regardless of input.
func FuzzValidateRegexSafety(f *testing.F) {
	// Seed corpus: known ReDoS patterns and edge cases.
	seeds := []string{
		// Catastrophic backtracking for NFA engines (Go RE2 handles these, but
		// confirms timeout guard fires on degenerate input).
		`(a+)+`,
		`(a|aa)+`,
		`(a|a?)+`,
		`([a-zA-Z]+)*`,
		`(a+a+)+b`,
		`(.*a){20}`,
		`(\w+\s*\w+)+$`,
		// Nested quantifiers in various forms.
		`([0-9]+)+\s`,
		`((a{1,3})b){4}`,
		`(a{2,5}){3,8}`,
		// Unbounded alternation in quantifier.
		`(foo|bar|baz)+`,
		`(0|[1-9][0-9]*)+`,
		// Length boundary: 500-char pattern.
		string(make([]byte, 501)),
		// Empty string.
		``,
		// Valid common patterns.
		`^\d{10}$`,
		`[A-Z]{5}[0-9]{4}[A-Z]`,
		`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`,
		// Unicode edge cases.
		`[\x{0000}-\x{FFFF}]+`,
		`\p{L}+`,
		// Null bytes and control characters.
		"\x00\x01\x02",
		// Deeply nested groups (stack overflow risk in some engines).
		`((((((((((a))))))))))`,
		`(?(1)a|b)`,
		// Backreference (invalid in RE2 — should return error, not panic).
		`(a)\1`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, pattern string) {
		// Skip non-UTF-8 — the function works on Go strings which must be valid.
		if !utf8.ValidString(pattern) {
			t.Skip()
		}

		done := make(chan struct{})
		var panicked bool

		go func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
				close(done)
			}()
			// We don't care about the return value — only that it doesn't panic
			// and completes within the timeout.
			_ = validateRegexSafety(pattern)
		}()

		select {
		case <-done:
			if panicked {
				t.Errorf("validateRegexSafety panicked on input %q", pattern)
			}
		case <-time.After(3 * time.Second):
			t.Errorf("validateRegexSafety did not return within 3s for input %q", pattern)
		}
	})
}

// TestValidateRegexSafety_KnownReDoS verifies that known ReDoS patterns are
// rejected by the static analysis layer (not just the runtime timeout).
func TestValidateRegexSafety_KnownReDoS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pattern   string
		wantError bool
		reason    string
	}{
		{`(a+)+`, true, "nested quantifier"},
		{`(a|a?)+`, true, "nested quantifier with alternation"},
		{`([a-zA-Z]+)*`, true, "nested quantifier"},
		{`(foo|bar)+`, true, "unbounded alternation in quantifier"},
		{string(make([]byte, 501)), true, "exceeds 500 char limit"},
		// Valid patterns must pass.
		{`^\d{10}$`, false, "valid phone pattern"},
		{`[A-Z]{5}[0-9]{4}[A-Z]`, false, "valid PAN pattern"},
		{`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`, false, "valid email"},
		{``, false, "empty pattern is syntactically valid"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.reason, func(t *testing.T) {
			t.Parallel()
			err := validateRegexSafety(tc.pattern)
			if tc.wantError && err == nil {
				t.Errorf("expected error for %q (%s), got nil", tc.pattern, tc.reason)
			}
			if !tc.wantError && err != nil {
				t.Errorf("unexpected error for %q (%s): %v", tc.pattern, tc.reason, err)
			}
		})
	}
}
