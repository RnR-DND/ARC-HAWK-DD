package api

import (
	"encoding/json"
	"net/http"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/gin-gonic/gin"
)

// IngestionHandler handles scan ingestion requests
type IngestionHandler struct {
	service *service.IngestionService
}

// NewIngestionHandler creates a new ingestion handler
func NewIngestionHandler(service *service.IngestionService) *IngestionHandler {
	return &IngestionHandler{service: service}
}

// IngestScan handles POST /api/v1/scans/ingest
func (h *IngestionHandler) IngestScan(c *gin.Context) {
	var input service.HawkeyeScanInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate input — Sources map is populated by UnmarshalJSON
	if len(input.Sources) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No findings in scan input",
		})
		return
	}

	// Process ingestion
	result, err := h.service.IngestScan(c.Request.Context(), &input)
	if err != nil {
		// Do not expose internal error details to the client
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to ingest scan",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Scan ingested successfully",
		"data":    result,
	})
}

// GetScanStatus handles GET /api/v1/scans/:id
func (h *IngestionHandler) GetScanStatus(c *gin.Context) {
	// Implementation for getting scan status
	c.JSON(http.StatusOK, gin.H{
		"message": "Get scan status endpoint",
	})
}

// GetLatestScan handles GET /api/v1/scans/latest
func (h *IngestionHandler) GetLatestScan(c *gin.Context) {
	scanRun, err := h.service.GetLatestScan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch latest scan",
		})
		return
	}

	if scanRun == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "No scans found",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": scanRun,
	})
}

// Ensure handler can marshal to JSON
func (h *IngestionHandler) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{}{})
}

// ClearScanData handles DELETE /api/v1/scans/clear
// Clears all previous scan data for clean scan-replace workflow.
// Requires admin role to prevent accidental or unauthorized data loss.
func (h *IngestionHandler) ClearScanData(c *gin.Context) {
	role, _ := c.Get("user_role")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Admin role required to clear scan data",
		})
		return
	}

	err := h.service.ClearAllScanData(c.Request.Context())
	if err != nil {
		// Do not expose internal error details to the client
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to clear scan data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Previous scan data cleared successfully",
	})
}
