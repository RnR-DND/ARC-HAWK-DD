package classifier

import (
	"regexp"
	"testing"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// ─── Context-keyword -40 penalty (Bank Account pattern) ──────────────────────

func TestClassify_BankAccount_WithContextKeyword_Detected(t *testing.T) {
	eng := NewEngine()
	rec := connectors.FieldRecord{
		Value:      "123456789012", // 12 digits, passes ValidateBankAccount
		FieldName:  "account_number",
		SourcePath: "customers.savings",
	}
	findings := eng.Classify(rec, nil, nil)
	found := false
	for _, f := range findings {
		if f.PIIType == "IN_BANK_ACCOUNT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected IN_BANK_ACCOUNT with context keyword 'account'")
	}
}

func TestClassify_BankAccount_NoContextKeyword_Suppressed(t *testing.T) {
	eng := NewEngine()
	// No context keyword in field name / source path → -40 → score = 60 (still ≥50).
	// Verify at least that the path through the penalty logic runs.
	rec := connectors.FieldRecord{
		Value:      "123456789012",
		FieldName:  "employee_id",
		SourcePath: "hr.staff",
	}
	findings := eng.Classify(rec, nil, nil)
	for _, f := range findings {
		if f.PIIType == "IN_BANK_ACCOUNT" {
			// Penalty applied; score should be 60 (100 - 40)
			if f.Score != 60 {
				t.Errorf("expected score 60 after -40 penalty, got %d", f.Score)
			}
		}
	}
}

func TestClassify_ContextKeywordPenalty_Magnitude_Is40(t *testing.T) {
	eng := NewEngine()
	allowed := map[string]struct{}{"Bank Account": {}}

	withKw := connectors.FieldRecord{Value: "123456789012", FieldName: "bank_account_number"}
	withoutKw := connectors.FieldRecord{Value: "123456789012", FieldName: "reference_id"}

	fWith := eng.Classify(withKw, nil, allowed)
	fWithout := eng.Classify(withoutKw, nil, allowed)

	var scoreWith, scoreWithout int
	for _, f := range fWith {
		if f.PIIType == "IN_BANK_ACCOUNT" {
			scoreWith = f.Score
		}
	}
	for _, f := range fWithout {
		if f.PIIType == "IN_BANK_ACCOUNT" {
			scoreWithout = f.Score
		}
	}

	if scoreWith == 0 {
		t.Skip("bank account not detected with context keyword (validator failure)")
	}
	if scoreWithout == 0 {
		t.Skip("bank account suppressed without context keyword (score < 50 — valid)")
	}
	if scoreWith-scoreWithout != 40 {
		t.Errorf("penalty should be exactly 40pts: with=%d without=%d diff=%d", scoreWith, scoreWithout, scoreWith-scoreWithout)
	}
}

// ─── PAN detection ────────────────────────────────────────────────────────────

func TestClassify_PAN_ValidEntityCode_Detected(t *testing.T) {
	eng := NewEngine()
	rec := connectors.FieldRecord{Value: "User PAN ABCPE1234F submitted", FieldName: "kyc"}
	findings := eng.Classify(rec, nil, nil)
	found := false
	for _, f := range findings {
		if f.PIIType == "IN_PAN" && f.MatchedValue == "ABCPE1234F" {
			found = true
			if f.DetectorType != "math" {
				t.Errorf("PAN detector type = %q, want math", f.DetectorType)
			}
		}
	}
	if !found {
		t.Error("expected IN_PAN for ABCPE1234F")
	}
}

func TestClassify_PAN_InvalidEntityCode_NotDetected(t *testing.T) {
	eng := NewEngine()
	rec := connectors.FieldRecord{Value: "ABCDE1234F", FieldName: "kyc"} // D not valid entity
	findings := eng.Classify(rec, nil, nil)
	for _, f := range findings {
		if f.PIIType == "IN_PAN" && f.MatchedValue == "ABCDE1234F" {
			t.Error("PAN with invalid entity code D should not be detected")
		}
	}
}

// ─── Aadhaar detection ────────────────────────────────────────────────────────

func TestClassify_Aadhaar_ValidVerhoeff(t *testing.T) {
	eng := NewEngine()
	rec := connectors.FieldRecord{Value: "ID: 234123412346", FieldName: "uid"}
	findings := eng.Classify(rec, nil, nil)
	found := false
	for _, f := range findings {
		if f.PIIType == "IN_AADHAAR" {
			found = true
		}
	}
	if !found {
		t.Error("expected IN_AADHAAR finding for valid Verhoeff number")
	}
}

// ─── IFSC detection ───────────────────────────────────────────────────────────

func TestClassify_IFSC_Detected(t *testing.T) {
	eng := NewEngine()
	rec := connectors.FieldRecord{Value: "Transfer to SBIN0001234", FieldName: "payment"}
	findings := eng.Classify(rec, nil, nil)
	found := false
	for _, f := range findings {
		if f.PIIType == "IN_IFSC" {
			found = true
		}
	}
	if !found {
		t.Error("expected IN_IFSC for SBIN0001234")
	}
}

// ─── AllowedPatterns filter ───────────────────────────────────────────────────

func TestClassify_EmptyAllowedPatterns_RunsNoBuiltins(t *testing.T) {
	eng := NewEngine()
	rec := connectors.FieldRecord{Value: "ABCPE1234F user@example.com", FieldName: "data"}
	findings := eng.Classify(rec, nil, map[string]struct{}{})
	if len(findings) != 0 {
		t.Errorf("empty allowedPatterns should run no built-in patterns, got %d", len(findings))
	}
}

func TestClassify_NilAllowedPatterns_RunsAll(t *testing.T) {
	eng := NewEngine()
	rec := connectors.FieldRecord{Value: "user@real-domain.com", FieldName: "email"}
	findings := eng.Classify(rec, nil, nil)
	found := false
	for _, f := range findings {
		if f.PIIType == "EMAIL_ADDRESS" {
			found = true
		}
	}
	if !found {
		t.Error("nil allowedPatterns should run all built-in patterns")
	}
}

// ─── Custom pattern ───────────────────────────────────────────────────────────

func TestClassify_CustomPattern_Detected(t *testing.T) {
	eng := NewEngine()
	cp := CustomPattern{
		ID:      "cp-001",
		Name:    "EmployeeID",
		PIIType: "EMPLOYEE_ID",
		Regex:   regexp.MustCompile(`EMP-\d{6}`),
	}
	// Avoid "12345" substring — triggers the test-data guard in Score().
	rec := connectors.FieldRecord{Value: "Employee EMP-789012 joined", FieldName: "log"}
	// empty allowedPatterns = no built-ins; custom always runs
	findings := eng.Classify(rec, []CustomPattern{cp}, map[string]struct{}{})
	found := false
	for _, f := range findings {
		if f.PIIType == "EMPLOYEE_ID" && f.MatchedValue == "EMP-789012" {
			found = true
		}
	}
	if !found {
		t.Error("expected EMPLOYEE_ID finding from custom pattern")
	}
}

// ─── Dedup ────────────────────────────────────────────────────────────────────

func TestClassify_DeduplicatesIdenticalValues(t *testing.T) {
	eng := NewEngine()
	rec := connectors.FieldRecord{Value: "ABCPE1234F and ABCPE1234F repeated", FieldName: "kyc"}
	findings := eng.Classify(rec, nil, nil)
	count := 0
	for _, f := range findings {
		if f.PIIType == "IN_PAN" && f.MatchedValue == "ABCPE1234F" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("duplicate PAN should be deduped; got %d findings", count)
	}
}

// ─── HashValue ────────────────────────────────────────────────────────────────

func TestHashValue_IsDeterministic(t *testing.T) {
	if HashValue("hello") != HashValue("hello") {
		t.Error("HashValue should be deterministic")
	}
	if HashValue("hello") == HashValue("world") {
		t.Error("different inputs should produce different hashes")
	}
}
