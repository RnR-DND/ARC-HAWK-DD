package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/discovery/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ReportHandler serves report generation, status, and download endpoints.
type ReportHandler struct {
	report *service.ReportService
	repo   *service.Repo
}

// NewReportHandler creates a new report handler.
func NewReportHandler(report *service.ReportService, repo *service.Repo) *ReportHandler {
	return &ReportHandler{report: report, repo: repo}
}

// generateReportRequest is the JSON body for POST /reports/generate.
type generateReportRequest struct {
	SnapshotID *uuid.UUID `json:"snapshot_id"`
	Format     string     `json:"format"`
}

// GenerateReport godoc
// @Summary Enqueue a discovery report
// @Description Generates html, csv, json, or pdf report asynchronously. Poll GET /discovery/reports/:id for status.
// @Tags discovery
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param body body object true "{snapshot_id, format: html|csv|json|pdf}"
// @Success 202 {object} gin.H "report_id, status"
// @Security BearerAuth
// @Router /discovery/reports/generate [post]
func (h *ReportHandler) GenerateReport(c *gin.Context) {
	var req generateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	format := domain.ReportFormat(req.Format)
	switch format {
	case domain.ReportPDF, domain.ReportCSV, domain.ReportJSON, domain.ReportHTML:
		// ok
	case "":
		format = domain.ReportHTML
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be html, pdf, csv, or json"})
		return
	}

	// Get triggering user from auth middleware (optional).
	var requestedBy *uuid.UUID
	if userID, ok := c.Get("user_id"); ok {
		if uid, ok := userID.(uuid.UUID); ok {
			requestedBy = &uid
		}
	}

	rep, err := h.report.EnqueueReport(c.Request.Context(), req.SnapshotID, format, requestedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"report_id": rep.ID,
		"status":    rep.Status,
		"format":    rep.Format,
	})
}

// GetReport godoc
// @Summary Get report job status
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Report UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /discovery/reports/{id} [get]
func (h *ReportHandler) GetReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}
	rep, err := h.repo.GetReport(c.Request.Context(), id)
	if err == service.ErrReportNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, rep)
}

// DownloadReport godoc
// @Summary Download generated report file
// @Description Returns application/pdf, text/html, text/csv, or application/json depending on format
// @Tags discovery
// @Produce application/pdf
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Report UUID"
// @Success 200 {string} string "Report bytes"
// @Failure 425 {object} gin.H "Report not ready yet"
// @Security BearerAuth
// @Router /discovery/reports/{id}/download [get]
func (h *ReportHandler) DownloadReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}
	content, contentType, err := h.repo.GetReportContent(c.Request.Context(), id)
	if err == service.ErrReportNotReady {
		c.JSON(http.StatusTooEarly, gin.H{"error": "report not ready"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Data(http.StatusOK, contentType, content)
}

// ListReports godoc
// @Summary List recent report jobs
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param limit query int false "Max results (default 50)"
// @Param offset query int false "Offset"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /discovery/reports [get]
func (h *ReportHandler) ListReports(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	reps, err := h.repo.ListReports(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": reps, "count": len(reps)})
}
