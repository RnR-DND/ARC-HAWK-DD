package classifier

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/arc-platform/go-scanner/internal/connectors"
)

// ClassifiedFinding is the output of the classification engine.
type ClassifiedFinding struct {
	PIIType        string
	ValueHash      string
	Score          int
	DetectorType   string
	SourcePath     string
	ContextExcerpt string
	PatternName    string
}

// Engine classifies FieldRecords against PII patterns.
type Engine struct {
	patterns []Pattern
}

// NewEngine creates a new Engine with all built-in patterns.
func NewEngine() *Engine {
	return &Engine{patterns: AllPatterns()}
}

// Classify runs all patterns against a FieldRecord and returns deduplicated findings.
func (e *Engine) Classify(record connectors.FieldRecord, custom []CustomPattern) []ClassifiedFinding {
	var findings []ClassifiedFinding

	// Built-in patterns
	for _, pat := range e.patterns {
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
					ContextExcerpt: excerpt(record.Value, m),
					PatternName:    pat.Name,
				})
			}
		}
	}

	// Custom patterns
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
	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := idx + len(match) + 50
	if end > len(text) {
		end = len(text)
	}
	return text[start:end]
}
