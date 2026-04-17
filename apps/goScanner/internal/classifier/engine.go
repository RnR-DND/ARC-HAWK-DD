package classifier

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// ClassifiedFinding is the output of the classification engine. Fields are
// named to line up with the backend's VerifiedFinding schema so the ingest
// layer is a straight serialization pass.
type ClassifiedFinding struct {
	PIIType        string
	ValueHash      string
	Score          int
	DetectorType   string
	SourcePath     string
	FieldName      string
	ContextExcerpt string
	PatternName    string

	// Populated by the orchestrator after Classify — Engine itself only
	// knows about the record, not the source/host.
	DataSource string
	Host       string
	Table      string
}

// Engine classifies FieldRecords against PII patterns.
type Engine struct {
	patterns []Pattern
}

// NewEngine creates a new Engine with all built-in patterns.
func NewEngine() *Engine {
	return &Engine{patterns: AllPatterns()}
}

// Classify runs patterns against a FieldRecord and returns deduplicated findings.
//
// Filtering semantics:
//   - `allowedPatterns == nil` — run every built-in pattern (the caller didn't
//     pass a pii_types filter at all).
//   - `allowedPatterns` is a non-nil but empty set — run NO built-in patterns
//     (the caller selected pii_types but none of them map to a built-in
//     pattern; this strictly honors the user's selection).
//   - `allowedPatterns` is non-empty — run only patterns whose Pattern.Name is
//     in the set.
//
// Custom patterns always run regardless of the allowlist — they are
// user-defined and explicit.
func (e *Engine) Classify(record connectors.FieldRecord, custom []CustomPattern, allowedPatterns map[string]struct{}) []ClassifiedFinding {
	var findings []ClassifiedFinding

	runBuiltins := allowedPatterns == nil || len(allowedPatterns) > 0
	for _, pat := range e.patterns {
		if !runBuiltins {
			break
		}
		if len(allowedPatterns) > 0 {
			if _, ok := allowedPatterns[pat.Name]; !ok {
				continue
			}
		}
		matches := pat.Regex.FindAllString(record.Value, -1)
		for _, m := range matches {
			score, detType := Score(m, pat.PIIType, record, nil, nil)
			if score >= 50 {
				findings = append(findings, ClassifiedFinding{
					PIIType:        pat.PIIType,
					ValueHash:      hashValue(m),
					Score:          score,
					DetectorType:   detType,
					SourcePath:     record.SourcePath,
					FieldName:      record.FieldName,
					ContextExcerpt: excerpt(record.Value, m),
					PatternName:    pat.Name,
				})
			}
		}
	}

	for _, cp := range custom {
		if cp.Regex == nil {
			continue
		}
		matches := cp.Regex.FindAllString(record.Value, -1)
		for _, m := range matches {
			score, detType := Score(m, cp.PIIType, record, cp.ContextKeywords, cp.NegativeKeywords)
			if score >= 50 {
				findings = append(findings, ClassifiedFinding{
					PIIType:        cp.PIIType,
					ValueHash:      hashValue(m),
					Score:          score,
					DetectorType:   detType,
					SourcePath:     record.SourcePath,
					FieldName:      record.FieldName,
					ContextExcerpt: excerpt(record.Value, m),
					PatternName:    cp.Name,
				})
			}
		}
	}

	return Dedup(findings)
}

func hashValue(v string) string {
	h := sha256.Sum256([]byte(v))
	return fmt.Sprintf("%x", h)
}

func excerpt(text, match string) string {
	idx := strings.Index(text, match)
	if idx < 0 {
		if len(text) > 200 {
			return text[:200]
		}
		return text
	}
	start := max(idx-50, 0)
	end := min(idx+len(match)+50, len(text))
	return text[start:end]
}
