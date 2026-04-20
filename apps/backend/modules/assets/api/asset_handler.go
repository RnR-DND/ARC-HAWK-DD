package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/assets/service"
	sharedapi "github.com/arc-platform/backend/modules/shared/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AssetHandler handles asset-related requests
type AssetHandler struct {
	service *service.AssetService
}

// NewAssetHandler creates a new asset handler
func NewAssetHandler(service *service.AssetService) *AssetHandler {
	return &AssetHandler{service: service}
}

// GetAsset handles GET /api/v1/assets/:id
func (h *AssetHandler) GetAsset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		sharedapi.BadRequest(c, "Invalid asset ID")
		return
	}

	asset, err := h.service.GetAsset(c.Request.Context(), id)
	if err != nil {
		sharedapi.NotFound(c, "Asset not found")
		return
	}

	sharedapi.Success(c, asset)
}

// ListAssets handles GET /api/v1/assets
func (h *AssetHandler) ListAssets(c *gin.Context) {
	assets, err := h.service.ListAssets(c.Request.Context(), 100, 0)
	if err != nil {
		sharedapi.InternalServerError(c, "Failed to list assets")
		return
	}

	sharedapi.Success(c, assets)
}

// DeleteAsset handles DELETE /api/v1/assets/:id
func (h *AssetHandler) DeleteAsset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		sharedapi.BadRequest(c, "Invalid asset ID")
		return
	}

	if err := h.service.DeleteAsset(c.Request.Context(), id); err != nil {
		sharedapi.InternalServerError(c, "Failed to delete asset")
		return
	}

	c.JSON(200, gin.H{"status": "deleted"})
}

// BulkTagAssets handles POST /api/v1/assets/bulk-tag
// Accepts a JSON body with asset_ids, tags, mode, and manual_override.
// Returns 202 Accepted immediately with a job_id; tagging runs in the background.
func (h *AssetHandler) BulkTagAssets(c *gin.Context) {
	var req service.BulkTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	actor := c.GetString("user_id") // set by auth middleware
	if actor == "" {
		actor = "api"
	}

	result, err := h.service.BulkTagAssets(c.Request.Context(), &req, actor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, result)
}
