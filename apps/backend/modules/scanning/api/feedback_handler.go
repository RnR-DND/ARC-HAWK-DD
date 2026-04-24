package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	authmiddleware "github.com/arc-platform/backend/modules/auth/middleware"
	"github.com/arc-platform/backend/modules/scanning/service"
)

// FeedbackHandler handles analyst feedback on PII findings.
type FeedbackHandler struct {
	feedbackSvc *service.FeedbackService
}

func NewFeedbackHandler(svc *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{feedbackSvc: svc}
}

type submitFeedbackRequest struct {
	CorrectionType string `json:"correction_type" binding:"required,oneof=false_positive false_negative confirmed"`
}

// SubmitFeedback POST /findings/:id/feedback
// SubmitFeedback godoc
// @Summary Submit analyst feedback on a finding
// @Tags findings
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Finding UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /findings/{id}/feedback [post]
func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	tenantID := authmiddleware.GetTenantIDFromToken(c).String()
	userID := authmiddleware.GetUserIDFromToken(c).String()
	findingID := c.Param("id")

	var req submitFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	if err := h.feedbackSvc.RecordCorrection(c.Request.Context(), tenantID, userID, findingID, req.CorrectionType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

// GetPatternPrecision GET /patterns/precision
// GetPatternPrecision godoc
// @Summary Get precision metrics for all patterns
// @Tags patterns
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /patterns/precision [get]
func (h *FeedbackHandler) GetPatternPrecision(c *gin.Context) {
	tenantID := authmiddleware.GetTenantIDFromToken(c).String()
	stats, err := h.feedbackSvc.GetPatternPrecisionStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, stats)
}
