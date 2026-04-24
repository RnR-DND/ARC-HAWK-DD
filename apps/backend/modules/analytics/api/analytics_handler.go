package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/analytics/service"
	"github.com/gin-gonic/gin"
)

// AnalyticsHandler handles analytics endpoints
type AnalyticsHandler struct {
	service *service.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(service *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
	}
}

// GetPIIHeatmap returns the PII distribution heatmap
// GET /api/v1/analytics/heatmap
// GetPIIHeatmap godoc
// @Summary Get PII type vs source heatmap
// @Tags analytics
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /analytics/heatmap [get]
func (h *AnalyticsHandler) GetPIIHeatmap(c *gin.Context) {
	heatmap, err := h.service.GetPIIHeatmap(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, heatmap)
}

// GetRiskDistribution returns the count of findings per severity level.
// GET /api/v1/analytics/risk-distribution
// GetRiskDistribution godoc
// @Summary Get risk score distribution across assets
// @Tags analytics
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /analytics/risk-distribution [get]
func (h *AnalyticsHandler) GetRiskDistribution(c *gin.Context) {
	result, err := h.service.GetRiskDistribution(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetRiskTrend returns risk trends over time
// GET /api/v1/analytics/trends?days=30
// GetRiskTrend godoc
// @Summary Get risk score trend over time
// @Tags analytics
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param days query int false "Lookback window in days (default 30)"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /analytics/trends [get]
func (h *AnalyticsHandler) GetRiskTrend(c *gin.Context) {
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 {
			days = d
		}
	}

	trend, err := h.service.GetRiskTrend(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, trend)
}
