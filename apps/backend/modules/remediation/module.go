// Package remediation orchestrates deletion and masking actions against data sources and records every action in the audit log.
package remediation

import (
	"database/sql"
	"log"

	"github.com/arc-platform/backend/modules/auth/middleware"
	"github.com/arc-platform/backend/modules/remediation/api"
	"github.com/arc-platform/backend/modules/remediation/service"
	remservice "github.com/arc-platform/backend/modules/remediation/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/audit"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

// RemediationModule implements the Module interface
type RemediationModule struct {
	db                *sql.DB
	lineageSync       interfaces.LineageSync
	service           *service.RemediationService
	escalationService *service.EscalationService
	authMiddleware    *middleware.AuthMiddleware
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

	// Initialize service with LineageSync + AuditLogger (shared hash-chained interface)
	m.service = service.NewRemediationService(m.db, m.lineageSync, deps.AuditLogger)
	m.service.SetLedger(audit.NewLedgerLogger(m.db))
	m.escalationService = service.NewEscalationService(m.db)

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
	exportHandler := api.NewRemediationExportHandler(m.service)
	escalationHandler := api.NewEscalationHandler(m.escalationService)

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

		// Export: GET /remediation/export?format=pdf|xlsx
		g.GET("/export", exportHandler.ExportReport)

		// Escalation scheduler endpoints
		escalation := g.Group("/escalation")
		{
			escalation.GET("/preview", escalationHandler.Preview)
			escalation.POST("/run", escalationHandler.Run)
		}

		// SOP registry — machine-readable runbooks for all known issue types
		g.GET("/sops", func(c *gin.Context) {
			c.JSON(200, gin.H{"sops": remservice.ListSOPs()})
		})
		g.GET("/sops/:issue_type", func(c *gin.Context) {
			sop := remservice.LookupSOP(remservice.IssueType(c.Param("issue_type")))
			if sop == nil {
				c.JSON(404, gin.H{"error": "SOP not found"})
				return
			}
			c.JSON(200, sop)
		})

		// Dynamic route last
		g.GET("/:id", handler.GetRemediationAction)
	}
}

// Shutdown cleans up resources
func (m *RemediationModule) Shutdown() error {
	return nil
}
