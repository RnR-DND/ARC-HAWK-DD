package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db        *sql.DB
	neo4jRepo *persistence.Neo4jRepository
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sql.DB, neo4jRepo *persistence.Neo4jRepository) *HealthHandler {
	return &HealthHandler{
		db:        db,
		neo4jRepo: neo4jRepo,
	}
}

// ComponentHealth represents the health status of a system component
type ComponentHealth struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // "online", "degraded", "offline"
	LastCheck time.Time `json:"last_check"`
	Message   string    `json:"message,omitempty"`
	Details   string    `json:"details,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status     string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Components []ComponentHealth `json:"components"`
	Timestamp  time.Time         `json:"timestamp"`
}

// GetComponentsHealth returns the health status of all system components
// GET /api/v1/health/components
func (h *HealthHandler) GetComponentsHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	components := []ComponentHealth{}
	overallHealthy := true
	degraded := false

	// Check PostgreSQL Database
	dbHealth := h.checkDatabase(ctx)
	components = append(components, dbHealth)
	if dbHealth.Status == "offline" {
		overallHealthy = false
	} else if dbHealth.Status == "degraded" {
		degraded = true
	}

	// Check Neo4j Graph Database
	neo4jHealth := h.checkNeo4j(ctx)
	components = append(components, neo4jHealth)
	if neo4jHealth.Status == "offline" {
		overallHealthy = false
	} else if neo4jHealth.Status == "degraded" {
		degraded = true
	}

	// Check Scanner (simplified - checks if we can query scans table)
	scannerHealth := h.checkScanner(ctx)
	components = append(components, scannerHealth)
	if scannerHealth.Status == "offline" {
		degraded = true // Scanner offline is degraded, not critical
	}

	// Determine overall status
	status := "healthy"
	if !overallHealthy {
		status = "unhealthy"
	} else if degraded {
		status = "degraded"
	}

	response := HealthResponse{
		Status:     status,
		Components: components,
		Timestamp:  time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

func (h *HealthHandler) checkDatabase(ctx context.Context) ComponentHealth {
	health := ComponentHealth{
		Name:      "PostgreSQL Database",
		LastCheck: time.Now(),
	}

	if err := h.db.Ping(); err != nil {
		health.Status = "offline"
		health.Message = "Database connection failed"
		health.Details = "Unable to ping PostgreSQL"
		return health
	}

	// Check if we can query
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM findings LIMIT 1").Scan(&count)
	if err != nil {
		health.Status = "degraded"
		health.Message = "Database connected but queries failing"
		return health
	}

	health.Status = "online"
	health.Message = "Database operational"
	return health
}

func (h *HealthHandler) checkNeo4j(ctx context.Context) ComponentHealth {
	health := ComponentHealth{
		Name:      "Neo4j Graph Database",
		LastCheck: time.Now(),
	}

	driver := h.neo4jRepo.GetDriver()
	if driver == nil {
		health.Status = "offline"
		health.Message = "Neo4j driver not initialized"
		return health
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		health.Status = "offline"
		health.Message = "Neo4j connection failed"
		health.Details = "Unable to verify connectivity"
		return health
	}

	health.Status = "online"
	health.Message = "Graph database operational"
	return health
}

func (h *HealthHandler) checkScanner(ctx context.Context) ComponentHealth {
	health := ComponentHealth{
		Name:      "Scanner Service",
		LastCheck: time.Now(),
	}

	// Check if scans table is accessible and has recent activity
	var lastScanTime *time.Time
	err := h.db.QueryRow(`
		SELECT MAX(created_at)
		FROM scan_runs
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&lastScanTime)

	if err != nil {
		health.Status = "degraded"
		health.Message = "Scanner status unknown"
		health.Details = "Unable to query recent scans"
		return health
	}

	if lastScanTime == nil {
		health.Status = "degraded"
		health.Message = "No recent scan activity"
		health.Details = "No scans in the last hour"
	} else {
		health.Status = "online"
		health.Message = "Scanner operational"
		health.Details = "Recent scan activity detected"
	}

	return health
}
