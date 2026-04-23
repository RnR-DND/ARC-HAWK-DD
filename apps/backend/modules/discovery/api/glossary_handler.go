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
// GetGlossary godoc
// @Summary Get data glossary and term registry
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /discovery/glossary [get]
func (h *GlossaryHandler) GetGlossary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"items":   []interface{}{},
		"count":   0,
		"version": "v1-stub",
		"note":    "glossary CRUD lands in v1.5",
	})
}
