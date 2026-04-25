// Package connections manages data-source connector configurations, credential encryption, health checks, and available-type discovery.
package connections

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

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
	m.testConnectionService = service.NewTestConnectionService(pgRepo, encryptionService, deps.VaultClient)

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
	router.GET("/connections/available-types", m.connectionHandler.AvailableSourceTypes)
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

	// Connector health: last scan time/status/findings per profile
	db := m.deps.DB
	router.GET("/connections/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()

		type ConnHealth struct {
			ProfileName    string     `json:"profile_name"`
			SourceType     string     `json:"source_type"`
			LastScanTime   *time.Time `json:"last_scan_time"`
			LastScanStatus string     `json:"last_scan_status"`
			FindingsCount  int        `json:"findings_count"`
			Status         string     `json:"status"` // ok | stale | never
		}

		rows, err := db.QueryContext(ctx, `
			SELECT
				c.profile_name,
				c.source_type,
				sr.scan_started_at,
				sr.status,
				COALESCE(sr.total_findings, 0)
			FROM connections c
			LEFT JOIN LATERAL (
				SELECT scan_started_at, status, total_findings
				FROM scan_runs
				WHERE profile_name = c.profile_name
				ORDER BY scan_started_at DESC
				LIMIT 1
			) sr ON true
			ORDER BY c.profile_name
		`)
		if err != nil {
			log.Printf("ERROR: connections health query: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		defer rows.Close()

		result := []ConnHealth{}
		for rows.Next() {
			var h ConnHealth
			var lastScan *time.Time
			var lastStatus *string
			var findings *int
			if err := rows.Scan(&h.ProfileName, &h.SourceType, &lastScan, &lastStatus, &findings); err != nil {
				continue
			}
			h.LastScanTime = lastScan
			if lastStatus != nil {
				h.LastScanStatus = *lastStatus
			}
			if findings != nil {
				h.FindingsCount = *findings
			}
			switch {
			case lastScan == nil:
				h.Status = "never"
			case time.Since(*lastScan) > 72*time.Hour:
				h.Status = "stale"
			default:
				h.Status = "ok"
			}
			result = append(result, h)
		}
		c.JSON(http.StatusOK, gin.H{"health": result})
	})

	log.Printf("🔌 Connections routes registered")
}

func (m *ConnectionsModule) Shutdown() error {
	log.Printf("🔌 Shutting down Connections Module...")
	return nil
}

func NewConnectionsModule() *ConnectionsModule {
	return &ConnectionsModule{}
}
