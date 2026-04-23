package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/remediation/service"
	"github.com/gin-gonic/gin"
)

// RemediationHistoryHandler handles remediation history requests
type RemediationHistoryHandler struct {
	service *service.RemediationService
}

// NewRemediationHistoryHandler creates a new remediation history handler
func NewRemediationHistoryHandler(svc *service.RemediationService) *RemediationHistoryHandler {
	return &RemediationHistoryHandler{
		service: svc,
	}
}

// GetHistory godoc
// @Summary Get remediation history
// @Description Returns the audit trail of all remediation actions
// @Tags remediation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param limit query int false "Maximum records to return (default 50)"
// @Param offset query int false "Number of records to skip (default 0)"
// @Param action query string false "Filter by action type"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/remediation/history [get]
func (h *RemediationHistoryHandler) GetHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	actionFilter := c.Query("action") // Optional filter

	actions, total, err := h.service.GetAllRemediationActions(c.Request.Context(), limit, offset, actionFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch remediation history: " + err.Error()})
		return
	}

	// Map to response format — include enriched asset/finding context
	history := make([]map[string]interface{}, 0, len(actions))
	for _, action := range actions {
		record := map[string]interface{}{
			"id":             action.ID,
			"finding_id":     action.FindingID,
			"action":         action.ActionType,
			"executed_by":    action.ExecutedBy,
			"executed_at":    action.ExecutedAt,
			"status":         action.Status,
			"original_value": action.OriginalValue,
			// Enriched fields from JOIN with assets + findings
			"asset_name": action.AssetName,
			"asset_path": action.AssetPath,
			"pii_type":   action.PIIType,
			"risk_level": action.RiskLevel,
		}
		history = append(history, record)
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}
