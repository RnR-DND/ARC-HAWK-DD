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

// GetObligationGaps returns the raw DPDPA obligation gap data as JSON.
// GET /api/v1/compliance/dpdpa/gaps
func (h *DPDPAReportHandler) GetObligationGaps(c *gin.Context) {
	report, err := h.obligationSvc.BuildGapReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// GenerateHTMLReport returns an HTML compliance gap report suitable for print-to-PDF.
// GET /api/v1/compliance/dpdpa/report
func (h *DPDPAReportHandler) GenerateHTMLReport(c *gin.Context) {
	html, err := h.reportSvc.GenerateHTMLReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", html)
}
