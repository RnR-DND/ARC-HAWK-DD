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

// GetStatus reports whether the memory backend is wired up.
// GET /api/v1/memory/status
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

// Search forwards /v3/search — hybrid memory + RAG.
// POST /api/v1/memory/search
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
