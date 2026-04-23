package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/remediation/service"
	"github.com/gin-gonic/gin"
)

// EscalationHandler serves the escalation preview and run endpoints.
type EscalationHandler struct {
	svc *service.EscalationService
}

// NewEscalationHandler creates a new EscalationHandler.
func NewEscalationHandler(svc *service.EscalationService) *EscalationHandler {
	return &EscalationHandler{svc: svc}
}

// Preview godoc
// @Summary Preview escalation candidates
// @Description Returns the list of findings that would be escalated (dry-run, no mutations)
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/escalation/preview [get]
func (h *EscalationHandler) Preview(c *gin.Context) {
	candidates, err := h.svc.PreviewEscalation(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"dry_run":    true,
		"count":      len(candidates),
		"candidates": candidates,
	})
}

// Run godoc
// @Summary Run escalation pass
// @Description Executes the escalation pass and returns the count of escalated findings
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param body body object false "Optional escalation run parameters"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/escalation/run [post]
func (h *EscalationHandler) Run(c *gin.Context) {
	count, err := h.svc.RunEscalationPass(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"escalated_count": count,
		"message":         "Escalation pass completed",
	})
}
