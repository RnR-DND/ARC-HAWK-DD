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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, fp)
}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, fp)
}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  fps,
		"total": total,
		"page":  page,
		"limit": pageSize,
	})
}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deactivated"})
}

func (h *FPLearningHandler) GetStats(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr.(string))

	stats, err := h.service.GetStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *FPLearningHandler) CheckFalsePositive(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr.(string))

	var req entity.FPMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entity.FPMatchResponse{
		IsFalsePositive: isFP,
		LearningID:      learningID,
		Confidence:      100,
	})
}
