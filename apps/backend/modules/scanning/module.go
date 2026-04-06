package scanning

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arc-platform/backend/modules/scanning/api"
	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
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

	// Handlers
	ingestionHandler      *api.IngestionHandler
	classificationHandler *api.ClassificationHandler
	sdkIngestHandler      *api.SDKIngestHandler
	scanTriggerHandler    *api.ScanTriggerHandler
	scanStatusHandler     *api.ScanStatusHandler
	dashboardHandler      *api.DashboardHandler

	// Dependencies
	deps *interfaces.ModuleDependencies

	// Background jobs
	stopTimeout chan struct{}
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

	// Initialize services
	m.enrichmentService = service.NewEnrichmentService(repo, deps.LineageSync)
	m.classificationService = service.NewClassificationService(repo, deps.Config)
	m.classificationSummaryService = service.NewClassificationSummaryService(repo)

	// Create scan service for scan orchestration
	m.scanService = service.NewScanService(repo)

	// Get AssetManager from dependencies (injected by main.go)
	var assetManager interfaces.AssetManager
	if deps.AssetManager != nil {
		assetManager = deps.AssetManager
	} else {
		log.Printf("⚠️  WARNING: AssetManager not available - this will cause errors")
		return fmt.Errorf("AssetManager dependency is required for Scanning Module")
	}

	// Ingestion service now uses AssetManager instead of creating assets directly
	m.ingestionService = service.NewIngestionService(
		repo,
		m.classificationService,
		m.enrichmentService,
		assetManager,
	)

	// Initialize handlers
	m.ingestionHandler = api.NewIngestionHandler(m.ingestionService)
	m.classificationHandler = api.NewClassificationHandler(
		m.classificationService,
		m.classificationSummaryService,
	)
	m.sdkIngestHandler = api.NewSDKIngestHandler(m.ingestionService)

	// Initialize encryption for runtime credential resolution
	encryptionService, err := encryption.NewEncryptionService()
	if err != nil {
		log.Printf("WARN: Encryption service unavailable, scanner will use connection.yml fallback: %v", err)
	}
	m.scanTriggerHandler = api.NewScanTriggerHandler(m.scanService, deps.WebSocketService, repo, encryptionService)
	m.scanStatusHandler = api.NewScanStatusHandler(m.scanService, deps.WebSocketService, repo)
	m.dashboardHandler = api.NewDashboardHandler(repo)

	// Start background ticker to check for stuck/timed-out scans every 5 minutes
	m.stopTimeout = make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.scanService.CheckAllScanTimeouts(context.Background())
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
		// SDK-verified ingestion (Intelligence-at-Edge)
		scans.POST("/ingest-verified", m.sdkIngestHandler.IngestVerified)

		// Scan trigger
		scans.POST("/trigger", m.scanTriggerHandler.TriggerScan)

		// Scan status and details
		scans.GET("/:id", m.scanStatusHandler.GetScan)
		scans.GET("/:id/status", m.scanStatusHandler.GetScanStatus)
		scans.POST("/:id/complete", m.scanStatusHandler.CompleteScan)
		scans.POST("/:id/cancel", m.scanStatusHandler.CancelScan)
		scans.DELETE("/:id", m.scanStatusHandler.DeleteScan)
		scans.GET("/:id/pii-summary", m.scanStatusHandler.GetScanPIISummary)

		// Scan management
		scans.GET("", m.scanStatusHandler.ListScans)
		scans.GET("/latest", m.ingestionHandler.GetLatestScan)
		scans.DELETE("/clear", m.ingestionHandler.ClearScanData)
	}

	// Classification
	classification := router.Group("/classification")
	{
		classification.GET("/summary", m.classificationHandler.GetClassificationSummary)
	}

	// Dashboard
	router.GET("/dashboard/metrics", m.dashboardHandler.GetDashboardMetrics)

	log.Printf("📡 Scanning & Classification routes registered")
}

// Shutdown performs cleanup
func (m *ScanningModule) Shutdown() error {
	log.Printf("🔌 Shutting down Scanning & Classification Module...")
	if m.stopTimeout != nil {
		close(m.stopTimeout)
	}
	return nil
}

// NewScanningModule creates a new scanning module
func NewScanningModule() *ScanningModule {
	return &ScanningModule{}
}
