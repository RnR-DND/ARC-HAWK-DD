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
// @Summary Export remediation report
// @Description Exports remediation history as PDF (HTML) or XLSX
// @Tags remediation
// @Accept json
// @Produce application/octet-stream
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param format query string false "Export format: pdf or xlsx (default pdf)"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/export [get]
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
