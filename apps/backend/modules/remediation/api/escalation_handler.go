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

// Preview handles GET /remediation/escalation/preview.
// Returns the list of findings that WOULD be escalated (dry-run, no mutations).
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

// Run handles POST /remediation/escalation/run.
// Executes the escalation pass and returns the count of escalated findings.
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
