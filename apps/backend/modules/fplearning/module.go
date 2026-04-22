package fplearning

import (
	"log"

	"github.com/arc-platform/backend/modules/fplearning/api"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

// FPlearningModule implements adaptive PII pattern learning (false-positive feedback loop).
type FPlearningModule struct {
	deps    *interfaces.ModuleDependencies
	handler *api.FPLearningHandler
}

// NewFPlearningModule creates a new fingerprint learning module.
func NewFPlearningModule() *FPlearningModule {
	return &FPlearningModule{}
}

// Name returns the module name.
func (m *FPlearningModule) Name() string {
	return "fplearning"
}

// Initialize sets up the module.
func (m *FPlearningModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps
	log.Printf("🧠 Initializing Fingerprint Learning Module...")

	repo := persistence.NewPostgresRepository(deps.DB)
	m.handler = api.NewFPLearningHandler(repo)

	log.Printf("✅ Fingerprint Learning Module initialized")
	return nil
}

// RegisterRoutes registers the module's HTTP routes.
func (m *FPlearningModule) RegisterRoutes(router *gin.RouterGroup) {
	fp := router.Group("/fplearning")
	{
		fp.POST("/false-positives", m.handler.MarkFalsePositive)
		fp.POST("/confirmed", m.handler.MarkConfirmed)
		fp.GET("/learnings", m.handler.ListFPLearnings)
		fp.GET("/learnings/:id", m.handler.GetFPLearning)
		fp.DELETE("/learnings/:id", m.handler.DeactivateFPLearning)
		fp.GET("/stats", m.handler.GetStats)
		fp.POST("/check", m.handler.CheckFalsePositive)
	}
	log.Printf("🧠 Fingerprint Learning routes registered")
}

// Shutdown cleans up resources.
func (m *FPlearningModule) Shutdown() error {
	log.Printf("🔌 Shutting down Fingerprint Learning Module...")
	return nil
}
