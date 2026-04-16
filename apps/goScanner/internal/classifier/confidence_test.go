package classifier

import (
	"testing"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// ── ShannonEntropy ──────────────────────────────────────────────────────────

func TestShannonEntropy_Empty(t *testing.T) {
	if ShannonEntropy("") != 0 {
		t.Error("ShannonEntropy(\"\") should be 0")
	}
}

func TestShannonEntropy_UniformString(t *testing.T) {
	if ShannonEntropy("aaaa") != 0 {
		t.Error("ShannonEntropy(\"aaaa\") should be 0 (all same char)")
	}
}

func TestShannonEntropy_HighEntropy(t *testing.T) {
	h := ShannonEntropy("aB3$xY9!mK2@")
	if h < 3.0 {
		t.Errorf("ShannonEntropy of mixed string = %f, want > 3.0", h)
	}
}

// ── Score ───────────────────────────────────────────────────────────────────

func TestScore_TestDataRejected(t *testing.T) {
	rec := connectors.FieldRecord{}
	score, _ := Score("test_value", "IN_PAN", rec, nil, nil)
	if score != 0 {
		t.Errorf("Score(\"test_value\") = %d, want 0 (test data should be rejected)", score)
	}
}

func TestScore_ExampleDataRejected(t *testing.T) {
	rec := connectors.FieldRecord{}
	score, _ := Score("example@email.com", "EMAIL_ADDRESS", rec, nil, nil)
	if score != 0 {
		t.Errorf("Score(\"example@...\") = %d, want 0 (contains 'example')", score)
	}
}

func TestScore_ValidPAN(t *testing.T) {
	// ABCPE1234F — regex [A-Z]{5}[0-9]{4}[A-Z] matches; entity pan[3]='P' is valid.
	rec := connectors.FieldRecord{}
	score, detType := Score("ABCPE1234F", "IN_PAN", rec, nil, nil)
	if score == 0 {
		t.Error("valid PAN \"ABCPE1234F\" should score > 0")
	}
	if detType != "math" {
		t.Errorf("valid PAN detector type = %q, want \"math\"", detType)
	}
}

func TestScore_InvalidPANEntity(t *testing.T) {
	// ABCDE1234F — pan[3]='D', which is NOT in validPANEntities.
	rec := connectors.FieldRecord{}
	score, _ := Score("ABCDE1234F", "IN_PAN", rec, nil, nil)
	if score != 0 {
		t.Errorf("PAN with invalid entity char (D) should score 0, got %d", score)
	}
}

func TestScore_InvalidPANFormat(t *testing.T) {
	// All lowercase — regex [A-Z]{5}... won't match.
	rec := connectors.FieldRecord{}
	score, _ := Score("abcpe1234f", "IN_PAN", rec, nil, nil)
	if score != 0 {
		t.Errorf("lowercase PAN should score 0, got %d", score)
	}
}

func TestScore_AadhaarSequentialRejected(t *testing.T) {
	// "123456789012" starts with '1' which is < '2', so ValidateAadhaar returns false.
	rec := connectors.FieldRecord{}
	score, _ := Score("123456789012", "IN_AADHAAR", rec, nil, nil)
	if score != 0 {
		t.Errorf("sequential Aadhaar \"123456789012\" should score 0, got %d", score)
	}
}

func TestScore_UnknownPIIType_HeuristicBase(t *testing.T) {
	// Type not in validatorMap → heuristic path, base score 50.
	rec := connectors.FieldRecord{}
	score, detType := Score("SomeValueHere", "CUSTOM_TYPE", rec, nil, nil)
	if score == 0 {
		t.Error("unknown PII type should receive heuristic score > 0")
	}
	if detType != "regex" {
		t.Errorf("unknown PII type detector = %q, want \"regex\"", detType)
	}
}

func TestScore_ContextKeywordBoost(t *testing.T) {
	// Unknown type: context keyword should raise score above base.
	rec := connectors.FieldRecord{RowContext: "user aadhaar number field"}
	scoreWith, _ := Score("SomeValue", "CUSTOM_TYPE", rec, []string{"aadhaar"}, nil)
	scoreWithout, _ := Score("SomeValue", "CUSTOM_TYPE", connectors.FieldRecord{}, nil, nil)
	if scoreWith <= scoreWithout {
		t.Errorf("context keyword should boost score: with=%d without=%d", scoreWith, scoreWithout)
	}
}

func TestScore_NegKeywordPenalty(t *testing.T) {
	// Unknown type: neg keyword should lower score.
	rec := connectors.FieldRecord{RowContext: "test dummy field"}
	// Use a value that won't hit the early test-data reject (no 'test' substring in value itself).
	scoreWith, _ := Score("SomeValue", "CUSTOM_TYPE", rec, nil, []string{"dummy"})
	scoreWithout, _ := Score("SomeValue", "CUSTOM_TYPE", connectors.FieldRecord{}, nil, nil)
	if scoreWith >= scoreWithout {
		t.Errorf("neg keyword should lower score: with=%d without=%d", scoreWith, scoreWithout)
	}
}
