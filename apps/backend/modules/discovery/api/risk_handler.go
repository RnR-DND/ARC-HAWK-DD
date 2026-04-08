package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/discovery/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RiskHandler serves risk-overview and per-asset score endpoints.
type RiskHandler struct {
	risk *service.RiskScoringService
	repo *service.Repo
}

// NewRiskHandler creates a new risk handler.
func NewRiskHandler(risk *service.RiskScoringService, repo *service.Repo) *RiskHandler {
	return &RiskHandler{risk: risk, repo: repo}
}

// GetRiskOverview returns the top hotspots and the weights used.
// GET /api/discovery/risk/overview
func (h *RiskHandler) GetRiskOverview(c *gin.Context) {
	hotspots, err := h.repo.ListTopRiskHotspots(c.Request.Context(), 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"hotspots": hotspots,
		"weights":  h.risk.Weights(),
	})
}

// GetRiskHotspots returns the top N highest-risk assets. limit query param.
// GET /api/discovery/risk/hotspots?limit=20
func (h *RiskHandler) GetRiskHotspots(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	hotspots, err := h.repo.ListTopRiskHotspots(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"hotspots": hotspots, "count": len(hotspots)})
}

// GetAssetRiskHistory recomputes and persists a fresh risk score for the asset, then returns it.
// GET /api/discovery/risk/scores/:assetId
func (h *RiskHandler) GetAssetRiskHistory(c *gin.Context) {
	assetID, err := uuid.Parse(c.Param("assetId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assetId"})
		return
	}
	score, err := h.risk.ScoreAsset(c.Request.Context(), assetID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, score)
}
