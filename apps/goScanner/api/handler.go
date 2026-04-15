package api

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/arc-platform/go-scanner/internal/classifier"
	"github.com/arc-platform/go-scanner/internal/orchestrator"
	"github.com/gin-gonic/gin"
)

// ScanRequest is the POST /scan payload.
type ScanRequest struct {
	ScanID      string         `json:"scan_id"`
	Sources     []SourceConfig `json:"sources"`
	BackendURL  string         `json:"backend_url"`
	MaxParallel int            `json:"max_parallel"`
}

// SourceConfig describes one data source in a scan request.
type SourceConfig struct {
	SourceType string         `json:"source_type"`
	Config     map[string]any `json:"config"`
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

	sources := make([]orchestrator.SourceSpec, 0, len(req.Sources))
	for _, s := range req.Sources {
		sources = append(sources, orchestrator.SourceSpec{
			SourceType: s.SourceType,
			Config:     s.Config,
		})
	}

	cfg := orchestrator.ScanConfig{
		ScanID:         req.ScanID,
		Sources:        sources,
		CustomPatterns: []classifier.CustomPattern{},
		MaxConcurrency: req.MaxParallel,
		BackendURL:     backendURL,
	}

	orch := orchestrator.NewOrchestrator()

	// Run scan asynchronously; respond immediately with 202 Accepted
	go func() {
		ctx := context.Background()
		findings, err := orch.RunScan(ctx, cfg)
		if err != nil {
			log.Printf("ERR: scan %s failed: %v", req.ScanID, err)
			return
		}
		log.Printf("Scan %s complete: %d findings", req.ScanID, len(findings))
		if backendURL != "" {
			if err := orchestrator.IngestFindings(req.ScanID, backendURL, findings); err != nil {
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

// HealthHandler handles GET /health.
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": "2.0-go",
	})
}
