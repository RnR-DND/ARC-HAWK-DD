package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/gin-gonic/gin"
)

// ClassificationHandler handles classification requests
type ClassificationHandler struct {
	service        *service.ClassificationService
	summaryService *service.ClassificationSummaryService
}

// NewClassificationHandler creates a new classification handler
func NewClassificationHandler(service *service.ClassificationService, summaryService *service.ClassificationSummaryService) *ClassificationHandler {
	return &ClassificationHandler{
		service:        service,
		summaryService: summaryService,
	}
}

// GetClassificationSummary handles GET /api/v1/classification/summary
func (h *ClassificationHandler) GetClassificationSummary(c *gin.Context) {
	summary, err := h.summaryService.GetClassificationSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get classification summary",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": summary,
	})
}

type ClassificationRequest struct {
	Text        string                 `json:"text" binding:"required"`
	PatternName string                 `json:"pattern_name" binding:"required"`
	FilePath    string                 `json:"file_path"`
	FileData    map[string]interface{} `json:"file_data"`
}

// Predict handles POST /api/v1/classification/predict
// It returns the classification result for a given text without persisting it.
func (h *ClassificationHandler) Predict(c *gin.Context) {
	var req ClassificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use match as text for simplicity in playground testing
	// In real scanner, match is the specific substring.
	// For regression, we pass sample text as match to test full logic.
	result := h.service.Classify(req.PatternName, req.FilePath, req.Text, req.FileData)

	c.JSON(http.StatusOK, gin.H{
		"classification": result,
	})
}
