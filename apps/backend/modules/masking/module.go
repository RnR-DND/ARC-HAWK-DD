// Package masking applies configurable redaction policies (partial mask, hash, nullify) to PII findings at the source.
package masking

import (
	"log"

	"github.com/arc-platform/backend/modules/masking/api"
	"github.com/arc-platform/backend/modules/masking/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

type MaskingModule struct {
	maskingService *service.MaskingService
	maskingHandler *api.MaskingHandler
	deps           *interfaces.ModuleDependencies
}

func (m *MaskingModule) Name() string {
	return "masking"
}

func (m *MaskingModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps
	log.Printf("🔒 Initializing Masking Module...")

	repo := persistence.NewPostgresRepository(deps.DB)
	maskingAuditRepo := persistence.NewMaskingAuditRepository(deps.DB)

	m.maskingService = service.NewMaskingService(repo, repo, maskingAuditRepo)
	m.maskingHandler = api.NewMaskingHandler(m.maskingService)

	log.Printf("✅ Masking Module initialized")
	return nil
}

func (m *MaskingModule) RegisterRoutes(router *gin.RouterGroup) {
	masking := router.Group("/masking")
	{
		masking.POST("/mask-asset", m.maskingHandler.MaskAsset)
		masking.GET("/status/:assetId", m.maskingHandler.GetMaskingStatus)
		masking.GET("/audit/:assetId", m.maskingHandler.GetMaskingAuditLog)
	}
	log.Printf("🔒 Masking routes registered")
}

func (m *MaskingModule) Shutdown() error {
	log.Printf("🔌 Shutting down Masking Module...")
	return nil
}

func NewMaskingModule() *MaskingModule {
	return &MaskingModule{}
}
