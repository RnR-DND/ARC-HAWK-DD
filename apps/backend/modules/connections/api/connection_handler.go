package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/connections/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ConnectionHandler handles HTTP requests for connection management
type ConnectionHandler struct {
	service           *service.ConnectionService
	syncService       *service.ConnectionSyncService
	testConnectionSvc *service.TestConnectionService
}

// NewConnectionHandler creates a new connection handler
func NewConnectionHandler(s *service.ConnectionService, syncService *service.ConnectionSyncService, testSvc *service.TestConnectionService) *ConnectionHandler {
	return &ConnectionHandler{
		service:           s,
		syncService:       syncService,
		testConnectionSvc: testSvc,
	}
}

// AddConnectionRequest represents the request body for adding a connection
type AddConnectionRequest struct {
	SourceType  string                 `json:"source_type" binding:"required,oneof=postgresql mysql mongodb s3 filesystem redis slack"`
	ProfileName string                 `json:"profile_name" binding:"required,min=1,max=50,alphanum"`
	Config      map[string]interface{} `json:"config" binding:"required"`
}

// AddConnection handles POST /api/v1/connections
func (h *ConnectionHandler) AddConnection(c *gin.Context) {
	var req AddConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Get user from auth context (Phase 2 - Authentication)
	createdBy := "system"

	conn, err := h.service.AddConnection(c.Request.Context(), req.SourceType, req.ProfileName, req.Config, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add connection: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      conn.ID,
		"status":  "success",
		"message": "Connection added successfully. Validation pending.",
	})

	// Auto-sync to scanner YAML in background
	go func() {
		if err := h.syncService.SyncToYAML(c.Request.Context()); err != nil {
			// Log error but don't fail the request
			println("WARNING: Failed to sync connection to scanner:", err.Error())
		}
	}()
}

// GetConnections handles GET /api/v1/connections
func (h *ConnectionHandler) GetConnections(c *gin.Context) {
	connections, err := h.service.GetConnections(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get connections: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"connections": connections})
}

// DeleteConnection handles DELETE /api/v1/connections/:id
func (h *ConnectionHandler) DeleteConnection(c *gin.Context) {
	id := c.Param("id")

	// Parse UUID
	uuid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid connection ID"})
		return
	}

	if err := h.service.DeleteConnection(c.Request.Context(), uuid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete connection: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Connection deleted successfully",
	})

	// Auto-sync to scanner YAML in background
	go func() {
		if err := h.syncService.SyncToYAML(c.Request.Context()); err != nil {
			// Log error but don't fail the request
			println("WARNING: Failed to sync after deletion:", err.Error())
		}
	}()
}

// TestConnectionRequest represents the request body for testing a connection
type TestConnectionRequest struct {
	SourceType string                 `json:"source_type" binding:"required,oneof=postgresql mysql mongodb s3 filesystem redis slack"`
	Config     map[string]interface{} `json:"config" binding:"required"`
}

// TestConnection handles POST /api/v1/connections/test
func (h *ConnectionHandler) TestConnection(c *gin.Context) {
	var req TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.testConnectionSvc.TestConnectionByConfig(c.Request.Context(), req.SourceType, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to test connection: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// TestConnectionByID handles POST /api/v1/connections/:id/test
func (h *ConnectionHandler) TestConnectionByID(c *gin.Context) {
	id := c.Param("id")

	result, err := h.testConnectionSvc.TestConnection(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
