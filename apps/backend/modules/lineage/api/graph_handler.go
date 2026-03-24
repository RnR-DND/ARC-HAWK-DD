package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/lineage/service"
	"github.com/gin-gonic/gin"
)

// GraphHandler handles semantic graph requests
type GraphHandler struct {
	semanticLineageService *service.SemanticLineageService
}

// NewGraphHandler creates a new graph handler
func NewGraphHandler(semanticLineageService *service.SemanticLineageService) *GraphHandler {
	return &GraphHandler{
		semanticLineageService: semanticLineageService,
	}
}

// GetSemanticGraph handles GET /api/v1/graph/semantic
func (h *GraphHandler) GetSemanticGraph(c *gin.Context) {
	// Parse query parameters
	systemID := c.Query("system_id")
	riskLevel := c.Query("risk_level")
	category := c.Query("category")

	filters := service.SemanticGraphFilters{
		SystemID:  systemID,
		RiskLevel: riskLevel,
		Category:  category,
	}

	// Get semantic graph
	graph, err := h.semanticLineageService.GetSemanticGraph(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get semantic graph",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": graph,
		"meta": gin.H{
			"node_count": len(graph.Nodes),
			"edge_count": len(graph.Edges),
			"filters":    filters,
		},
	})
}
