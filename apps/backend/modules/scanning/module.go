package scanning

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/arc-platform/backend/modules/scanning/api"
	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/audit"
	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	sharedmiddleware "github.com/arc-platform/backend/modules/shared/middleware"
	"github.com/gin-gonic/gin"
)

// ScanningModule handles scan ingestion, PII classification, and enrichment
type ScanningModule struct {
	// Services
	ingestionService             *service.IngestionService
	classificationService        *service.ClassificationService
	classificationSummaryService *service.ClassificationSummaryService
	enrichmentService            *service.EnrichmentService
	scanService                  *service.ScanService
	patternsService              *service.PatternsService

	// Handlers
	ingestionHandler      *api.IngestionHandler
	classificationHandler *api.ClassificationHandler
	sdkIngestHandler      *api.SDKIngestHandler
	scanTriggerHandler    *api.ScanTriggerHandler
	scanStatusHandler     *api.ScanStatusHandler
	dashboardHandler      *api.DashboardHandler
	patternsHandler       *api.PatternsHandler
	agentSyncHandler      *api.AgentSyncHandler
	feedbackHandler       *api.FeedbackHandler

	// Dependencies
	deps *interfaces.ModuleDependencies

	// Background jobs
	stopTimeout        chan struct{}
	triggerRateLimiter *sharedmiddleware.RateLimiter
}

// Name returns the module name
func (m *ScanningModule) Name() string {
	return "scanning"
}

// Initialize sets up the scanning module
func (m *ScanningModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps

	log.Printf("📡 Initializing Scanning & Classification Module...")

	// Create PostgreSQL repository
	repo := persistence.NewPostgresRepository(deps.DB)

	// Nil-guard LineageSync — fall back to no-op if not wired
	lineageSync := deps.LineageSync
	if lineageSync == nil {
		lineageSync = &interfaces.NoOpLineageSync{}
	}

	// Initialize services
	m.enrichmentService = service.NewEnrichmentService(repo, lineageSync)
	m.classificationService = service.NewClassificationService(repo, deps.Config)
	m.classificationSummaryService = service.NewClassificationSummaryService(repo)

	// Create scan service for scan orchestration
	m.scanService = service.NewScanService(repo)
	// Inject memory recorder (supermemory.ai) if available. Safe-nil: scans still work
	// with memory disabled; the recorder no-ops in that case.
	if deps.MemoryRecorder != nil {
		m.scanService.SetMemoryRecorder(deps.MemoryRecorder)
	}

	// Get AssetManager from dependencies (injected by main.go)
	var assetManager interfaces.AssetManager
	if deps.AssetManager != nil {
		assetManager = deps.AssetManager
	} else {
		log.Printf("⚠️  WARNING: AssetManager not available - this will cause errors")
		return fmt.Errorf("AssetManager dependency is required for Scanning Module")
	}

	// Initialize encryption service — used by both ingestion (PII sample encryption, P0-1)
	// and the scan trigger handler (runtime credential resolution).
	// A nil service is acceptable in dev when ENCRYPTION_KEY is not set.
	encryptionService, err := encryption.NewEncryptionService()
	if err != nil {
		log.Printf("WARN: Encryption service unavailable — PII samples will be stored unencrypted: %v", err)
	}

	// Ingestion service now uses AssetManager instead of creating assets directly.
	// EncryptionService is wired here to encrypt PII sample values at rest (DPDPA P0-1).
	// C4/C5: Wire Neo4j repository so ingestion writes Asset→PII_Category edges after commit.
	m.ingestionService = service.NewIngestionService(
		repo,
		m.classificationService,
		m.enrichmentService,
		assetManager,
		encryptionService,
	).WithNeo4jRepo(deps.Neo4jRepo).WithWebSocket(deps.WebSocketService)
	if deps.DB != nil {
		m.ingestionService.SetLedger(audit.NewLedgerLogger(deps.DB))
	}

	// Initialize handlers
	m.ingestionHandler = api.NewIngestionHandler(m.ingestionService)
	m.classificationHandler = api.NewClassificationHandler(
		m.classificationService,
		m.classificationSummaryService,
	)
	m.sdkIngestHandler = api.NewSDKIngestHandler(m.ingestionService)

	m.scanTriggerHandler = api.NewScanTriggerHandler(m.scanService, deps.WebSocketService, repo, encryptionService, deps.VaultClient, deps.AuditLogger)
	m.scanTriggerHandler.StartNeo4jQueueGauge(deps.DB)
	m.scanStatusHandler = api.NewScanStatusHandler(m.scanService, deps.WebSocketService, repo, deps.AuditLogger, lineageSync)
	m.dashboardHandler = api.NewDashboardHandler(repo)

	// Custom patterns
	m.patternsService = service.NewPatternsService(repo)
	m.patternsHandler = api.NewPatternsHandler(m.patternsService, deps.AuditLogger)

	// Agent sync (idempotent batch ingestion from EDR agents)
	m.agentSyncHandler = api.NewAgentSyncHandler(repo)

	// PII feedback loop — Bayesian confidence adjustment from analyst corrections
	feedbackSvc := service.NewFeedbackService(deps.DB)
	m.feedbackHandler = api.NewFeedbackHandler(feedbackSvc)

	// Start background ticker to check for stuck/timed-out scans every 5 minutes
	m.stopTimeout = make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				m.scanService.CheckAllScanTimeouts(timeoutCtx)
				cancel()
			case <-m.stopTimeout:
				return
			}
		}
	}()
	log.Printf("⏰ Scan timeout checker started (every 5 min)")

	log.Printf("✅ Scanning & Classification Module initialized")
	return nil
}

// RegisterRoutes registers the module's HTTP routes
func (m *ScanningModule) RegisterRoutes(router *gin.RouterGroup) {
	scans := router.Group("/scans")
	{
		// SDK-verified ingestion (Intelligence-at-Edge). This endpoint accepts
		// either a real user session OR a valid scanner service token. See
		// scanner_callback_auth.go for the dual-auth policy.
		scans.POST("/ingest-verified", api.ScannerCallbackAuth(), m.sdkIngestHandler.IngestVerified)

		// Scan trigger — strict rate limit (10 req/min per IP)
		m.triggerRateLimiter = sharedmiddleware.StrictRateLimiter()
		scans.POST("/trigger", m.triggerRateLimiter.Middleware(), m.scanTriggerHandler.TriggerScan)

		// Scan management — static routes MUST be registered before /:id
		// wildcards to avoid router conflicts on /clear and /latest.
		scans.GET("", m.scanStatusHandler.ListScans)
		scans.GET("/latest", m.ingestionHandler.GetLatestScan)
		scans.DELETE("/clear", m.ingestionHandler.ClearScanData)

		// Scan status and details (wildcard routes)
		scans.GET("/:id", m.scanStatusHandler.GetScan)
		scans.GET("/:id/status", m.scanStatusHandler.GetScanStatus)
		scans.POST("/:id/complete", api.ScannerCallbackAuth(), m.scanStatusHandler.CompleteScan)
		scans.POST("/:id/cancel", m.scanStatusHandler.CancelScan)
		// DELETE /:id requires admin role — no tenant_id on scan_runs yet (see RISK.md).
		// This gate limits blast radius until tenant-scoped deletion is implemented.
		scans.DELETE("/:id", func(c *gin.Context) {
			role, _ := c.Get("user_role")
			if role != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "scan deletion requires admin role"})
				c.Abort()
				return
			}
			c.Next()
		}, m.scanStatusHandler.DeleteScan)
		scans.GET("/:id/pii-summary", m.scanStatusHandler.GetScanPIISummary)
		scans.GET("/:id/delta", m.scanTriggerHandler.GetScanDelta)
		scans.POST("/:id/progress-event", api.ScannerCallbackAuth(), m.scanStatusHandler.ReceiveProgressEvent)
	}

	// Classification
	classification := router.Group("/classification")
	{
		classification.GET("/summary", m.classificationHandler.GetClassificationSummary)
	}

	// Custom PII patterns
	patterns := router.Group("/patterns")
	{
		patterns.GET("", m.patternsHandler.ListPatterns)
		patterns.POST("", m.patternsHandler.CreatePattern)
		patterns.PUT("/:id", m.patternsHandler.UpdatePattern)
		patterns.DELETE("/:id", m.patternsHandler.DeletePattern)
		// Stats, false-positive feedback, and test-suite sub-resources.
		// Static sub-paths must be registered before generic /:id to prevent router conflicts.
		patterns.GET("/precision", m.feedbackHandler.GetPatternPrecision)
		patterns.GET("/:id/stats", m.patternsHandler.GetPatternStats)
		patterns.POST("/:id/false-positive", m.patternsHandler.RecordFalsePositive)
		patterns.POST("/:id/test", m.patternsHandler.TestPattern)
	}

	// PII feedback loop
	findings := router.Group("/findings")
	{
		findings.POST("/:id/feedback", m.feedbackHandler.SubmitFeedback)
	}

	// Dashboard
	router.GET("/dashboard/metrics", m.dashboardHandler.GetDashboardMetrics)
	router.GET("/dashboard/risk-trend", m.dashboardHandler.GetRiskTrend)

	// Agent sync — idempotent batch ingestion from EDR agents
	agent := router.Group("/agent")
	{
		agent.POST("/sync", m.agentSyncHandler.Sync)
	}

	log.Printf("📡 Scanning & Classification routes registered")
}

// Shutdown performs cleanup
func (m *ScanningModule) Shutdown() error {
	log.Printf("🔌 Shutting down Scanning & Classification Module...")
	if m.stopTimeout != nil {
		close(m.stopTimeout)
	}
	if m.scanTriggerHandler != nil {
		m.scanTriggerHandler.StopGauge()
	}
	if m.triggerRateLimiter != nil {
		m.triggerRateLimiter.Stop()
	}
	return nil
}

// SetScanCompletionHook registers a hook called after each scan reaches a
// terminal success state. Used by the compliance module to write audit ledger
// entries and run obligation regression detection without a circular import.
func (m *ScanningModule) SetScanCompletionHook(fn service.ScanCompletionHook) {
	if m.scanService != nil {
		m.scanService.SetCompletionHook(fn)
	}
}

// NewScanningModule creates a new scanning module
func NewScanningModule() *ScanningModule {
	return &ScanningModule{}
}
