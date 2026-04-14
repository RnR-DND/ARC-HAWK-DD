package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/arc-platform/backend/modules/shared/config"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

// ==================================================================================
// LOCKED PII SCOPE - Intelligence-at-Edge Architecture
// ==================================================================================
// Only these 12 India PII types are in scope. All others MUST be rejected.
// Language: English only
// ==================================================================================
var LOCKED_PII_TYPES = map[string]bool{
	"IN_PAN":             true, // Permanent Account Number
	"IN_PASSPORT":        true, // Indian Passport Number
	"IN_AADHAAR":         true, // Aadhaar (UID)
	"CREDIT_CARD":        true, // Credit/Debit Card
	"IN_UPI":             true, // UPI ID
	"IN_IFSC":            true, // IFSC Code
	"IN_BANK_ACCOUNT":    true, // Bank Account Number
	"IN_PHONE":           true, // Indian Phone (10 digit)
	"EMAIL_ADDRESS":      true, // Email
	"IN_VOTER_ID":        true, // Voter ID (EPIC)
	"IN_DRIVING_LICENSE": true, // Driving License (India)
	"IN_GST":             true, // GST Identification Number (GSTIN)
}

// IsLockedPIIType validates if a PII type is in the locked scope.
// Also allows CUSTOM_* and USR_* types from user-defined patterns, with suffix validation.
func IsLockedPIIType(piiType string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(piiType))
	if LOCKED_PII_TYPES[normalized] {
		return true
	}
	// Validate user-defined custom patterns (prefixed with CUSTOM_ or USR_)
	// Require a non-empty, well-formed suffix (no whitespace or control chars)
	if strings.HasPrefix(normalized, "CUSTOM_") || strings.HasPrefix(normalized, "USR_") {
		suffix := strings.TrimPrefix(strings.TrimPrefix(normalized, "CUSTOM_"), "USR_")
		return len(suffix) > 0 && !strings.ContainsAny(suffix, " \t\n")
	}
	return false
}

// ClassificationService handles PII classification with multi-signal intelligence
type ClassificationService struct {
	repo          *persistence.PostgresRepository
	config        *config.Config
	engineVersion string
}

// NewClassificationService creates a new classification service
func NewClassificationService(repo *persistence.PostgresRepository, cfg *config.Config) *ClassificationService {
	// MEDIUM FIX #12: Load version from environment
	version := os.Getenv("CLASSIFIER_VERSION")
	if version == "" {
		version = "v2.0-multisignal"
	}

	return &ClassificationService{
		repo:          repo,
		config:        cfg,
		engineVersion: version,
	}
}

// ClassificationResult is the legacy result format for backward compatibility
type ClassificationResult struct {
	ClassificationType string                 `json:"classification_type"`
	SubCategory        string                 `json:"sub_category"`
	Justification      string                 `json:"justification"`
	ConfidenceScore    float64                `json:"confidence_score"`
	Signals            map[string]interface{} `json:"signals"`
	DPDPACategory      string                 `json:"dpdpa_category"`
	RequiresConsent    bool                   `json:"requires_consent"`
}

// MultiSignalInput contains all inputs for multi-signal classification
type MultiSignalInput struct {
	PatternName       string
	FilePath          string
	MatchValue        string
	ColumnName        string
	FileData          map[string]interface{}
	EnrichmentScore   float64 // From enrichment layer
	EnrichmentSignals EnrichmentSignals
}

// SignalScore represents a single signal's contribution
type SignalScore struct {
	RawScore      float64 `json:"raw_score"`
	WeightedScore float64 `json:"weighted_score"`
	Weight        float64 `json:"weight"`
	Confidence    float64 `json:"confidence"`
	Explanation   string  `json:"explanation"`
}

// MultiSignalDecision contains the final classification with full explainability
type MultiSignalDecision struct {
	// Final Decision
	Classification  string  `json:"classification"`
	SubCategory     string  `json:"sub_category"`
	FinalScore      float64 `json:"final_score"`
	ConfidenceLevel string  `json:"confidence_level"` // Confirmed, High, NeedsReview, Discard
	Justification   string  `json:"justification"`

	// Signal Breakdown
	RuleSignal     SignalScore `json:"rule_signal"`
	PresidioSignal SignalScore `json:"presidio_signal"`
	ContextSignal  SignalScore `json:"context_signal"`
	EntropySignal  SignalScore `json:"entropy_signal"`

	// Metadata
	EngineVersion   string                 `json:"engine_version"`
	DPDPACategory   string                 `json:"dpdpa_category"`
	RequiresConsent bool                   `json:"requires_consent"`
	SignalBreakdown map[string]interface{} `json:"signal_breakdown"`
}

// Confidence thresholds (Use config where possible, but mapping strings to levels can remain for now)
const (
	ThresholdConfirmed   = 0.85
	ThresholdHigh        = 0.65
	ThresholdNeedsReview = 0.45
)

// ClassifyMultiSignal performs gate-based classification with deterministic validation
// ARCHITECTURE: Detection → Validation (GATE) → Enrichment → Classification
func (s *ClassificationService) ClassifyMultiSignal(ctx context.Context, input MultiSignalInput) (*MultiSignalDecision, error) {
	decision := &MultiSignalDecision{
		EngineVersion:   s.engineVersion,
		SignalBreakdown: make(map[string]interface{}),
	}

	// STAGE 1: Rule-Based Entity Type Detection
	ruleSignal := s.classifyWithRules(input)
	decision.RuleSignal = ruleSignal

	// STAGE 2: Presidio ML - REMOVED (now handled by scanner SDK)
	// Backend trusts scanner's verified findings
	presidioSignal := SignalScore{
		RawScore:      0.0,
		WeightedScore: 0.0,
		Weight:        0.0,
		Confidence:    0.0,
		Explanation:   "Presidio handled by scanner SDK (Intelligence-at-Edge)",
	}
	decision.PresidioSignal = presidioSignal

	// STAGE 3: VALIDATION GATE - REMOVED (Intelligence-at-Edge)
	// ========================================================
	// Backend NO LONGER validates findings - this is now handled exclusively
	// by the scanner SDK before findings are sent to backend.
	// Scanner SDK performs:
	//   - Presidio ML analysis (embedded)
	//   - Mathematical validation (Luhn, Verhoeff, PAN format)
	//   - Context extraction
	// Backend receives ONLY verified findings with proof of validation.
	// ========================================================

	// STAGE 4: Enrichment (ONLY for validated findings)
	contextSignal := s.classifyWithContext(input)
	decision.ContextSignal = contextSignal

	entropySignal := s.classifyWithEntropy(input)
	decision.EntropySignal = entropySignal

	// STAGE 5: Confidence Tier Assignment (NOT probabilistic scoring)
	// All validated findings have FinalScore = 1.0 (binary: validated or not)
	// Confidence tier is based on enrichment, not validation
	decision.FinalScore = 1.0
	decision.ConfidenceLevel = s.assignConfidenceTier(
		presidioSignal.Confidence,
		contextSignal.RawScore,
	)

	// STAGE 6: Classification Type Assignment (Trust-based)
	// Backend trusts scanner SDK - classify based on pattern name
	decision.Classification = s.extractClassificationFromPattern(input.PatternName)
	decision.SubCategory = s.extractSubCategory(decision.Classification)

	// Set DPDPA metadata
	s.setDPDPAMetadata(decision)

	// Build comprehensive justification
	decision.Justification = s.buildJustification(decision)

	// Store signal breakdown (for transparency, not scoring)
	decision.SignalBreakdown = map[string]interface{}{
		"rule":     ruleSignal,
		"presidio": presidioSignal,
		"context":  contextSignal,
		"entropy":  entropySignal,
		"validation": map[string]interface{}{
			"handled_by":        "scanner_sdk",
			"backend_validated": false,
			"note":              "Intelligence-at-Edge - validation in scanner only",
		},
	}

	return decision, nil
}

// assignConfidenceTier determines confidence tier based on enrichment signals
// Decision table from Phase 2 architecture
func (s *ClassificationService) assignConfidenceTier(presidioMLConf float64, contextScore float64) string {
	// TIER 1: CONFIRMED - High ML confidence + High-risk context
	if presidioMLConf > 0.80 && contextScore > 0.7 {
		return "CONFIRMED"
	}

	// TIER 2: HIGH_CONFIDENCE - Medium ML OR high context
	if presidioMLConf >= 0.60 || contextScore > 0.7 {
		return "HIGH_CONFIDENCE"
	}

	// TIER 3: VALIDATED - All validated findings that don't meet higher tiers
	return "VALIDATED"
}

// extractClassificationFromPattern maps pattern names to classification types
// Used when backend trusts scanner SDK verified findings
func (s *ClassificationService) extractClassificationFromPattern(patternName string) string {
	lower := strings.ToLower(patternName)

	// Sensitive Personal Data
	if containsStrict(lower, []string{"aadhaar", "aadhar", "pan", "pancard", "passport", "ssn", "social_security"}) {
		return "Sensitive Personal Data"
	}
	if containsStrict(lower, []string{"credit_card", "card", "debit_card"}) {
		return "Sensitive Personal Data"
	}

	// Personal Data
	if containsStrict(lower, []string{"email", "e-mail", "mail"}) {
		return "Personal Data"
	}
	if containsStrict(lower, []string{"phone", "mobile", "cellphone"}) {
		return "Personal Data"
	}

	// Business identifiers
	if containsStrict(lower, []string{"gst", "gstin", "tax_id", "vat"}) {
		return "Sensitive Personal Data"
	}

	// Secrets
	if containsStrict(lower, []string{"aws_key", "api_key", "auth_token", "private_key", "secret_key", "password", "token"}) {
		return "Secrets"
	}

	return "Non-PII"
}

// setDPDPAMetadata assigns DPDPA compliance metadata
func (s *ClassificationService) setDPDPAMetadata(decision *MultiSignalDecision) {
	switch decision.Classification {
	case "Sensitive Personal Data":
		decision.DPDPACategory = "Sensitive Personal Data"
		decision.RequiresConsent = true
	case "Personal Data":
		decision.DPDPACategory = "Personal Data"
		decision.RequiresConsent = true
	case "Secrets":
		decision.DPDPACategory = "N/A"
		decision.RequiresConsent = false
	default:
		decision.DPDPACategory = "N/A"
		decision.RequiresConsent = false
	}
}

// classifyWithRules performs rule-based classification (Primary signal)
func (s *ClassificationService) classifyWithRules(input MultiSignalInput) SignalScore {
	lowerPattern := strings.ToLower(input.PatternName)
	lowerPath := strings.ToLower(input.FilePath)
	lowerCol := strings.ToLower(input.ColumnName)

	score := 0.0
	explanation := ""

	// Secrets detection
	if containsStrict(lowerPattern, []string{"aws_key", "aws_secret", "api_key", "auth_token", "private_key", "secret_key", "password", "aws access key", "access key"}) ||
		containsStrict(lowerCol, []string{"password", "secret", "apikey", "token"}) {
		score = 0.95
		explanation = "Strong pattern match for credentials/secrets"
	} else if containsStrict(lowerPattern, []string{"email", "e-mail", "mail"}) || containsStrict(lowerCol, []string{"email", "e-mail"}) {
		score = 0.95
		explanation = "Email address pattern detected"
	} else if containsStrict(lowerPattern, []string{"pan", "pancard", "permanent_account_number"}) || containsStrict(lowerCol, []string{"pan", "pancard"}) {
		score = 0.99
		explanation = "PAN Card pattern detected"
	} else if containsStrict(lowerPattern, []string{"aadhaar", "uidai", "adhaar", "aadhar"}) {
		score = 0.99
		explanation = "Aadhaar pattern detected"
	} else if containsStrict(lowerPattern, []string{"phone", "mobile", "cellphone"}) || containsStrict(lowerCol, []string{"phone", "mobile"}) {
		score = 0.90
		explanation = "Phone number pattern detected"
	} else if containsStrict(lowerPattern, []string{"credit_card", "debit_card", "cvv", "card_number", "credit card", "card"}) {
		score = 0.95
		explanation = "Financial data pattern detected"
	} else {
		score = 0.30
		explanation = "No strong PII pattern matched"
	}

	// Context boost - HIGH FIX #8: Multiplicative instead of additive
	if containsStrict(lowerPath, []string{"user", "users", "customer", "customers", "billing", "auth", "login", "account"}) {
		score *= 1.05 // 5% multiplicative boost (naturally bounded)
		explanation += " + high-risk path context"
	}

	// Test data penalty
	if containsStrict(lowerPath, []string{"test", "tests", "fixture", "mock", "example", "spec"}) {
		score *= 0.8
		explanation += " (reduced: test data)"
	}

	// Clamp
	if score > 1.0 {
		score = 1.0
	}

	return SignalScore{
		RawScore:      score,
		WeightedScore: score * s.config.Classification.WeightRules,
		Weight:        s.config.Classification.WeightRules,
		Confidence:    score,
		Explanation:   fmt.Sprintf("Rules: %s", explanation),
	}
}

// classifyWithContext uses enrichment signals as context
func (s *ClassificationService) classifyWithContext(input MultiSignalInput) SignalScore {
	score := input.EnrichmentScore
	explanation := fmt.Sprintf("Context: Enrichment score %.2f (env: %s, semantics: %.2f)",
		score, input.EnrichmentSignals.Environment, input.EnrichmentSignals.AssetSemantics)

	return SignalScore{
		RawScore:      score,
		WeightedScore: score * s.config.Classification.WeightContext,
		Weight:        s.config.Classification.WeightContext,
		Confidence:    score,
		Explanation:   explanation,
	}
}

// classifyWithEntropy uses statistical analysis
// HIGH FIX #7: Entropy only applies to secrets/tokens/API keys
func (s *ClassificationService) classifyWithEntropy(input MultiSignalInput) SignalScore {
	// Only apply entropy to secrets/tokens
	if !isSecretPattern(input.PatternName) {
		return SignalScore{
			RawScore:      0.0,
			WeightedScore: 0.0,
			Weight:        s.config.Classification.WeightEntropy,
			Confidence:    0.0,
			Explanation:   "Entropy N/A for non-secrets",
		}
	}

	entropy := input.EnrichmentSignals.Entropy
	diversity := input.EnrichmentSignals.CharsetDiversity

	// Normalize entropy (typical range 0-5)
	normEntropy := math.Min(entropy/5.0, 1.0)

	// Combine entropy and diversity
	score := (normEntropy * 0.7) + (diversity * 0.3)

	explanation := fmt.Sprintf("Stats: Entropy %.2f, Diversity %.2f → Score %.2f",
		entropy, diversity, score)

	return SignalScore{
		RawScore:      score,
		WeightedScore: score * s.config.Classification.WeightEntropy,
		Weight:        s.config.Classification.WeightEntropy,
		Confidence:    score,
		Explanation:   explanation,
	}
}

// isSecretPattern determines if a pattern represents a secret/token
func isSecretPattern(patternName string) bool {
	secretKeywords := []string{"aws_key", "api_key", "auth_token", "private_key", "secret_key", "password", "token", "access_key"}
	lower := strings.ToLower(patternName)
	for _, kw := range secretKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// extractSubCategory determines sub-category
func (s *ClassificationService) extractSubCategory(classification string) string {
	switch classification {
	case "Secrets":
		return "API Keys & Secrets"
	case "Sensitive Personal Data":
		return "Government ID / Financial Data"
	case "Personal Data":
		return "Contact Information"
	default:
		return "Other"
	}
}

// buildJustification creates human-readable explanation
func (s *ClassificationService) buildJustification(decision *MultiSignalDecision) string {
	return fmt.Sprintf("%s (Score: %.2f, Level: %s). Signals: %s | %s | %s | %s",
		decision.Classification,
		decision.FinalScore,
		decision.ConfidenceLevel,
		decision.RuleSignal.Explanation,
		decision.PresidioSignal.Explanation,
		decision.ContextSignal.Explanation,
		decision.EntropySignal.Explanation,
	)
}

// containsStrict checks word boundaries
func containsStrict(text string, keywords []string) bool {
	for _, kw := range keywords {
		idx := strings.Index(text, kw)
		if idx == -1 {
			continue
		}

		// Check preceding char
		if idx > 0 {
			prev := text[idx-1]
			if (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z') {
				continue
			}
		}

		// Check succeeding char
		end := idx + len(kw)
		if end < len(text) {
			next := text[end]
			if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
				continue
			}
		}
		return true
	}
	return false
}

// Legacy Classify method for backward compatibility
func (s *ClassificationService) Classify(patternName, filePath, sampleText string, fileData map[string]interface{}) ClassificationResult {
	// For backward compatibility, use rule-based only
	// This will be phased out once multi-signal is fully integrated
	return s.classifyLegacy(patternName, filePath, sampleText, fileData)
}

func (s *ClassificationService) classifyLegacy(patternName, filePath, _ string, fileData map[string]interface{}) ClassificationResult {
	signals := map[string]interface{}{
		"pattern_match": true,
		"context_score": 0.0,
	}

	result := ClassificationResult{
		ClassificationType: "Non-PII",
		SubCategory:        "Other",
		ConfidenceScore:    0.5,
		Signals:            signals,
		RequiresConsent:    false,
	}

	lowerPattern := strings.ToLower(patternName)
	lowerPath := strings.ToLower(filePath)
	lowerCol := ""
	if colName, ok := fileData["column_name"].(string); ok {
		lowerCol = strings.ToLower(colName)
	}

	// Use same logic as rule-based signal
	if containsStrict(lowerPattern, []string{"aws_key", "aws_secret", "api_key", "auth_token", "private_key", "secret_key", "password"}) ||
		containsStrict(lowerCol, []string{"password", "secret", "apikey", "token"}) {
		result.ClassificationType = "Secrets"
		result.SubCategory = "API Keys & Secrets"
		result.DPDPACategory = "N/A"
		result.ConfidenceScore = 0.95
		result.Justification = "Strong pattern match for credentials"
	} else if containsStrict(lowerPattern, []string{"email", "e-mail", "mail"}) || containsStrict(lowerCol, []string{"email", "e-mail"}) {
		result.ClassificationType = "Personal Data"
		result.SubCategory = "Email Address"
		result.DPDPACategory = "Personal Data"
		result.ConfidenceScore = 0.95
		result.RequiresConsent = true
		result.Justification = "Pattern/Column indicates Email Address"
	} else if containsStrict(lowerPattern, []string{"pan", "pancard", "permanent_account_number"}) || containsStrict(lowerCol, []string{"pan", "pancard"}) {
		result.ClassificationType = "Sensitive Personal Data"
		result.SubCategory = "Government ID"
		result.DPDPACategory = "Sensitive Personal Data"
		result.ConfidenceScore = 0.99
		result.RequiresConsent = true
		result.Justification = "Pattern/Column indicates PAN Card"
	} else if containsStrict(lowerPattern, []string{"aadhaar", "uidai", "adhaar"}) {
		result.ClassificationType = "Sensitive Personal Data"
		result.SubCategory = "Government ID"
		result.DPDPACategory = "Sensitive Personal Data"
		result.ConfidenceScore = 0.99
		result.Justification = "Pattern indicates Aadhaar"
	} else if containsStrict(lowerPattern, []string{"phone", "mobile", "cellphone"}) || containsStrict(lowerCol, []string{"phone", "mobile"}) {
		result.ClassificationType = "Personal Data"
		result.SubCategory = "Phone Number"
		result.DPDPACategory = "Personal Data"
		result.ConfidenceScore = 0.90
		result.RequiresConsent = true
		result.Justification = "Pattern/Column indicates Phone Number"
	} else if containsStrict(lowerPattern, []string{"credit_card", "debit_card", "cvv", "card_number"}) {
		result.ClassificationType = "Sensitive Personal Data"
		result.SubCategory = "Financial Data"
		result.DPDPACategory = "Sensitive Personal Data"
		result.ConfidenceScore = 0.95
		result.RequiresConsent = true
		result.Justification = "Pattern indicates Financial Data"
	}

	// Context boost
	if containsStrict(lowerPath, []string{"user", "users", "customer", "customers", "billing", "auth", "login", "account"}) {
		result.ConfidenceScore += 0.05
		signals["context_match"] = true
		result.Justification += " + Found in high-risk context"
	}

	// Test data penalty
	if containsStrict(lowerPath, []string{"test", "tests", "fixture", "mock", "example", "spec"}) {
		result.ConfidenceScore *= 0.9
		signals["is_test_data"] = true
		result.Justification += " (Reduced: Test Data)"
	}

	if result.ConfidenceScore > 1.0 {
		result.ConfidenceScore = 1.0
	}

	result.Signals = signals
	return result
}
