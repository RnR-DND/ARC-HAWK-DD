package analytics

import (
	"log"

	"github.com/arc-platform/backend/modules/analytics/api"
	"github.com/arc-platform/backend/modules/analytics/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

type AnalyticsModule struct {
	analyticsService *service.AnalyticsService
	analyticsHandler *api.AnalyticsHandler
	deps             *interfaces.ModuleDependencies
}

func (m *AnalyticsModule) Name() string {
	return "analytics"
}

func (m *AnalyticsModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps
	log.Printf("📊 Initializing Analytics Module...")

	repo := persistence.NewPostgresRepository(deps.DB)

	m.analyticsService = service.NewAnalyticsService(repo)
	m.analyticsHandler = api.NewAnalyticsHandler(m.analyticsService)

	log.Printf("✅ Analytics Module initialized")
	return nil
}

func (m *AnalyticsModule) RegisterRoutes(router *gin.RouterGroup) {
	analytics := router.Group("/analytics")
	{
		analytics.GET("/heatmap", m.analyticsHandler.GetPIIHeatmap)
		analytics.GET("/trends", m.analyticsHandler.GetRiskTrend)
	}
	log.Printf("📊 Analytics routes registered")
}

func (m *AnalyticsModule) Shutdown() error {
	log.Printf("🔌 Shutting down Analytics Module...")
	return nil
}

func NewAnalyticsModule() *AnalyticsModule {
	return &AnalyticsModule{}
}
