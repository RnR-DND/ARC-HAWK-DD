package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/shared/domain/entity"
	"github.com/arc-platform/backend/modules/assets/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FindingsHandler handles findings requests
type FindingsHandler struct {
	service *service.FindingsService
}

// NewFindingsHandler creates a new findings handler
func NewFindingsHandler(service *service.FindingsService) *FindingsHandler {
	return &FindingsHandler{service: service}
}

// GetFindings handles GET /api/v1/findings
func (h *FindingsHandler) GetFindings(c *gin.Context) {
	// Parse query parameters
	query := service.FindingsQuery{
		Severity:    c.Query("severity"),
		PatternName: c.Query("pattern_name"),
		DataSource:  c.Query("data_source"),
		SortBy:      c.DefaultQuery("sort_by", "created_at"),
		SortOrder:   c.DefaultQuery("sort_order", "desc"),
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
