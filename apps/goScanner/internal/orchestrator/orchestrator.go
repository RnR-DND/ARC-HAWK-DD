package orchestrator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/arc-platform/go-scanner/internal/classifier"
	"github.com/arc-platform/go-scanner/internal/connectors"
	"github.com/arc-platform/go-scanner/internal/presidio"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// ExecutionMode selects serial or parallel scanning across sources.
type ExecutionMode string

const (
	ExecutionModeSequential ExecutionMode = "sequential"
	ExecutionModeParallel   ExecutionMode = "parallel"
)

// ClassificationMode selects which detection engines run on each field.
//
//   - regex:      regex + validators only; Presidio is skipped
//   - ner:        regex + Presidio (spaCy NER) with no context boosting
//   - contextual: regex + Presidio with field/table name passed as context words
//     so the LemmaContextAwareEnhancer can boost confidence
type ClassificationMode string

const (
	ClassificationModeRegex      ClassificationMode = "regex"
	ClassificationModeNER        ClassificationMode = "ner"
	ClassificationModeContextual ClassificationMode = "contextual"
)

// ScanConfig is the input to RunScan.
type ScanConfig struct {
	ScanID             string
	Sources            []SourceSpec
	CustomPatterns     []classifier.CustomPattern
	MaxConcurrency     int
	BackendURL         string
	ExecutionMode      ExecutionMode
	GlobalPIITypes     []string            // frontend PII names applied to every source without an entry in PIITypesPerSource
	PIITypesPerSource  map[string][]string // profile_name → frontend PII names
	ClassificationMode ClassificationMode
	Presidio           *presidio.Client
}

// SourceSpec describes one data source to scan.
type SourceSpec struct {
	ProfileName string
	SourceType  string
	Config      map[string]any
}

// Orchestrator coordinates scanning across multiple sources.
type Orchestrator struct {
	engine *classifier.Engine
}

// NewOrchestrator creates a new Orchestrator.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{engine: classifier.NewEngine()}
}

// presidioEnabled is true when the caller wants Presidio to run
// alongside regex (ner or contextual modes). Regex-only mode skips it.
func presidioEnabled(mode ClassificationMode) bool {
	switch mode {
	case ClassificationModeNER, ClassificationModeContextual:
		return true
	default:
		return false
	}
}

// RunScan scans all sources (parallel or sequential based on cfg.ExecutionMode)
// and returns aggregated findings.
func (o *Orchestrator) RunScan(ctx context.Context, cfg ScanConfig) ([]classifier.ClassifiedFinding, error) {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 4 // matches Python ThreadPoolExecutor default
	}
	if cfg.ExecutionMode == "" {
		cfg.ExecutionMode = ExecutionModeParallel
	}

	if strings.EqualFold(string(cfg.ExecutionMode), string(ExecutionModeSequential)) {
		return o.runSequential(ctx, cfg)
	}
	return o.runParallel(ctx, cfg)
}

func (o *Orchestrator) runSequential(ctx context.Context, cfg ScanConfig) ([]classifier.ClassifiedFinding, error) {
	var all []classifier.ClassifiedFinding
	for _, src := range cfg.Sources {
		if ctx.Err() != nil {
			return all, ctx.Err()
		}
		allowed := o.allowedForSource(src.ProfileName, cfg)
		presidioEntities := o.presidioEntitiesForSource(src.ProfileName, cfg)
		findings, err := o.scanSource(ctx, src, cfg, allowed, presidioEntities)
		if err != nil {
			log.Printf("WARN: sequential source %s/%s failed: %v", src.SourceType, src.ProfileName, err)
			continue
		}
		all = append(all, findings...)
	}
	return all, nil
}

func (o *Orchestrator) runParallel(ctx context.Context, cfg ScanConfig) ([]classifier.ClassifiedFinding, error) {
	g, gctx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(int64(cfg.MaxConcurrency))

	var mu sync.Mutex
	var all []classifier.ClassifiedFinding

	for _, src := range cfg.Sources {
		src := src
		g.Go(func() error {
			if err := sem.Acquire(gctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			allowed := o.allowedForSource(src.ProfileName, cfg)
			presidioEntities := o.presidioEntitiesForSource(src.ProfileName, cfg)
			findings, err := o.scanSource(gctx, src, cfg, allowed, presidioEntities)
			if err != nil {
				log.Printf("WARN: parallel source %s/%s failed: %v", src.SourceType, src.ProfileName, err)
				return nil // non-fatal: continue other sources
			}
			if len(findings) > 0 {
				mu.Lock()
				all = append(all, findings...)
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return all, err
	}
	return all, nil
}

// allowedForSource resolves the pattern allowlist for a single source:
// per-source PII types win; otherwise fall back to global.
func (o *Orchestrator) allowedForSource(profileName string, cfg ScanConfig) map[string]struct{} {
	if cfg.PIITypesPerSource != nil {
		if list, ok := cfg.PIITypesPerSource[profileName]; ok {
			return classifier.AllowedPatternSet(list)
		}
	}
	return classifier.AllowedPatternSet(cfg.GlobalPIITypes)
}

// presidioEntitiesForSource resolves which Presidio entity names should be
// requested for this source. Returns nil when Presidio must not run for this
// source — either because mode disables it, no PII types were selected, or
// the selected types don't map to any Presidio entity.
//
// This enforces the user's selection strictly: if you asked for IN_PAN and
// IN_AADHAAR only (neither has a Presidio mapping), Presidio is skipped
// rather than falling back to "detect everything".
func (o *Orchestrator) presidioEntitiesForSource(profileName string, cfg ScanConfig) []string {
	if !presidioEnabled(cfg.ClassificationMode) {
		return nil
	}
	piiTypes := cfg.GlobalPIITypes
	if cfg.PIITypesPerSource != nil {
		if list, ok := cfg.PIITypesPerSource[profileName]; ok {
			piiTypes = list
		}
	}
	if len(piiTypes) == 0 {
		return nil
	}
	entities := classifier.PresidioEntitiesFor(piiTypes)
	if len(entities) == 0 {
		return nil
	}
	return entities
}

func (o *Orchestrator) scanSource(ctx context.Context, src SourceSpec, cfg ScanConfig, allowed map[string]struct{}, presidioEntities []string) ([]classifier.ClassifiedFinding, error) {
	conn, err := connectors.Dispatch(src.SourceType)
	if err != nil {
		return nil, err
	}
	if err := conn.Connect(ctx, src.Config); err != nil {
		return nil, err
	}
	defer conn.Close()

	host := stringFromConfig(src.Config, "host")
	if host == "" {
		host = stringFromConfig(src.Config, "bucket")
	}
	if host == "" {
		host = stringFromConfig(src.Config, "path")
	}

	usePresidio := presidioEntities != nil && cfg.Presidio.Enabled()

	fieldsCh, errCh := conn.StreamFields(ctx)

	var findings []classifier.ClassifiedFinding
	for {
		select {
		case rec, ok := <-fieldsCh:
			if !ok {
				return findings, nil
			}
			batch := o.engine.Classify(rec, cfg.CustomPatterns, allowed)
			for i := range batch {
				batch[i].DataSource = src.SourceType
				batch[i].Host = host
				batch[i].Table = tableFromSourcePath(batch[i].SourcePath)
			}
			findings = append(findings, batch...)

			if usePresidio {
				presidioFindings := o.runPresidio(ctx, cfg.Presidio, rec, src, cfg, host, presidioEntities)
				findings = append(findings, presidioFindings...)
			}
		case err, ok := <-errCh:
			if ok && err != nil {
				return findings, err
			}
		case <-ctx.Done():
			return findings, ctx.Err()
		}
	}
}

// runPresidio sends the record's value to Presidio and converts the returned
// entities into ClassifiedFindings.
//
// Three things are shipped to Presidio per call:
//  1. Built-in Presidio recognizers (always).
//  2. Indian PII recognizers (IN_PAN, IN_AADHAAR, ...) as ad-hoc — Presidio's
//     default model is US-centric and does not know these entities.
//  3. The user's custom regex patterns as ad-hoc — Presidio runs them with
//     context-aware scoring; matches come back under the user's category
//     (e.g. USR_EMPLOYEE_ID) so they pass IsLockedPIIType in the backend.
//
// The `entities` allowlist constrains the results to what the user selected
// plus any custom-pattern categories that apply.
func (o *Orchestrator) runPresidio(ctx context.Context, client *presidio.Client, rec connectors.FieldRecord, src SourceSpec, cfg ScanConfig, host string, entities []string) []classifier.ClassifiedFinding {
	contextual := cfg.ClassificationMode == ClassificationModeContextual

	var contextWords []string
	if contextual {
		if rec.FieldName != "" {
			contextWords = append(contextWords, rec.FieldName)
		}
		if src.ProfileName != "" {
			contextWords = append(contextWords, src.ProfileName)
		}
		if src.SourceType != "" {
			contextWords = append(contextWords, src.SourceType)
		}
	}

	adHoc := buildAdHocRecognizers(entities, cfg.CustomPatterns)
	allowedEntities := extendEntitiesForCustomPatterns(entities, cfg.CustomPatterns)

	opts := presidio.AnalyzeOptions{
		Entities:         allowedEntities,
		ContextWords:     contextWords,
		AdHocRecognizers: adHoc,
	}
	results := client.Analyze(ctx, rec.Value, opts)
	if len(results) == 0 {
		return nil
	}

	out := make([]classifier.ClassifiedFinding, 0, len(results))
	for _, e := range results {
		if e.Start < 0 || e.End > len(rec.Value) || e.Start >= e.End {
			continue
		}
		matched := rec.Value[e.Start:e.End]
		out = append(out, classifier.ClassifiedFinding{
			PIIType:        e.Type,
			ValueHash:      hashValue(matched),
			Score:          int(e.Score * 100),
			DetectorType:   "presidio",
			SourcePath:     rec.SourcePath,
			FieldName:      rec.FieldName,
			ContextExcerpt: excerptAround(rec.Value, e.Start, e.End),
			PatternName:    "presidio:" + e.Type,
			DataSource:     src.SourceType,
			Host:           host,
			Table:          tableFromSourcePath(rec.SourcePath),
		})
	}
	return out
}

// buildAdHocRecognizers returns the list of Presidio ad-hoc recognizers to
// register on each /analyze call:
//
//   - Indian recognizers whose supported_entity overlaps with `entities`
//     (the user's selection). Sending the full Indian pack unconditionally
//     would cause noise when the user only asked for, say, PAN and Email.
//   - Every user-defined custom regex pattern — Presidio applies
//     context-aware scoring and returns matches with the user's category.
func buildAdHocRecognizers(entities []string, customPatterns []classifier.CustomPattern) []presidio.AdHocRecognizer {
	entitySet := make(map[string]struct{}, len(entities))
	for _, e := range entities {
		entitySet[e] = struct{}{}
	}

	var out []presidio.AdHocRecognizer
	for _, r := range presidio.IndianRecognizers() {
		if _, ok := entitySet[r.SupportedEntity]; ok {
			out = append(out, r)
		}
	}
	for _, cp := range customPatterns {
		if cp.PIIType == "" || cp.RawRegex == "" {
			continue
		}
		out = append(out, presidio.AdHocRecognizer{
			Name:              "custom_" + cp.Name,
			SupportedLanguage: "en",
			SupportedEntity:   cp.PIIType,
			Context:           cp.ContextKeywords,
			Patterns: []presidio.PatternSpec{
				{Name: cp.Name, Regex: cp.RawRegex, Score: 0.85},
			},
		})
	}
	return out
}

// extendEntitiesForCustomPatterns appends user-defined pattern categories to
// the Presidio entities allowlist. Without this, Presidio would drop matches
// from the custom ad-hoc recognizers because their supported_entity (e.g.
// USR_EMPLOYEE_ID) is not in the caller-provided filter.
func extendEntitiesForCustomPatterns(entities []string, customPatterns []classifier.CustomPattern) []string {
	if len(customPatterns) == 0 {
		return entities
	}
	seen := make(map[string]struct{}, len(entities)+len(customPatterns))
	out := make([]string, 0, len(entities)+len(customPatterns))
	for _, e := range entities {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	for _, cp := range customPatterns {
		if cp.PIIType == "" {
			continue
		}
		if _, ok := seen[cp.PIIType]; ok {
			continue
		}
		seen[cp.PIIType] = struct{}{}
		out = append(out, cp.PIIType)
	}
	return out
}

// hashValue duplicates classifier.hashValue to avoid exposing it.
func hashValue(v string) string {
	h := sha256.Sum256([]byte(v))
	return fmt.Sprintf("%x", h)
}

func excerptAround(text string, start, end int) string {
	left := start - 50
	if left < 0 {
		left = 0
	}
	right := end + 50
	if right > len(text) {
		right = len(text)
	}
	return text[left:right]
}

func stringFromConfig(cfg map[string]any, key string) string {
	if v, ok := cfg[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// tableFromSourcePath pulls the table/collection segment out of a path like
// "schema.table.column" (databases) or "bucket/key" (object storage). For paths
// without a clear table segment, it falls back to the full SourcePath.
func tableFromSourcePath(path string) string {
	if path == "" {
		return ""
	}
	// Database convention: schema.table.column — return "table".
	if parts := splitOn(path, '.'); len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return path
}

func splitOn(s string, sep byte) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
