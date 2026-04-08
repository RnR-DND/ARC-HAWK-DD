package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/discovery/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InventoryHandler serves discovery_inventory endpoints.
type InventoryHandler struct {
	repo *service.Repo
}

// NewInventoryHandler creates a new inventory handler.
func NewInventoryHandler(repo *service.Repo) *InventoryHandler {
	return &InventoryHandler{repo: repo}
}

// ListInventory returns paginated inventory rows for the tenant in ctx.
// GET /api/discovery/inventory?classification=&source_id=&limit=100&offset=0
func (h *InventoryHandler) ListInventory(c *gin.Context) {
	classification := c.Query("classification")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var sourceID *uuid.UUID
	if s := c.Query("source_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			sourceID = &id
		}
	}

	rows, err := h.repo.ListInventory(c.Request.Context(), classification, sourceID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items": rows,
		"count": len(rows),
		"limit": limit,
		"offset": offset,
	})
}

// GetAssetInventory returns all inventory rows for one asset.
// GET /api/discovery/inventory/:assetId
func (h *InventoryHandler) GetAssetInventory(c *gin.Context) {
	assetID, err := uuid.Parse(c.Param("assetId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assetId"})
		return
	}

	// Reuse ListInventory with no classification filter, capped to this asset.
	// We don't yet have a per-asset list method on the repo; emulate via filter.
	rows, err := h.repo.ListInventory(c.Request.Context(), "", nil, 1000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var matched []interface{}
	for _, r := range rows {
		if r.AssetID == assetID {
			matched = append(matched, r)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"asset_id": assetID,
		"rows":     matched,
		"count":    len(matched),
	})
}
