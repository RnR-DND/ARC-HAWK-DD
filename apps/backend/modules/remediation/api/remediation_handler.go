package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/arc-platform/backend/modules/remediation/service"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
)

// RemediationHandler handles remediation API requests
type RemediationHandler struct {
	service *service.RemediationService
}

// NewRemediationHandler creates a new remediation handler
func NewRemediationHandler(svc *service.RemediationService) *RemediationHandler {
	return &RemediationHandler{
		service: svc,
	}
}

// ExecuteRemediationRequest represents a remediation execution request
type ExecuteRemediationRequest struct {
	FindingIDs []string `json:"finding_ids" binding:"required"`
	ActionType string   `json:"action_type" binding:"required,oneof=MASK DELETE ENCRYPT"`
}

// ExecuteRemediationResponse represents a remediation execution response
type ExecuteRemediationResponse struct {
	ActionIDs []string `json:"action_ids"`
	Success   int      `json:"success"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
}

// ExecuteRemediation executes remediation for multiple findings
func (h *RemediationHandler) ExecuteRemediation(c *gin.Context) {
	var req ExecuteRemediationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, interfaces.NewErrorResponse(interfaces.ErrCodeBadRequest, "Invalid request format", err.Error()))
		return
	}

	// Extract user ID from JWT context (prevents IDOR - never trust body-supplied user_id)
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	var actionIDs []string
	var errs []string
	success := 0
	failed := 0

	for _, findingID := range req.FindingIDs {
		actionID, err := h.service.ExecuteRemediation(c.Request.Context(), findingID, req.ActionType, userIDStr)
		if err != nil {
			// If the feature flag is off, fail the whole batch fast with 503 so the
			// UI can surface a clear "Remediation disabled" banner rather than a per-
			// finding error cascade.
			if errors.Is(err, service.ErrRemediationDisabled) {
				c.JSON(http.StatusServiceUnavailable, interfaces.NewErrorResponse(
					"remediation_disabled",
					"Remediation is disabled on this deployment",
					"Set REMEDIATION_ENABLED=true only after connector implementations are verified; stubs will not mutate source data.",
				))
				return
			}
			errs = append(errs, fmt.Sprintf("Finding %s: %s", findingID, err.Error()))
			failed++
		} else {
			actionIDs = append(actionIDs, actionID)
			success++
		}
	}

	c.JSON(http.StatusOK, ExecuteRemediationResponse{
		ActionIDs: actionIDs,
		Success:   success,
		Failed:    failed,
		Errors:    errs,
	})
}

// RollbackRemediation rolls back a remediation action
func (h *RemediationHandler) RollbackRemediation(c *gin.Context) {
	actionID := c.Param("actionId")

	if err := h.service.RollbackRemediation(c.Request.Context(), actionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Remediation rolled back successfully",
		"action_id": actionID,
	})
}

// GeneratePreview generates a remediation preview
func (h *RemediationHandler) GeneratePreview(c *gin.Context) {
	var req struct {
		FindingIDs []string `json:"finding_ids"`
		ActionType string   `json:"action_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, interfaces.NewErrorResponse(interfaces.ErrCodeBadRequest, "Invalid request format", err.Error()))
		return
	}

	preview, err := h.service.GenerateRemediationPreview(c.Request.Context(), req.FindingIDs, req.ActionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to generate remediation preview", err.Error()))
		return
	}

	c.JSON(http.StatusOK, preview)
}

// GetRemediationAction retrieves a single remediation action
func (h *RemediationHandler) GetRemediationAction(c *gin.Context) {
	actionID := c.Param("id")

	action, err := h.service.GetRemediationAction(c.Request.Context(), actionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to retrieve remediation action", err.Error()))
		return
	}

	c.JSON(http.StatusOK, action)
}

// GetRemediationActions retrieves remediation actions for a finding
func (h *RemediationHandler) GetRemediationActions(c *gin.Context) {
	findingID := c.Param("findingId")

	actions, err := h.service.GetRemediationActions(c.Request.Context(), findingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to retrieve remediation actions", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"finding_id": findingID,
		"actions":    actions,
	})
}

// GetRemediationHistory retrieves remediation history for an asset
func (h *RemediationHandler) GetRemediationHistory(c *gin.Context) {
	assetID := c.Param("assetId")

	history, err := h.service.GetRemediationHistory(c.Request.Context(), assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to retrieve remediation history", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"asset_id": assetID,
		"history":  history,
	})
}

// GetPIIPreview returns masked preview of PII before remediation
func (h *RemediationHandler) GetPIIPreview(c *gin.Context) {
	findingID := c.Param("findingId")

	preview, err := h.service.GetPIIPreview(c.Request.Context(), findingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to generate PII preview", err.Error()))
		return
	}

	c.JSON(http.StatusOK, preview)
}
