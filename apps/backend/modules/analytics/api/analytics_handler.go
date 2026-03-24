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

// GetRiskTrend returns risk trends over time
// GET /api/v1/analytics/trends?days=30
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
