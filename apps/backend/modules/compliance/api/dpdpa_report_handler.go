package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/compliance/service"
	"github.com/gin-gonic/gin"
)

// DPDPAReportHandler serves DPDPA obligation gap report endpoints.
type DPDPAReportHandler struct {
	obligationSvc *service.DPDPAObligationService
	reportSvc     *service.ReportService
}

// NewDPDPAReportHandler creates a new DPDPA report handler.
func NewDPDPAReportHandler(
	obligationSvc *service.DPDPAObligationService,
	reportSvc *service.ReportService,
) *DPDPAReportHandler {
	return &DPDPAReportHandler{
		obligationSvc: obligationSvc,
		reportSvc:     reportSvc,
	}
}

// GetObligationGaps godoc
// @Summary Get DPDPA obligation gaps
// @Description Returns the raw DPDPA obligation gap data as JSON
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/dpdpa/gaps [get]
func (h *DPDPAReportHandler) GetObligationGaps(c *gin.Context) {
	report, err := h.obligationSvc.BuildGapReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// GenerateHTMLReport godoc
// @Summary Generate DPDPA HTML compliance report
// @Description Returns an HTML compliance gap report suitable for print-to-PDF
// @Tags compliance
// @Accept json
// @Produce html
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/dpdpa/report [get]
func (h *DPDPAReportHandler) GenerateHTMLReport(c *gin.Context) {
	html, err := h.reportSvc.GenerateHTMLReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", html)
}
