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

// GenerateReport enqueues an async report job.
// POST /api/discovery/reports/generate
//
// Body: { "snapshot_id": <uuid|null>, "format": "html|csv|json|pdf" }
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"report_id": rep.ID,
		"status":    rep.Status,
		"format":    rep.Format,
	})
}

// GetReport returns the status and metadata of a report job.
// GET /api/discovery/reports/:id
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rep)
}

// DownloadReport streams the report content with the right Content-Type.
// GET /api/discovery/reports/:id/download
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, contentType, content)
}

// ListReports returns recent report jobs for the tenant.
// GET /api/discovery/reports?limit=50&offset=0
func (h *ReportHandler) ListReports(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	reps, err := h.repo.ListReports(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": reps, "count": len(reps)})
}
