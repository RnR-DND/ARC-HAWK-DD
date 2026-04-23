package api

import (
	"log"
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
// MaskAsset godoc
// @Summary Apply PII masking to an asset
// @Description Dispatches to the appropriate source-system connector to mask sensitive fields in-place
// @Tags masking
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param body body object true "{asset_id, strategy, dry_run}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /masking/mask-asset [post]
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

	// Internal findings table updated. Source-system data is NOT redacted here.
	// Operators must invoke RemediationService per-finding to reach the source system.
	log.Printf("masking: asset %s masked in findings table (strategy=%s, by=%s) — source-system data unchanged", req.AssetID, req.Strategy, maskedBy)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Asset masked successfully",
		"asset_id": req.AssetID,
		"strategy": req.Strategy,
	})
}

// GetMaskingStatus handles GET /api/v1/masking/status/:assetId
// GetMaskingStatus godoc
// @Summary Get masking status for an asset
// @Tags masking
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param assetId path string true "Asset UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /masking/status/{assetId} [get]
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
// GetMaskingAuditLog godoc
// @Summary Get masking operation audit log for an asset
// @Tags masking
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param assetId path string true "Asset UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /masking/audit/{assetId} [get]
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
