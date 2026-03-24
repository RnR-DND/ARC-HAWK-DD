package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/arc-platform/backend/modules/compliance/service"
	"github.com/gin-gonic/gin"
)

// RetentionHandler handles retention policy API endpoints
type RetentionHandler struct {
	service *service.RetentionService
}

// NewRetentionHandler creates a new retention handler
func NewRetentionHandler(service *service.RetentionService) *RetentionHandler {
	return &RetentionHandler{service: service}
}

// SetRetentionPolicy sets the retention policy for an asset
// POST /api/v1/retention/policies
func (h *RetentionHandler) SetRetentionPolicy(c *gin.Context) {
	var req struct {
		AssetID     string `json:"asset_id" binding:"required"`
		PolicyDays  int    `json:"policy_days" binding:"required,min=1"`
		PolicyName  string `json:"policy_name" binding:"required"`
		PolicyBasis string `json:"policy_basis"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SetRetentionPolicy(c.Request.Context(), req.AssetID, req.PolicyDays, req.PolicyName, req.PolicyBasis)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Retention policy set successfully",
	})
}

// GetRetentionPolicy gets the retention policy for an asset
// GET /api/v1/retention/policies/:assetId
func (h *RetentionHandler) GetRetentionPolicy(c *gin.Context) {
	assetID := c.Param("assetId")
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "asset_id is required"})
		return
	}

	policy, err := h.service.GetRetentionPolicy(c.Request.Context(), assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// GetRetentionViolations returns findings that exceed retention policy
// GET /api/v1/retention/violations
func (h *RetentionHandler) GetRetentionViolations(c *gin.Context) {
	violations, err := h.service.GetRetentionViolations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"violations": violations,
		"total":      len(violations),
	})
}

// GetRetentionTimeline gets the retention timeline for an asset
// GET /api/v1/retention/timeline/:assetId
func (h *RetentionHandler) GetRetentionTimeline(c *gin.Context) {
	assetID := c.Param("assetId")
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "asset_id is required"})
		return
	}

	timeline, err := h.service.GetRetentionTimeline(c.Request.Context(), assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"timeline": timeline,
		"total":    len(timeline),
	})
}

// AuditHandler handles audit log API endpoints
type AuditHandler struct {
	service *service.AuditService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(service *service.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

// ListAuditLogs lists audit logs with optional filters
// GET /api/v1/audit/logs
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	filters := service.AuditFilters{
		UserID:       c.Query("user_id"),
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		ResourceID:   c.Query("resource_id"),
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters.StartTime = &t
		}
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters.EndTime = &t
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}

	logs, err := h.service.ListAuditLogs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
	})
}

// GetUserActivity gets activity summary for a user
// GET /api/v1/audit/user/:userId
func (h *AuditHandler) GetUserActivity(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	activity, err := h.service.GetUserActivity(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, activity)
}

// GetResourceHistory gets audit history for a specific resource
// GET /api/v1/audit/resource/:resourceType/:resourceId
func (h *AuditHandler) GetResourceHistory(c *gin.Context) {
	resourceType := c.Param("resourceType")
	resourceID := c.Param("resourceId")

	if resourceType == "" || resourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_type and resource_id are required"})
		return
	}

	logs, err := h.service.GetResourceHistory(c.Request.Context(), resourceType, resourceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
	})
}

// GetRecentActivity gets recent activity across all users
// GET /api/v1/audit/recent
func (h *AuditHandler) GetRecentActivity(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	logs, err := h.service.GetRecentActivity(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
	})
}
