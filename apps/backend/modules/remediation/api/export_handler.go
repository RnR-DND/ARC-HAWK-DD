package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/remediation/service"
	"github.com/gin-gonic/gin"
)

// RemediationExportHandler handles PDF/XLSX export of remediation history.
type RemediationExportHandler struct {
	svc *service.RemediationService
}

func NewRemediationExportHandler(svc *service.RemediationService) *RemediationExportHandler {
	return &RemediationExportHandler{svc: svc}
}

// ExportReport godoc
// GET /api/v1/remediation/export?format=pdf|xlsx
func (h *RemediationExportHandler) ExportReport(c *gin.Context) {
	format := c.DefaultQuery("format", "pdf")

	data, contentType, err := h.svc.ExportReport(c.Request.Context(), format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	ext := "html"
	if format == "xlsx" {
		ext = "xlsx"
	}
	filename := "remediation-report." + ext

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, contentType, data)
}
