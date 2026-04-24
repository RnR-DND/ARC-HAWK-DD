package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/fplearning/entity"
	"github.com/arc-platform/backend/modules/fplearning/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FPLearningHandler struct {
	service *service.FPLearningService
}

func NewFPLearningHandler(repo *persistence.PostgresRepository) *FPLearningHandler {
	return &FPLearningHandler{
		service: service.NewFPLearningService(repo),
	}
}

type CreateFPLearningRequest struct {
	AssetID         uuid.UUID  `json:"asset_id" binding:"required"`
	PatternName     string     `json:"pattern_name" binding:"required"`
	PIIType         string     `json:"pii_type" binding:"required"`
	FieldName       string     `json:"field_name"`
	FieldPath       string     `json:"field_path"`
	MatchedValue    string     `json:"matched_value" binding:"required"`
	Justification   string     `json:"justification"`
	SourceFindingID *uuid.UUID `json:"source_finding_id"`
}

// MarkFalsePositive godoc
// @Summary Mark a finding as false positive
// @Tags fp-learning
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /fp/false-positives [post]
func (h *FPLearningHandler) MarkFalsePositive(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr.(string))

	var req CreateFPLearningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	sourceFindingID := req.SourceFindingID
	if sourceFindingID == nil || *sourceFindingID == uuid.Nil {
		sourceFindingID = nil
	}

	fp, err := h.service.CreateFalsePositive(
		c.Request.Context(),
		tenantID,
		userID.(uuid.UUID),
		req.AssetID,
		req.PatternName,
		req.PIIType,
		req.FieldName,
		req.FieldPath,
		req.MatchedValue,
		req.Justification,
		sourceFindingID,
		nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, fp)
}

// MarkConfirmed godoc
// @Summary Confirm a finding as true PII
// @Tags fp-learning
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /fp/confirmed [post]
func (h *FPLearningHandler) MarkConfirmed(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr.(string))

	var req CreateFPLearningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	sourceFindingID := req.SourceFindingID
	if sourceFindingID == nil || *sourceFindingID == uuid.Nil {
		sourceFindingID = nil
	}

	fp, err := h.service.CreateConfirmed(
		c.Request.Context(),
		tenantID,
		userID.(uuid.UUID),
		req.AssetID,
		req.PatternName,
		req.PIIType,
		req.FieldName,
		req.FieldPath,
		req.MatchedValue,
		sourceFindingID,
		nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, fp)
}

// ListFPLearnings godoc
// @Summary List active FP learning entries
// @Tags fp-learning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /fp/learnings [get]
func (h *FPLearningHandler) ListFPLearnings(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr.(string))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var filter entity.FPLearningFilter
	if assetIDStr := c.Query("asset_id"); assetIDStr != "" {
		assetID, _ := uuid.Parse(assetIDStr)
		filter.AssetID = &assetID
	}
	if patternName := c.Query("pattern_name"); patternName != "" {
		filter.PatternName = patternName
	}
	if piiType := c.Query("pii_type"); piiType != "" {
		filter.PIIType = piiType
	}

	fps, total, err := h.service.GetFPLearnings(c.Request.Context(), tenantID, filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  fps,
		"total": total,
		"page":  page,
		"limit": pageSize,
	})
}

// GetFPLearning godoc
// @Summary Get a specific FP learning entry
// @Tags fp-learning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Learning UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /fp/learnings/{id} [get]
func (h *FPLearningHandler) GetFPLearning(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	fp, err := h.service.GetFPLearningByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, fp)
}

// DeactivateFPLearning godoc
// @Summary Soft-delete an FP learning entry
// @Tags fp-learning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Learning UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /fp/learnings/{id} [delete]
func (h *FPLearningHandler) DeactivateFPLearning(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.service.DeactivateFPLearning(c.Request.Context(), id, userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deactivated"})
}

// GetStats godoc
// @Summary Get FP learning precision/recall stats
// @Tags fp-learning
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /fp/stats [get]
func (h *FPLearningHandler) GetStats(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr.(string))

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CheckFalsePositive godoc
// @Summary Check if field/pattern is a known false positive
// @Tags fp-learning
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param body body object true "{pattern_name, field_name, asset_id}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /fp/check [post]
func (h *FPLearningHandler) CheckFalsePositive(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr.(string))

	var req entity.FPMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	isFP, learningID, err := h.service.CheckAndSuppressFinding(
		c.Request.Context(),
		tenantID,
		tenantID,
		req.AssetID,
		req.PatternName,
		req.PIIType,
		req.FieldPath,
		req.MatchedValue,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, entity.FPMatchResponse{
		IsFalsePositive: isFP,
		LearningID:      learningID,
		Confidence:      100,
	})
}
