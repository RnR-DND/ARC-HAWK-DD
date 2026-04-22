package api

import (
	"context"
	"log"
	"net/http"
	"regexp"

	"github.com/arc-platform/backend/modules/connections/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var profileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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
	Name        string         `json:"name"`
	SourceType  string         `json:"source_type" binding:"required,oneof=postgresql mysql mongodb redis sqlite oracle mssql firebase couchdb s3 gcs azure_blob gdrive gdrive_workspace bigquery snowflake redshift kafka kinesis filesystem csv_excel pdf docx pptx html_files email_files parquet orc avro scanned_images text slack salesforce hubspot jira ms_teams"`
	ProfileName string         `json:"profile_name" binding:"required,min=1,max=50"`
	Config      map[string]any `json:"config" binding:"required"`
}

// AddConnection handles POST /api/v1/connections
func (h *ConnectionHandler) AddConnection(c *gin.Context) {
	var req AddConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !profileNameRegex.MatchString(req.ProfileName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile_name must contain only letters, digits, underscores, or hyphens"})
		return
	}
	createdBy := c.GetString("user_email")
	if createdBy == "" {
		createdBy = c.GetString("user_id")
	}
	if createdBy == "" {
		createdBy = "system"
	}

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
		if err := h.syncService.SyncToYAML(context.Background()); err != nil {
			log.Printf("failed to sync connection to scanner: %v", err)
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
		if err := h.syncService.SyncToYAML(context.Background()); err != nil {
			log.Printf("failed to sync after deletion: %v", err)
		}
	}()
}

// TestConnectionRequest represents the request body for testing a connection
type TestConnectionRequest struct {
	SourceType string         `json:"source_type" binding:"required,oneof=postgresql mysql mongodb redis sqlite oracle mssql firebase couchdb s3 gcs azure_blob gdrive gdrive_workspace bigquery snowflake redshift kafka kinesis filesystem csv_excel pdf docx pptx html_files email_files parquet orc avro scanned_images text slack salesforce hubspot jira ms_teams"`
	Config     map[string]any `json:"config" binding:"required"`
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

// AvailableSourceTypes handles GET /api/v1/connections/available-types
// Returns all supported connector types with display metadata.
func (h *ConnectionHandler) AvailableSourceTypes(c *gin.Context) {
	types := []map[string]string{
		{"source_type": "postgresql", "display_name": "PostgreSQL", "category": "databases", "icon": "database"},
		{"source_type": "mysql", "display_name": "MySQL", "category": "databases", "icon": "database"},
		{"source_type": "mongodb", "display_name": "MongoDB", "category": "databases", "icon": "database"},
		{"source_type": "redis", "display_name": "Redis", "category": "databases", "icon": "database"},
		{"source_type": "sqlite", "display_name": "SQLite", "category": "databases", "icon": "database"},
		{"source_type": "oracle", "display_name": "Oracle DB", "category": "databases", "icon": "database"},
		{"source_type": "mssql", "display_name": "SQL Server", "category": "databases", "icon": "database"},
		{"source_type": "firebase", "display_name": "Firebase", "category": "databases", "icon": "database"},
		{"source_type": "couchdb", "display_name": "CouchDB", "category": "databases", "icon": "database"},
		{"source_type": "s3", "display_name": "AWS S3", "category": "cloud", "icon": "cloud"},
		{"source_type": "gcs", "display_name": "Google Cloud Storage", "category": "cloud", "icon": "cloud"},
		{"source_type": "azure_blob", "display_name": "Azure Blob Storage", "category": "cloud", "icon": "cloud"},
		{"source_type": "gdrive", "display_name": "Google Drive", "category": "cloud", "icon": "cloud"},
		{"source_type": "gdrive_workspace", "display_name": "Google Workspace", "category": "cloud", "icon": "cloud"},
		{"source_type": "bigquery", "display_name": "BigQuery", "category": "warehouses", "icon": "warehouse"},
		{"source_type": "snowflake", "display_name": "Snowflake", "category": "warehouses", "icon": "warehouse"},
		{"source_type": "redshift", "display_name": "Redshift", "category": "warehouses", "icon": "warehouse"},
		{"source_type": "kafka", "display_name": "Apache Kafka", "category": "queues", "icon": "queue"},
		{"source_type": "kinesis", "display_name": "AWS Kinesis", "category": "queues", "icon": "queue"},
		{"source_type": "filesystem", "display_name": "File System", "category": "files", "icon": "folder"},
		{"source_type": "csv_excel", "display_name": "CSV / Excel", "category": "files", "icon": "file"},
		{"source_type": "pdf", "display_name": "PDF Files", "category": "files", "icon": "file"},
		{"source_type": "docx", "display_name": "Word Documents", "category": "files", "icon": "file"},
		{"source_type": "pptx", "display_name": "PowerPoint Files", "category": "files", "icon": "file"},
		{"source_type": "html_files", "display_name": "HTML Files", "category": "files", "icon": "file"},
		{"source_type": "email_files", "display_name": "Email Files (EML/MSG)", "category": "files", "icon": "file"},
		{"source_type": "parquet", "display_name": "Parquet Files", "category": "files", "icon": "file"},
		{"source_type": "orc", "display_name": "ORC Files", "category": "files", "icon": "file"},
		{"source_type": "avro", "display_name": "Avro Files", "category": "files", "icon": "file"},
		{"source_type": "scanned_images", "display_name": "Scanned Images (OCR)", "category": "files", "icon": "image"},
		{"source_type": "text", "display_name": "Text Files", "category": "files", "icon": "file"},
		{"source_type": "slack", "display_name": "Slack", "category": "saas", "icon": "chat"},
		{"source_type": "salesforce", "display_name": "Salesforce", "category": "saas", "icon": "crm"},
		{"source_type": "hubspot", "display_name": "HubSpot", "category": "saas", "icon": "crm"},
		{"source_type": "jira", "display_name": "Jira", "category": "saas", "icon": "ticket"},
		{"source_type": "ms_teams", "display_name": "Microsoft Teams", "category": "saas", "icon": "chat"},
	}
	c.JSON(http.StatusOK, gin.H{"types": types})
}
