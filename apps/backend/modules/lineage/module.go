// Package lineage tracks how PII flows between systems, assets, and fields using a Neo4j-backed semantic graph.
package lineage

import (
	"fmt"
	"log"

	"github.com/arc-platform/backend/modules/lineage/api"
	"github.com/arc-platform/backend/modules/lineage/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

type LineageModule struct {
	semanticLineageService *service.SemanticLineageService

	graphHandler   *api.GraphHandler
	lineageHandler *api.LineageHandler

	deps *interfaces.ModuleDependencies
}

func (m *LineageModule) Name() string {
	return "lineage"
}

func (m *LineageModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps
	log.Printf("🔗 Initializing Lineage Module...")

	repo := persistence.NewPostgresRepository(deps.DB)

	// Get FindingsProvider from dependencies
	var findingsProvider interfaces.FindingsProvider
	if deps.FindingsProvider != nil {
		findingsProvider = deps.FindingsProvider
	} else {
		return fmt.Errorf("FindingsProvider dependency is required for Lineage Module")
	}

	m.semanticLineageService = service.NewSemanticLineageService(
		deps.Neo4jRepo,
		repo,
		findingsProvider,
	)

	m.graphHandler = api.NewGraphHandler(m.semanticLineageService)
	m.lineageHandler = api.NewLineageHandler(m.semanticLineageService)

	log.Printf("✅ Lineage Module initialized")
	return nil
}

func (m *LineageModule) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/lineage", m.lineageHandler.GetLineage)
	router.GET("/lineage/stats", m.lineageHandler.GetLineageStats)
	router.POST("/lineage/sync", m.lineageHandler.SyncLineage)

	graph := router.Group("/graph")
	{
		graph.GET("/semantic", m.graphHandler.GetSemanticGraph)
	}

	log.Printf("🔗 Lineage routes registered")
}

func (m *LineageModule) Shutdown() error {
	log.Printf("🔌 Shutting down Lineage Module...")
	return nil
}

// GetSemanticLineageService returns the semantic lineage service for inter-module use
// Returns as LineageSync interface for loose coupling
func (m *LineageModule) GetSemanticLineageService() interfaces.LineageSync {
	return m.semanticLineageService
}

func NewLineageModule() *LineageModule {
	return &LineageModule{}
}
