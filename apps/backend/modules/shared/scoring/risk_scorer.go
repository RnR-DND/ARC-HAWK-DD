// Package scoring provides the canonical risk scoring formula for ARC-Hawk.
//
// Problem: Two incompatible formulas existed in the codebase:
//
//  1. apps/backend/modules/scanning/service/ingestion_service.go
//     calculateComprehensiveRiskScore():
//     score = classificationWeight*0.6 + confidence*0.2 + context*0.2
//
//  2. hawk/backend/internal/risk/handler.go RecalculateRiskScores():
//     score = piiDensity*0.35 + sensitivityWeight*0.30 + accessExposure*0.20 + retentionViolation*0.15
//
// This package provides ComputeRiskScore() — a unified formula that preserves
// the strengths of both approaches — so the same asset always receives the same
// risk score regardless of which code path calculated it.
package scoring

import "math"

// RiskScoreParams holds the normalised inputs to the canonical risk formula.
// All float64 fields are expected in the [0, 1] range; the scorer clamps them.
type RiskScoreParams struct {
	// PIIType drives the classification sensitivity tier.
	// Accepted values: "Sensitive Personal Data", "Secrets", "Personal Data",
	// any other string → lowest tier (10 points).
	PIIType string

	// Confidence is the multi-signal classifier confidence score [0, 1].
	// Maps from the ingestion pipeline's CONFIRMED / HIGH_CONFIDENCE labels
	// (1.0 / 0.75 / 0.5 / 0.3) or a raw float from hawk/backend.
	Confidence float64

	// PIIDensity is the fraction of PII fields in the asset [0, 1].
	// Derived from piiFieldCount / totalFieldCount (hawk formula term).
	PIIDensity float64

	// AccessExposure reflects who can reach the asset.
	// Suggested mapping: external/public=1.0, internal=0.5, local/encrypted=0.2
	AccessExposure float64
}

// ComputeRiskScore returns a canonical 0-100 risk score for a PII finding or asset.
//
// Unified formula (weights sum to 1.0):
//
//	score = classificationSensitivity*0.40 + confidence*0.20 + piiDensity*0.20 + accessExposure*0.20
//
// The classification tier dominates (40 %), matching the ingestion service intent,
// while density and access exposure carry equal weight, preserving the hawk formula's
// operational context. Confidence adjusts for signal quality.
//
// Result is rounded to the nearest integer and clamped to [0, 100].
func ComputeRiskScore(params RiskScoreParams) float64 {
	classification := classificationSensitivity(params.PIIType)   // [0, 1]
	confidence := clamp(params.Confidence, 0, 1)
	density := clamp(params.PIIDensity, 0, 1)
	access := clamp(params.AccessExposure, 0, 1)

	raw := classification*0.40 + confidence*0.20 + density*0.20 + access*0.20
	score := math.Round(raw * 100)

	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

// CalculateTier maps a numeric risk score to a named tier string.
// Thresholds are consistent with hawk/backend and discovery/service/risk_engine.go.
func CalculateTier(score float64) string {
	switch {
	case score >= 80:
		return "Critical"
	case score >= 60:
		return "High"
	case score >= 40:
		return "Medium"
	default:
		return "Low"
	}
}

// classificationSensitivity returns a [0, 1] weight for each PII classification tier.
func classificationSensitivity(piiType string) float64 {
	switch piiType {
	case "Sensitive Personal Data":
		return 1.0 // Aadhaar, PAN, SSN, Credit Card, Health, Biometric
	case "Secrets":
		return 0.9 // API keys, credentials
	case "Personal Data":
		return 0.5 // Email, phone, name
	default:
		return 0.1 // Non-PII or unrecognised
	}
}

// clamp restricts v to the [lo, hi] interval.
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
