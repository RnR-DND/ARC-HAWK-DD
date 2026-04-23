package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/memory/service"
	"github.com/gin-gonic/gin"
)

type MemoryHandler struct {
	svc *service.MemoryService
}

func NewMemoryHandler(svc *service.MemoryService) *MemoryHandler {
	return &MemoryHandler{svc: svc}
}

// GetStatus godoc
// @Summary Get memory service status
// @Tags memory
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /memory/status [get]
func (h *MemoryHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled":  h.svc.Enabled(),
		"provider": "supermemory.ai",
	})
}

type searchBody struct {
	Q     string `json:"q" binding:"required"`
	Limit int    `json:"limit"`
}

// Search godoc
// @Summary Semantic search over scan history
// @Description Hybrid memory + RAG search via Supermemory
// @Tags memory
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param body body object true "{query, limit}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /memory/search [post]
func (h *MemoryHandler) Search(c *gin.Context) {
	var body searchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.svc.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "memory backend disabled",
			"details": "set SUPERMEMORY_ENABLED=true and SUPERMEMORY_API_KEY in backend env",
		})
		return
	}
	resp, err := h.svc.Search(c.Request.Context(), service.SearchQuery{
		Q:     body.Q,
		Limit: body.Limit,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
