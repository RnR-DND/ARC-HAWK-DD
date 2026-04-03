package api

import (
	"context"
	"log"
	"net/http"

	"github.com/arc-platform/backend/modules/lineage/service"
	"github.com/gin-gonic/gin"
)

// LineageHandlerV2 handles lineage-related requests
// Phase 3: Unified Neo4j-Only Lineage
type LineageHandlerV2 struct {
	semanticLineageService *service.SemanticLineageService
}

// NewLineageHandlerV2 creates a new lineage handler
func NewLineageHandlerV2(semanticLineageService *service.SemanticLineageService) *LineageHandlerV2 {
	return &LineageHandlerV2{
		semanticLineageService: semanticLineageService,
	}
}

// GetLineage handles GET /api/v1/lineage
// Returns the complete 3-level hierarchy (System → Asset → PII_Category)
func (h *LineageHandlerV2) GetLineage(c *gin.Context) {
	// Parse filters from query params
	systemFilter := c.Query("system")
	riskFilter := c.Query("risk") // Critical, High, Medium, Low

	// Get graph from Neo4j
	ctx := c.Request.Context()
	graph, err := h.semanticLineageService.GetSemanticGraph(ctx, service.SemanticGraphFilters{
		SystemID:  systemFilter,
		RiskLevel: riskFilter,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve lineage",
			"details": err.Error(),
		})
		return
	}

	// Calculate aggregations
	totalSystems := 0
	totalAssets := 0
	totalPIITypes := 0
	piiTypeAggregations := make(map[string]int)

	for _, node := range graph.Nodes {
		switch node.Type {
		case "system":
			totalSystems++
		case "asset":
			totalAssets++
		case "pii_category":
			totalPIITypes++
			if piiType, ok := node.Metadata["pii_type"].(string); ok {
				piiTypeAggregations[piiType]++
			}
		}
	}

	// Convert to response format
	byPIIType := make([]gin.H, 0, len(piiTypeAggregations))
	for piiType, count := range piiTypeAggregations {
		byPIIType = append(byPIIType, gin.H{
			"pii_type": piiType,
			"count":    count,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"hierarchy": gin.H{
				"nodes": graph.Nodes,
				"edges": graph.Edges,
			},
			"aggregations": gin.H{
				"total_systems":   totalSystems,
				"total_assets":    totalAssets,
				"total_pii_types": totalPIITypes,
				"by_pii_type":     byPIIType,
			},
		},
	})
}

// GetLineageStats handles GET /api/v1/lineage/stats
// Returns aggregated statistics from the graph
func (h *LineageHandlerV2) GetLineageStats(c *gin.Context) {
	ctx := c.Request.Context()

	graph, err := h.semanticLineageService.GetSemanticGraph(ctx, service.SemanticGraphFilters{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve stats",
			"details": err.Error(),
		})
		return
	}

	// Calculate stats from graph
	stats := map[string]interface{}{
		"total_systems":        countNodesByType(graph.Nodes, "system"),
		"total_assets":         countNodesByType(graph.Nodes, "asset"),
		"total_pii_categories": countNodesByType(graph.Nodes, "pii_category"),
		"total_edges":          len(graph.Edges),
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"stats":  stats,
	})
}

// Helper function to count nodes by type
func countNodesByType(nodes []service.SemanticNode, nodeType string) int {
	count := 0
	for _, node := range nodes {
		if node.Type == nodeType {
			count++
		}
	}
	return count
}

// SyncLineage handles POST /api/v1/lineage/sync
// Triggers full sync from PostgreSQL to Neo4j
func (h *LineageHandlerV2) SyncLineage(c *gin.Context) {
	// Launch sync in background to avoid timeout
	go func() {
		// Create a new background context since request context will be cancelled
		bgCtx := context.Background()
		if err := h.semanticLineageService.SyncLineage(bgCtx); err != nil {
			log.Printf("ERROR: Async lineage sync failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Lineage synchronization started in background",
	})
}
