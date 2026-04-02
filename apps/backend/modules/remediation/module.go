package remediation

import (
	"database/sql"
	"log"

	"github.com/arc-platform/backend/modules/auth/middleware"
	"github.com/arc-platform/backend/modules/remediation/api"
	"github.com/arc-platform/backend/modules/remediation/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

// RemediationModule implements the Module interface
type RemediationModule struct {
	db             *sql.DB
	lineageSync    interfaces.LineageSync
	service        *service.RemediationService
	authMiddleware *middleware.AuthMiddleware
}

// NewRemediationModule creates a new remediation module
func NewRemediationModule() *RemediationModule {
	return &RemediationModule{}
}

// Name returns the module name
func (m *RemediationModule) Name() string {
	return "Remediation"
}

// Initialize sets up the module
func (m *RemediationModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.db = deps.DB

	// Get LineageSync from dependencies
	if deps.LineageSync != nil {
		m.lineageSync = deps.LineageSync
	} else {
		m.lineageSync = &interfaces.NoOpLineageSync{}
		log.Printf("⚠️  LineageSync not available - using NoOp implementation")
	}

	// Initialize service with LineageSync instead of Neo4j driver
	m.service = service.NewRemediationService(m.db, m.lineageSync)

	// Initialize Auth Middleware for permission checks
	repo := persistence.NewPostgresRepository(m.db)
	m.authMiddleware = middleware.NewAuthMiddleware(repo, m.db)

	log.Println("✅ Remediation module initialized")
	return nil
}

// RegisterRoutes registers the module's routes
func (m *RemediationModule) RegisterRoutes(router *gin.RouterGroup) {
	handler := api.NewRemediationHandler(m.service)
	historyHandler := api.NewRemediationHistoryHandler(m.service)

	// Create remediation group
	g := router.Group("/remediation")
	{
		g.POST("/preview", handler.GeneratePreview)
		// Enforce "remediation:execute" permission for execution
		g.POST("/execute", m.authMiddleware.RequirePermission("remediation:execute"), handler.ExecuteRemediation)

		// Specific routes MUST come before dynamic /:id route
		g.GET("/history", historyHandler.GetHistory)
		g.GET("/history/:assetId", handler.GetRemediationHistory)
		g.GET("/actions/:findingId", handler.GetRemediationActions)
		g.POST("/rollback/:id", handler.RollbackRemediation)

		// Dynamic route last
		g.GET("/:id", handler.GetRemediationAction)
	}
}

// Shutdown cleans up resources
func (m *RemediationModule) Shutdown() error {
	return nil
}
