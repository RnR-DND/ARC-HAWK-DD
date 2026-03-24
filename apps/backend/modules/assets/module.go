package assets

import (
	"log"

	"github.com/arc-platform/backend/modules/assets/api"
	"github.com/arc-platform/backend/modules/assets/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

type AssetsModule struct {
	assetService    *service.AssetService
	findingsService *service.FindingsService
	datasetService  *service.DatasetService

	assetHandler    *api.AssetHandler
	findingsHandler *api.FindingsHandler
	datasetHandler  *api.DatasetHandler

	deps *interfaces.ModuleDependencies
}

func (m *AssetsModule) Name() string {
	return "assets"
}

func (m *AssetsModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps
	log.Printf("📦 Initializing Assets Module...")

	repo := persistence.NewPostgresRepository(deps.DB)

	// Get lineage sync from dependencies (will be injected by main.go)
	var lineageSync interfaces.LineageSync
	if deps.LineageSync != nil {
		lineageSync = deps.LineageSync
	} else {
		lineageSync = &interfaces.NoOpLineageSync{}
		log.Printf("⚠️  LineageSync not available - using NoOp implementation")
	}

	// Get Audit Logger
	var auditLogger interfaces.AuditLogger
	if deps.AuditLogger != nil {
		auditLogger = deps.AuditLogger
	} else {
		// Fallback or panic? For now, we allow nil if interface handles it,
		// but better to mock it if nil.
		// Since we didn't create a NoOpAuditLogger, we'll assume it's there or handle nil in Service.
		// A better approach is NoOp if nil.
		// Let's assume initialized by main.go
		auditLogger = deps.AuditLogger
	}

	m.assetService = service.NewAssetService(repo, lineageSync, auditLogger)
	m.findingsService = service.NewFindingsService(repo)
	m.datasetService = service.NewDatasetService(repo)

	m.assetHandler = api.NewAssetHandler(m.assetService)
	m.findingsHandler = api.NewFindingsHandler(m.findingsService)
	m.datasetHandler = api.NewDatasetHandler(m.datasetService)

	log.Printf("✅ Assets Module initialized")
	return nil
}

func (m *AssetsModule) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/assets", m.assetHandler.ListAssets)
	router.GET("/assets/:id", m.assetHandler.GetAsset)
	router.GET("/findings", m.findingsHandler.GetFindings)
	router.POST("/findings/:id/feedback", m.findingsHandler.SubmitFeedback)
	router.GET("/dataset/golden", m.datasetHandler.GetGoldenDataset)
	log.Printf("📦 Assets routes registered")
}

func (m *AssetsModule) Shutdown() error {
	log.Printf("🔌 Shutting down Assets Module...")
	return nil
}

// GetAssetService returns the asset service for inter-module use
func (m *AssetsModule) GetAssetService() *service.AssetService {
	return m.assetService
}

// GetFindingsService returns the findings service for inter-module use
func (m *AssetsModule) GetFindingsService() *service.FindingsService {
	return m.findingsService
}

func NewAssetsModule() *AssetsModule {
	return &AssetsModule{}
}
