package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GlossaryHandler serves the glossary stub. v1 returns an empty list.
// v1.5 will add full CRUD on a discovery_glossary table.
type GlossaryHandler struct{}

// NewGlossaryHandler creates a new glossary handler.
func NewGlossaryHandler() *GlossaryHandler {
	return &GlossaryHandler{}
}

// GetGlossary returns the (empty) glossary list.
// GET /api/discovery/glossary
func (h *GlossaryHandler) GetGlossary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"items":   []interface{}{},
		"count":   0,
		"version": "v1-stub",
		"note":    "glossary CRUD lands in v1.5",
	})
}
