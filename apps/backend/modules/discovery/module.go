// Package discovery is the Data Discovery Module for ARC-HAWK-DD.
//
// It synthesizes asset, classification, lineage, and compliance state from existing
// modules into a board-readable PII risk overview. It is read-only against other
// modules: it does not write to scanning, lineage, or compliance tables.
//
// Module pattern matches scanning/compliance/lineage/connections — see
// apps/backend/modules/shared/interfaces/module.go for the contract.
package discovery

import (
	"context"
	"log"
	"time"

	"github.com/arc-platform/backend/modules/discovery/api"
	"github.com/arc-platform/backend/modules/discovery/service"
	"github.com/arc-platform/backend/modules/discovery/worker"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

// DiscoveryModule provides the Data Discovery feature: inventory, snapshots,
// risk scoring, drift detection, and board reports.
type DiscoveryModule struct {
	// Services
	repo             *service.Repo
	inventoryService *service.InventoryService
	snapshotService  *service.SnapshotService
	riskScoringSvc   *service.RiskScoringService
	driftDetector    *service.DriftDetectionService
	reportService    *service.ReportService

	// Handlers
	overviewHandler  *api.OverviewHandler
	inventoryHandler *api.InventoryHandler
	snapshotHandler  *api.SnapshotHandler
	riskHandler      *api.RiskHandler
	driftHandler     *api.DriftHandler
	reportHandler    *api.ReportHandler
	glossaryHandler  *api.GlossaryHandler

	// Background worker
	snapshotWorker *worker.SnapshotWorker
	stopWorker     chan struct{}

	deps *interfaces.ModuleDependencies
}

// NewDiscoveryModule creates a new discovery module.
func NewDiscoveryModule() *DiscoveryModule {
	return &DiscoveryModule{}
}

// Name returns the module name.
func (m *DiscoveryModule) Name() string {
	return "discovery"
}

// Initialize sets up the discovery module with its dependencies.
func (m *DiscoveryModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps
	log.Printf("🔍 Initializing Data Discovery Module...")

	// Repo
	m.repo = service.NewRepo(deps.DB)

	// Services
	// Inventory + Snapshot read from upstream via the cross-module readers wired in main.go.
	upstream := service.NewUpstreamFromDeps(deps)
	m.inventoryService = service.NewInventoryService(m.repo, upstream)
	m.snapshotService = service.NewSnapshotService(m.repo, m.inventoryService, deps.AuditLogger)
	m.riskScoringSvc = service.NewRiskScoringService(m.repo)
	m.driftDetector = service.NewDriftDetectionService(m.repo)
	m.reportService = service.NewReportService(m.repo)

	// Start the report worker pool (in-process job queue per E6).
	m.reportService.StartWorkers(2)

	// Handlers
	m.overviewHandler = api.NewOverviewHandler(m.inventoryService, m.snapshotService, m.riskScoringSvc, m.repo)
	m.inventoryHandler = api.NewInventoryHandler(m.repo)
	m.snapshotHandler = api.NewSnapshotHandler(m.snapshotService, m.repo)
	m.riskHandler = api.NewRiskHandler(m.riskScoringSvc, m.repo)
	m.driftHandler = api.NewDriftHandler(m.driftDetector, m.repo)
	m.reportHandler = api.NewReportHandler(m.reportService, m.repo)
	m.glossaryHandler = api.NewGlossaryHandler()

	// Background worker — reuses the same ticker+stop-chan pattern as scanning/module.go:97.
	interval := 24 * time.Hour
	if deps.Config != nil && deps.Config.DiscoverySnapshotInterval > 0 {
		interval = deps.Config.DiscoverySnapshotInterval
	}
	m.snapshotWorker = worker.NewSnapshotWorker(
		m.snapshotService,
		m.driftDetector,
		service.NewTenantListerFromDB(deps.DB),
		interval,
	)
	m.stopWorker = make(chan struct{})
	go m.snapshotWorker.Run(context.Background(), m.stopWorker)

	log.Printf("⏰ Discovery snapshot worker started (interval: %s)", interval)
	log.Printf("✅ Data Discovery Module initialized")
	return nil
}

// RegisterRoutes registers the discovery module's HTTP routes.
func (m *DiscoveryModule) RegisterRoutes(router *gin.RouterGroup) {
	discovery := router.Group("/discovery")
	{
		// Overview dashboard
		discovery.GET("/overview", m.overviewHandler.GetOverview)

		// Inventory
		inventory := discovery.Group("/inventory")
		{
			inventory.GET("", m.inventoryHandler.ListInventory)
			inventory.GET("/:assetId", m.inventoryHandler.GetAssetInventory)
		}

		// Snapshots
		snapshots := discovery.Group("/snapshots")
		{
			snapshots.GET("", m.snapshotHandler.ListSnapshots)
			snapshots.GET("/:id", m.snapshotHandler.GetSnapshot)
			snapshots.POST("/trigger", m.snapshotHandler.TriggerSnapshot)
		}

		// Risk
		risk := discovery.Group("/risk")
		{
			risk.GET("/overview", m.riskHandler.GetRiskOverview)
			risk.GET("/hotspots", m.riskHandler.GetRiskHotspots)
			risk.GET("/scores/:assetId", m.riskHandler.GetAssetRiskHistory)
		}

		// Drift
		drift := discovery.Group("/drift")
		{
			drift.GET("/since/:snapshotId", m.driftHandler.GetDriftSince)
			drift.GET("/timeline", m.driftHandler.GetDriftTimeline)
		}

		// Reports
		reports := discovery.Group("/reports")
		{
			reports.POST("/generate", m.reportHandler.GenerateReport)
			reports.GET("", m.reportHandler.ListReports)
			reports.GET("/:id", m.reportHandler.GetReport)
			reports.GET("/:id/download", m.reportHandler.DownloadReport)
		}

		// Glossary stub (v1.5)
		discovery.GET("/glossary", m.glossaryHandler.GetGlossary)
	}

	log.Printf("🔍 Discovery routes registered (15 endpoints)")
}

// Shutdown stops the background worker and releases resources.
func (m *DiscoveryModule) Shutdown() error {
	log.Printf("🔌 Shutting down Data Discovery Module...")
	if m.stopWorker != nil {
		close(m.stopWorker)
	}
	if m.reportService != nil {
		m.reportService.StopWorkers()
	}
	return nil
}
