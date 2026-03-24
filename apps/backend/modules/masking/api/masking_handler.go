package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/masking/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MaskingHandler handles masking-related HTTP requests
type MaskingHandler struct {
	maskingService *service.MaskingService
}

// NewMaskingHandler creates a new masking handler
func NewMaskingHandler(maskingService *service.MaskingService) *MaskingHandler {
	return &MaskingHandler{
		maskingService: maskingService,
	}
}

// MaskAssetRequest represents the request to mask an asset
type MaskAssetRequest struct {
	AssetID  string `json:"asset_id" binding:"required"`
	Strategy string `json:"strategy" binding:"required,oneof=REDACT PARTIAL TOKENIZE"`
	MaskedBy string `json:"masked_by,omitempty"`
}

// MaskAsset handles POST /api/v1/masking/mask-asset
func (h *MaskingHandler) MaskAsset(c *gin.Context) {
	var req MaskAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Parse asset ID
	assetID, err := uuid.Parse(req.AssetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid asset ID format",
		})
		return
	}

	// Default masked_by to "system" if not provided
	maskedBy := req.MaskedBy
	if maskedBy == "" {
		maskedBy = "system"
	}

	// Perform masking
	err = h.maskingService.MaskAsset(
		c.Request.Context(),
		assetID,
		service.MaskingStrategy(req.Strategy),
		maskedBy,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to mask asset",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Asset masked successfully",
		"asset_id": req.AssetID,
		"strategy": req.Strategy,
	})
}

// GetMaskingStatus handles GET /api/v1/masking/status/:assetId
func (h *MaskingHandler) GetMaskingStatus(c *gin.Context) {
	assetIDStr := c.Param("assetId")

	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid asset ID format",
		})
		return
	}

	status, err := h.maskingService.GetMaskingStatus(c.Request.Context(), assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get masking status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetMaskingAuditLog handles GET /api/v1/masking/audit/:assetId
func (h *MaskingHandler) GetMaskingAuditLog(c *gin.Context) {
	assetIDStr := c.Param("assetId")

	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid asset ID format",
		})
		return
	}

	auditLog, err := h.maskingService.GetMaskingAuditLog(c.Request.Context(), assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get audit log",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"asset_id":  assetIDStr,
		"audit_log": auditLog,
	})
}
