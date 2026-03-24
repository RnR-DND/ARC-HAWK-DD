package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/connections/service"
	"github.com/gin-gonic/gin"
)

// ScanOrchestrationHandler handles scan orchestration endpoints
type ScanOrchestrationHandler struct {
	service *service.ScanOrchestrationService
}

// NewScanOrchestrationHandler creates a new scan orchestration handler
func NewScanOrchestrationHandler(service *service.ScanOrchestrationService) *ScanOrchestrationHandler {
	return &ScanOrchestrationHandler{
		service: service,
	}
}

// ScanAllAssets triggers scans for all assets
// POST /api/v1/scans/scan-all
func (h *ScanOrchestrationHandler) ScanAllAssets(c *gin.Context) {
	status, err := h.service.ScanAllAssets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scan initiated for all assets",
		"status":  status,
	})
}

// GetScanStatus returns the current scan status
// GET /api/v1/scans/status
func (h *ScanOrchestrationHandler) GetScanStatus(c *gin.Context) {
	status := h.service.GetScanStatus(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

// GetAllJobs returns all scan jobs
// GET /api/v1/scans/jobs
func (h *ScanOrchestrationHandler) GetAllJobs(c *gin.Context) {
	jobs := h.service.GetAllJobs(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
	})
}
