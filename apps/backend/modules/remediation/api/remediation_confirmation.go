package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/remediation/service"
	"github.com/gin-gonic/gin"
)

// RemediationConfirmationHandler handles remediation confirmation workflow
type RemediationConfirmationHandler struct {
	service *service.RemediationService
}

// NewRemediationConfirmationHandler creates a new confirmation handler
func NewRemediationConfirmationHandler(service *service.RemediationService) *RemediationConfirmationHandler {
	return &RemediationConfirmationHandler{
		service: service,
	}
}

// PreviewRequest represents a remediation preview request
type PreviewRequest struct {
	FindingIDs []string `json:"finding_ids" binding:"required"`
	ActionType string   `json:"action_type" binding:"required,oneof=MASK DELETE ENCRYPT"`
}

// PreviewResponse represents a remediation preview response
type PreviewResponse struct {
	RequestID            string            `json:"request_id"`
	FindingIDs           []string          `json:"finding_ids"`
	ActionType           string            `json:"action_type"`
	Impact               RemediationImpact `json:"impact"`
	Findings             []FindingPreview  `json:"findings"`
	RequiresConfirmation bool              `json:"requires_confirmation"`
}

// RemediationImpact represents the impact of remediation
type RemediationImpact struct {
	TotalFindings    int      `json:"total_findings"`
	AffectedAssets   int      `json:"affected_assets"`
	AffectedSystems  int      `json:"affected_systems"`
	PIITypes         []string `json:"pii_types"`
	EstimatedRecords int      `json:"estimated_records"`
}

// FindingPreview represents a finding in the preview
type FindingPreview struct {
	FindingID    string `json:"finding_id"`
	AssetName    string `json:"asset_name"`
	AssetPath    string `json:"asset_path"`
	PIIType      string `json:"pii_type"`
	FieldName    string `json:"field_name"`
	SampleBefore string `json:"sample_before"`
	SampleAfter  string `json:"sample_after"`
}

// ApprovalRequest represents a remediation approval request
type ApprovalRequest struct {
	RequestID string `json:"request_id" binding:"required"`
	Approved  bool   `json:"approved" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	Comment   string `json:"comment"`
}

// PreviewRemediation generates a preview of remediation impact
func (h *RemediationConfirmationHandler) PreviewRemediation(c *gin.Context) {
	var req PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate preview
	preview, err := h.service.GenerateRemediationPreview(c.Request.Context(), req.FindingIDs, req.ActionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preview)
}

// ApproveRemediation approves and executes remediation
func (h *RemediationConfirmationHandler) ApproveRemediation(c *gin.Context) {
	var req ApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.Approved {
		// Reject remediation
		c.JSON(http.StatusOK, gin.H{
			"status":  "rejected",
			"message": "Remediation request rejected by user",
		})
		return
	}

	// Execute remediation
	result, err := h.service.ExecuteRemediationRequest(c.Request.Context(), req.RequestID, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "approved",
		"message": "Remediation executed successfully",
		"result":  result,
	})
}

// RollbackRemediation rolls back a completed remediation
func (h *RemediationConfirmationHandler) RollbackRemediation(c *gin.Context) {
	actionID := c.Param("actionId")
	if actionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action_id is required"})
		return
	}

	if err := h.service.RollbackRemediation(c.Request.Context(), actionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "rolled_back",
		"message":   "Remediation rolled back successfully",
		"action_id": actionID,
	})
}
