package api

import (
	"github.com/arc-platform/backend/modules/assets/service"
	"github.com/arc-platform/backend/modules/shared/api"
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
		api.BadRequest(c, "Invalid asset ID")
		return
	}

	asset, err := h.service.GetAsset(c.Request.Context(), id)
	if err != nil {
		api.NotFound(c, "Asset not found")
		return
	}

	api.Success(c, asset)
}

// ListAssets handles GET /api/v1/assets
func (h *AssetHandler) ListAssets(c *gin.Context) {
	assets, err := h.service.ListAssets(c.Request.Context(), 100, 0)
	if err != nil {
		api.InternalServerError(c, "Failed to list assets")
		return
	}

	api.Success(c, assets)
}
