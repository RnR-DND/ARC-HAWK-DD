package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/discovery/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SnapshotHandler serves snapshot CRUD + manual trigger endpoints.
type SnapshotHandler struct {
	snapshotService *service.SnapshotService
	repo            *service.Repo
}

// NewSnapshotHandler creates a new snapshot handler.
func NewSnapshotHandler(snapshotService *service.SnapshotService, repo *service.Repo) *SnapshotHandler {
	return &SnapshotHandler{snapshotService: snapshotService, repo: repo}
}

// ListSnapshots returns recent snapshots for the tenant in ctx.
// GET /api/discovery/snapshots?limit=50&offset=0
// ListSnapshots godoc
// @Summary List point-in-time snapshots
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /discovery/snapshots [get]
func (h *SnapshotHandler) ListSnapshots(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	snaps, err := h.repo.ListSnapshots(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items": snaps,
		"count": len(snaps),
	})
}

// GetSnapshot returns a single snapshot with its facts.
// GET /api/discovery/snapshots/:id
// GetSnapshot godoc
// @Summary Get a specific snapshot
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Snapshot UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /discovery/snapshots/{id} [get]
func (h *SnapshotHandler) GetSnapshot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
		return
	}

	snap, err := h.repo.GetSnapshot(c.Request.Context(), id)
	if err == service.ErrSnapshotNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	facts, _ := h.repo.ListFactsForSnapshot(c.Request.Context(), id)
	c.JSON(http.StatusOK, gin.H{
		"snapshot": snap,
		"facts":    facts,
	})
}

// TriggerSnapshot kicks off a manual snapshot for the tenant in ctx.
// POST /api/discovery/snapshots/trigger
// TriggerSnapshot godoc
// @Summary Trigger a new data snapshot
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 202 {object} gin.H
// @Security BearerAuth
// @Router /discovery/snapshots/trigger [post]
func (h *SnapshotHandler) TriggerSnapshot(c *gin.Context) {
	// Get triggering user ID from auth middleware (optional — cron has no user).
	var triggeredBy *uuid.UUID
	if userID, ok := c.Get("user_id"); ok {
		if uid, ok := userID.(uuid.UUID); ok {
			triggeredBy = &uid
		}
	}

	// Verify tenant context exists before async-ing.
	if _, err := persistence.EnsureTenantID(c.Request.Context()); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context required"})
		return
	}

	snap, err := h.snapshotService.TakeSnapshot(c.Request.Context(), domain.TriggerManual, triggeredBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":       err.Error(),
			"snapshot_id": snap.ID,
		})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"snapshot_id": snap.ID,
		"status":      snap.Status,
		"asset_count": snap.AssetCount,
		"duration_ms": snap.DurationMS,
	})
}
