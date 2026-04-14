package connections

import (
	"fmt"
	"log"

	"github.com/arc-platform/backend/modules/connections/api"
	"github.com/arc-platform/backend/modules/connections/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/encryption"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

type ConnectionsModule struct {
	connectionService        *service.ConnectionService
	connectionSyncService    *service.ConnectionSyncService
	testConnectionService    *service.TestConnectionService
	scanOrchestrationService *service.ScanOrchestrationService

	connectionHandler        *api.ConnectionHandler
	connectionSyncHandler    *api.ConnectionSyncHandler
	scanOrchestrationHandler *api.ScanOrchestrationHandler

	deps *interfaces.ModuleDependencies
}

func (m *ConnectionsModule) Name() string {
	return "connections"
}

// Initialize initializes the connections module
func (m *ConnectionsModule) Initialize(deps *interfaces.ModuleDependencies) error {
	m.deps = deps // Keep this line as it's part of the original method's setup
	log.Println("Initializing Connections Module...")

	// Initialize encryption service
	encryptionService, err := encryption.NewEncryptionService()
	if err != nil {
		return fmt.Errorf("failed to initialize encryption service: %w", err)
	}

	// Initialize PostgreSQL repository
	pgRepo := persistence.NewPostgresRepository(deps.DB)

	// Initialize connection service with encryption and optional Vault
	m.connectionService = service.NewConnectionService(pgRepo, encryptionService, deps.VaultClient)

	// Initialize connection sync service
	m.connectionSyncService = service.NewConnectionSyncService(pgRepo, encryptionService)

	// Initialize test connection service
	m.testConnectionService = service.NewTestConnectionService(pgRepo, encryptionService)

	// Initialize scan orchestration service
	m.scanOrchestrationService = service.NewScanOrchestrationService(pgRepo)

	// Initialize handlers
	m.connectionHandler = api.NewConnectionHandler(m.connectionService, m.connectionSyncService, m.testConnectionService)
	m.connectionSyncHandler = api.NewConnectionSyncHandler(m.connectionSyncService)
	m.scanOrchestrationHandler = api.NewScanOrchestrationHandler(m.scanOrchestrationService)

	log.Println("✅ Connections Module initialized")
	return nil
}

func (m *ConnectionsModule) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/connections", m.connectionHandler.AddConnection)
	router.GET("/connections", m.connectionHandler.GetConnections)
	router.DELETE("/connections/:id", m.connectionHandler.DeleteConnection)
	router.POST("/connections/test", m.connectionHandler.TestConnection)
	router.POST("/connections/:id/test", m.connectionHandler.TestConnectionByID)

	// Connection sync routes
	router.POST("/connections/sync", m.connectionSyncHandler.SyncToScanner)
	router.GET("/connections/sync/validate", m.connectionSyncHandler.ValidateSync)

	scans := router.Group("/scans")
	{
		scans.POST("/scan-all", m.scanOrchestrationHandler.ScanAllAssets)
		scans.GET("/status", m.scanOrchestrationHandler.GetScanStatus)
		scans.GET("/jobs", m.scanOrchestrationHandler.GetAllJobs)
	}

	log.Printf("🔌 Connections routes registered")
}

func (m *ConnectionsModule) Shutdown() error {
	log.Printf("🔌 Shutting down Connections Module...")
	return nil
}

func NewConnectionsModule() *ConnectionsModule {
	return &ConnectionsModule{}
}
