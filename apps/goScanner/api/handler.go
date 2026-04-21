package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/arc-platform/go-scanner/internal/classifier"
	"github.com/arc-platform/go-scanner/internal/orchestrator"
	"github.com/arc-platform/go-scanner/internal/presidio"
	"github.com/gin-gonic/gin"
)

// defaultScanTimeout caps the total wall-clock a single scan may run.
// Override with SCAN_TIMEOUT_SECONDS env var (integer seconds).
const defaultScanTimeout = 30 * time.Minute

func resolveScanTimeout() time.Duration {
	if v := os.Getenv("SCAN_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return defaultScanTimeout
}

// presidioClient is shared across scans and set from the Presidio base URL
// in the PRESIDIO_URL / PRESIDIO_ADDR environment variable. If neither is
// set, the client's Enabled() returns false and orchestrator skips NER.
var presidioClient = presidio.NewClient(resolvePresidioURL())

func resolvePresidioURL() string {
	if v := os.Getenv("PRESIDIO_URL"); v != "" {
		return v
	}
	return os.Getenv("PRESIDIO_ADDR")
}

// ScanRequest is the POST /scan payload the backend sends.
// It matches the shape produced by apps/backend/modules/scanning/api/scan_trigger_handler.go.
//
// Sources is polymorphic for backwards compatibility:
//   - []string of profile names (production path — resolved via ConnectionConfigs)
//   - []SourceConfig inline (test/direct path — uses Config directly)
type ScanRequest struct {
	ScanID             string                    `json:"scan_id"`
	ScanName           string                    `json:"scan_name"`
	TenantID           string                    `json:"tenant_id"`
	Sources            json.RawMessage           `json:"sources"`
	PIITypes           []string                  `json:"pii_types"`
	PIITypesPerSource  map[string][]string       `json:"pii_types_per_source"`
	ExecutionMode      string                    `json:"execution_mode"`
	ConnectionConfigs  map[string]map[string]any `json:"connection_configs"`
	CustomPatterns     []customPatternPayload    `json:"custom_patterns"`
	ClassificationMode string                    `json:"classification_mode"`
	BackendURL         string                    `json:"backend_url"`
	MaxParallel        int                       `json:"max_parallel"`
}

// SourceConfig is the legacy inline source shape, kept for tests and
// direct-API callers that still POST the source_type+config shape.
type SourceConfig struct {
	SourceType  string         `json:"source_type"`
	ProfileName string         `json:"profile_name"`
	Config      map[string]any `json:"config"`
}

type customPatternPayload struct {
	Name             string   `json:"name"`
	DisplayName      string   `json:"display_name"`
	Regex            string   `json:"regex"`
	Category         string   `json:"category"`
	ContextKeywords  []string `json:"context_keywords"`
	NegativeKeywords []string `json:"negative_keywords"`
}

// ScanHandler handles POST /scan.
func ScanHandler(c *gin.Context) {
	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ScanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scan_id is required"})
		return
	}

	backendURL := req.BackendURL
	if backendURL == "" {
		backendURL = os.Getenv("BACKEND_URL")
	}

	sourceSpecs, err := resolveSources(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customPatterns := compileCustomPatterns(req.CustomPatterns)

	classificationMode := orchestrator.ClassificationMode(req.ClassificationMode)
	if classificationMode == "" {
		classificationMode = orchestrator.ClassificationModeContextual
	}

	cfg := orchestrator.ScanConfig{
		ScanID:             req.ScanID,
		Sources:            sourceSpecs,
		CustomPatterns:     customPatterns,
		MaxConcurrency:     req.MaxParallel,
		BackendURL:         backendURL,
		ExecutionMode:      orchestrator.ExecutionMode(req.ExecutionMode),
		GlobalPIITypes:     req.PIITypes,
		PIITypesPerSource:  req.PIITypesPerSource,
		ClassificationMode: classificationMode,
		Presidio:           presidioClient,
	}

	orch := orchestrator.NewOrchestrator()

	// Run scan asynchronously; respond immediately with 202 Accepted.
	// Bound total scan wall-clock so a hung source can't leak goroutines indefinitely.
	scanTimeout := resolveScanTimeout()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
		defer cancel()
		findings, err := orch.RunScan(ctx, cfg)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log.Printf("ERR: scan %s exceeded timeout %s", req.ScanID, scanTimeout)
			} else {
				log.Printf("ERR: scan %s failed: %v", req.ScanID, err)
			}
			return
		}
		log.Printf("Scan %s complete: %d findings across %d sources", req.ScanID, len(findings), len(sourceSpecs))
		if backendURL != "" {
			if err := orchestrator.IngestFindings(req.ScanID, req.ScanName, req.TenantID, backendURL, findings); err != nil {
				log.Printf("ERR: ingest failed for scan %s: %v", req.ScanID, err)
			}
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"scan_id": req.ScanID,
		"status":  "running",
		"message": "Scan started",
	})
}

// resolveSources translates the request's polymorphic Sources field into a
// list of concrete orchestrator.SourceSpec entries.
//
// When Sources is an array of strings, each name is looked up in
// ConnectionConfigs to resolve a source_type and credential map.
// When Sources is an array of objects, each object's source_type and config
// are used directly (legacy/test path).
func resolveSources(req ScanRequest) ([]orchestrator.SourceSpec, error) {
	if len(req.Sources) == 0 {
		return nil, nil
	}

	// Try profile-name array first.
	var profileNames []string
	if err := json.Unmarshal(req.Sources, &profileNames); err == nil {
		return resolveProfileNames(profileNames, req.ConnectionConfigs), nil
	}

	// Fall back to inline source objects.
	var inline []SourceConfig
	if err := json.Unmarshal(req.Sources, &inline); err == nil {
		out := make([]orchestrator.SourceSpec, 0, len(inline))
		for _, s := range inline {
			out = append(out, orchestrator.SourceSpec{
				SourceType:  s.SourceType,
				ProfileName: s.ProfileName,
				Config:      s.Config,
			})
		}
		return out, nil
	}

	return nil, errInvalidSources
}

// errInvalidSources is returned when the sources field can't be decoded as
// either a string array (profile names) or an object array (inline configs).
var errInvalidSources = &scanRequestError{"sources must be either []string (profile names) or []{source_type,config} objects"}

type scanRequestError struct{ msg string }

func (e *scanRequestError) Error() string { return e.msg }

func resolveProfileNames(names []string, connectionConfigs map[string]map[string]any) []orchestrator.SourceSpec {
	out := make([]orchestrator.SourceSpec, 0, len(names))
	for _, name := range names {
		sourceType, cfg := lookupProfile(name, connectionConfigs)
		if sourceType == "" {
			log.Printf("WARN: profile %q not found in connection_configs; skipping", name)
			continue
		}
		out = append(out, orchestrator.SourceSpec{
			SourceType:  sourceType,
			ProfileName: name,
			Config:      cfg,
		})
	}
	return out
}

// lookupProfile finds a profile across the {source_type: {profile_name: config}} map
// and returns (sourceType, config).
func lookupProfile(name string, connectionConfigs map[string]map[string]any) (string, map[string]any) {
	for sourceType, profiles := range connectionConfigs {
		raw, ok := profiles[name]
		if !ok {
			continue
		}
		cfg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		return sourceType, cfg
	}
	return "", nil
}

func compileCustomPatterns(payloads []customPatternPayload) []classifier.CustomPattern {
	out := make([]classifier.CustomPattern, 0, len(payloads))
	for _, p := range payloads {
		r, err := regexp.Compile(p.Regex)
		if err != nil {
			log.Printf("WARN: custom pattern %q has invalid regex %q: %v", p.Name, p.Regex, err)
			continue
		}
		out = append(out, classifier.CustomPattern{
			Name:             p.Name,
			PIIType:          p.Category,
			Regex:            r,
			RawRegex:         p.Regex,
			ContextKeywords:  p.ContextKeywords,
			NegativeKeywords: p.NegativeKeywords,
		})
	}
	return out
}

// HealthHandler handles GET /health.
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": "2.0-go",
	})
}
