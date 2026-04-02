package api

import (
	"log"
	"net/http"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/gin-gonic/gin"
)

// SDKIngestHandler handles SDK-verified finding ingestion
type SDKIngestHandler struct {
	ingestionService *service.IngestionService
}

func NewSDKIngestHandler(ingestionService *service.IngestionService) *SDKIngestHandler {
	return &SDKIngestHandler{
		ingestionService: ingestionService,
	}
}

// IngestVerified handles POST /api/v1/scans/ingest-verified
func (h *SDKIngestHandler) IngestVerified(c *gin.Context) {
	var input service.VerifiedScanInput

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("ERROR: Invalid ingest request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// H-5: Batch size limit
	if len(input.Findings) > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Batch size exceeds maximum of 10,000 findings"})
		return
	}

	// Validate input
	if len(input.Findings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No findings provided",
		})
		return
	}

	// Process findings
	ctx := c.Request.Context()
	if err := h.ingestionService.IngestSDKVerified(ctx, input); err != nil {
		log.Printf("ERROR: Failed to ingest findings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to ingest findings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"findings_count": len(input.Findings),
		"scan_id":        input.ScanID,
		"message":        "SDK-verified findings ingested successfully",
	})
}
