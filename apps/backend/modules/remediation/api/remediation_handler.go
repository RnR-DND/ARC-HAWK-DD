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

// ExecuteRemediation godoc
// @Summary Execute remediation for findings
// @Description Executes remediation actions (MASK, DELETE, or ENCRYPT) for one or more findings
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param body body ExecuteRemediationRequest true "Remediation execution request"
// @Success 200 {object} ExecuteRemediationResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Failure 503 {object} gin.H
// @Router /api/v1/remediation [post]
func (h *RemediationHandler) ExecuteRemediation(c *gin.Context) {
	var req ExecuteRemediationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, interfaces.NewErrorResponse(interfaces.ErrCodeBadRequest, "Invalid request format", nil))
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

// RollbackRemediation godoc
// @Summary Rollback a remediation action
// @Description Rolls back a previously executed remediation action by action ID
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param id path string true "Remediation action ID"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/rollback/{id} [post]
func (h *RemediationHandler) RollbackRemediation(c *gin.Context) {
	actionID := c.Param("actionId")

	if err := h.service.RollbackRemediation(c.Request.Context(), actionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Remediation rolled back successfully",
		"action_id": actionID,
	})
}

// GeneratePreview godoc
// @Summary Generate remediation preview
// @Description Generates a preview of what remediation would do without executing it
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param body body object true "Preview request with finding_ids and action_type"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/preview [post]
func (h *RemediationHandler) GeneratePreview(c *gin.Context) {
	var req struct {
		FindingIDs []string `json:"finding_ids"`
		ActionType string   `json:"action_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, interfaces.NewErrorResponse(interfaces.ErrCodeBadRequest, "Invalid request format", nil))
		return
	}

	preview, err := h.service.GenerateRemediationPreview(c.Request.Context(), req.FindingIDs, req.ActionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to generate remediation preview", nil))
		return
	}

	c.JSON(http.StatusOK, preview)
}

// GetRemediationAction godoc
// @Summary Get a single remediation action
// @Description Retrieves details of a specific remediation action by ID
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param id path string true "Remediation action ID"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/{id} [get]
func (h *RemediationHandler) GetRemediationAction(c *gin.Context) {
	actionID := c.Param("id")

	action, err := h.service.GetRemediationAction(c.Request.Context(), actionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to retrieve remediation action", nil))
		return
	}

	c.JSON(http.StatusOK, action)
}

// GetRemediationActions godoc
// @Summary Get remediation actions for a finding
// @Description Returns all remediation actions associated with a specific finding
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param findingId path string true "Finding ID"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/actions/{findingId} [get]
func (h *RemediationHandler) GetRemediationActions(c *gin.Context) {
	findingID := c.Param("findingId")

	actions, err := h.service.GetRemediationActions(c.Request.Context(), findingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to retrieve remediation actions", nil))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"finding_id": findingID,
		"actions":    actions,
	})
}

// GetRemediationHistory godoc
// @Summary Get remediation history for an asset
// @Description Returns the full remediation action history for a specific asset
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param assetId path string true "Asset ID"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/history/{assetId} [get]
func (h *RemediationHandler) GetRemediationHistory(c *gin.Context) {
	assetID := c.Param("assetId")

	history, err := h.service.GetRemediationHistory(c.Request.Context(), assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to retrieve remediation history", nil))
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
		c.JSON(http.StatusInternalServerError, interfaces.NewErrorResponse(interfaces.ErrCodeInternalServer, "Failed to generate PII preview", nil))
		return
	}

	c.JSON(http.StatusOK, preview)
}
