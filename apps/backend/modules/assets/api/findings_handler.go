package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/assets/service"
	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FindingsHandler handles findings requests
type FindingsHandler struct {
	service *service.FindingsService
	repo    *persistence.PostgresRepository
}

// NewFindingsHandler creates a new findings handler
func NewFindingsHandler(svc *service.FindingsService) *FindingsHandler {
	return &FindingsHandler{service: svc}
}

// NewFindingsHandlerWithRepo creates a new findings handler with repo access for facets queries.
func NewFindingsHandlerWithRepo(svc *service.FindingsService, repo *persistence.PostgresRepository) *FindingsHandler {
	return &FindingsHandler{service: svc, repo: repo}
}

// GetFindings handles GET /api/v1/findings
func (h *FindingsHandler) GetFindings(c *gin.Context) {
	// Parse query parameters
	// pii_type is an alias for pattern_name (sent by findings explorer UI)
	patternName := c.Query("pattern_name")
	if patternName == "" {
		patternName = c.Query("pii_type")
	}
	query := service.FindingsQuery{
		Severity:     c.Query("severity"),
		PatternName:  patternName,
		DataSource:   c.Query("data_source"),
		Search:       c.Query("search"),
		AssetName:    c.Query("asset"),  // asset name filter from UI
		ReviewStatus: c.Query("status"), // "Active" | "Suppressed" | "Remediated"
		SortBy:       c.DefaultQuery("sort_by", "created_at"),
		SortOrder:    c.DefaultQuery("sort_order", "desc"),
	}

	// Parse pagination
	if pageStr := c.DefaultQuery("page", "1"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil {
			query.Page = page
		}
	}

	if pageSizeStr := c.DefaultQuery("page_size", "20"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err == nil {
			query.PageSize = pageSize
		}
	}

	// Parse scan_run_id if provided
	if scanRunIDStr := c.Query("scan_run_id"); scanRunIDStr != "" {
		scanRunID, err := uuid.Parse(scanRunIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid scan_run_id format",
				"details": err.Error(),
			})
			return
		}
		query.ScanRunID = &scanRunID
	}

	// Parse asset_id if provided
	if assetIDStr := c.Query("asset_id"); assetIDStr != "" {
		assetID, err := uuid.Parse(assetIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid asset_id format",
				"details": err.Error(),
			})
			return
		}
		query.AssetID = &assetID
	}

	// Get findings
	response, err := h.service.GetFindings(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get findings",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// SubmitFeedback handles POST /api/v1/findings/:id/feedback
func (h *FindingsHandler) SubmitFeedback(c *gin.Context) {
	findingIDStr := c.Param("id")
	findingID, err := uuid.Parse(findingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid finding ID"})
		return
	}

	var request struct {
		FeedbackType           string `json:"feedback_type" binding:"required"`
		OriginalClassification string `json:"original_classification"`
		ProposedClassification string `json:"proposed_classification"`
		Comments               string `json:"comments"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create domain entity
	feedback := &entity.FindingFeedback{
		FindingID:              findingID,
		UserID:                 "user", // In real app, get from context/token
		FeedbackType:           request.FeedbackType,
		OriginalClassification: request.OriginalClassification,
		ProposedClassification: request.ProposedClassification,
		Comments:               request.Comments,
	}

	if err := h.service.SubmitFeedback(c.Request.Context(), feedback); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success"})
}

// GetFindingsFacets returns distinct filter options from actual findings data.
// GET /findings/facets
func (h *FindingsHandler) GetFindingsFacets(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "repo not configured"})
		return
	}

	// PII types
	piiRows, err := h.repo.GetDB().QueryContext(ctx,
		`SELECT DISTINCT pattern_name FROM findings WHERE tenant_id = $1 AND pattern_name IS NOT NULL ORDER BY pattern_name`,
		tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pii types"})
		return
	}
	defer piiRows.Close()
	var piiTypes []string
	for piiRows.Next() {
		var t string
		if err := piiRows.Scan(&t); err == nil && t != "" {
			piiTypes = append(piiTypes, t)
		}
	}
	if piiTypes == nil {
		piiTypes = []string{}
	}

	// Asset names
	assetRows, err := h.repo.GetDB().QueryContext(ctx,
		`SELECT DISTINCT a.name FROM findings f JOIN assets a ON f.asset_id = a.id WHERE f.tenant_id = $1 AND a.name IS NOT NULL ORDER BY a.name`,
		tenantID)
	if err != nil {
		// Non-fatal: return empty
		assetRows = nil
	}
	var assets []string
	if assetRows != nil {
		defer assetRows.Close()
		for assetRows.Next() {
			var n string
			if err := assetRows.Scan(&n); err == nil && n != "" {
				assets = append(assets, n)
			}
		}
	}
	if assets == nil {
		assets = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"pii_types":  piiTypes,
		"assets":     assets,
		"severities": []string{"Critical", "High", "Medium", "Low"},
	})
}
