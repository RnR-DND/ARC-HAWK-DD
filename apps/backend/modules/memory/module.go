package memory

import (
	"log"

	"github.com/arc-platform/backend/modules/memory/api"
	"github.com/arc-platform/backend/modules/memory/service"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

type MemoryModule struct {
	svc     *service.MemoryService
	handler *api.MemoryHandler
}

func (m *MemoryModule) Name() string { return "memory" }

func (m *MemoryModule) Initialize(deps *interfaces.ModuleDependencies) error {
	log.Printf("🧠 Initializing Memory Module...")
	// If main.go pre-constructed the recorder and put it in deps, reuse it —
	// avoids duplicating the HTTP client and keeps a single instance the
	// scanning module also sees.
	if rec, ok := deps.MemoryRecorder.(*service.MemoryService); ok && rec != nil {
		m.svc = rec
	} else {
		m.svc = service.NewMemoryService(service.NewClientFromEnv())
	}
	m.handler = api.NewMemoryHandler(m.svc)
	if m.svc.Enabled() {
		log.Printf("✅ Memory Module initialized (supermemory.ai connected)")
	} else {
		log.Printf("⚠️  Memory Module initialized DISABLED (SUPERMEMORY_ENABLED!=true or key missing)")
	}
	return nil
}

func (m *MemoryModule) RegisterRoutes(router *gin.RouterGroup) {
	grp := router.Group("/memory")
	{
		grp.GET("/status", m.handler.GetStatus)
		grp.POST("/search", m.handler.Search)
	}
}

func (m *MemoryModule) Shutdown() error { return nil }

// Service exposes the domain service so other modules can call
// m.Service().RecordScanCompletion(...) after wiring in main.go.
func (m *MemoryModule) Service() *service.MemoryService { return m.svc }
