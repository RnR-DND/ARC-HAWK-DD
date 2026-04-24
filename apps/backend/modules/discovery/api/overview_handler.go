// Package api contains HTTP handlers for the discovery module.
//
// All handlers extract tenant ID via the same middleware that the rest of the app
// uses (auth/middleware/tenant_middleware.go), and pass the request context through
// to services. Tenant isolation is enforced at the repo layer via persistence.EnsureTenantID.
package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/discovery/service"
	"github.com/gin-gonic/gin"
)

// OverviewHandler serves the dashboard overview endpoint.
type OverviewHandler struct {
	inventory *service.InventoryService
	snapshot  *service.SnapshotService
	risk      *service.RiskScoringService
	repo      *service.Repo
}

// NewOverviewHandler creates a new overview handler.
func NewOverviewHandler(
	inventory *service.InventoryService,
	snapshot *service.SnapshotService,
	risk *service.RiskScoringService,
	repo *service.Repo,
) *OverviewHandler {
	return &OverviewHandler{
		inventory: inventory,
		snapshot:  snapshot,
		risk:      risk,
		repo:      repo,
	}
}

// GetOverview returns the dashboard payload: KPI cards, hotspots, trend.
// GET /api/discovery/overview
// GetOverview godoc
// @Summary Get discovery dashboard overview
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /discovery/overview [get]
func (h *OverviewHandler) GetOverview(c *gin.Context) {
	summary, err := h.inventory.GetOverviewSummary(c.Request.Context(), h.repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, summary)
}
