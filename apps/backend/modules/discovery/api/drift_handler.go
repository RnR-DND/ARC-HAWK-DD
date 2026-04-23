package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/discovery/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DriftHandler serves drift event endpoints.
type DriftHandler struct {
	drift *service.DriftDetectionService
	repo  *service.Repo
}

// NewDriftHandler creates a new drift handler.
func NewDriftHandler(drift *service.DriftDetectionService, repo *service.Repo) *DriftHandler {
	return &DriftHandler{drift: drift, repo: repo}
}

// GetDriftSince returns drift events for a specific snapshot ID.
// GET /api/discovery/drift/since/:snapshotId?limit=100
// GetDriftSince godoc
// @Summary Get drift events since a snapshot
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param snapshotId path string true "Snapshot UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /discovery/drift/since/{snapshotId} [get]
func (h *DriftHandler) GetDriftSince(c *gin.Context) {
	snapshotID, err := uuid.Parse(c.Param("snapshotId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot id"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	events, err := h.repo.ListDriftSinceSnapshot(c.Request.Context(), snapshotID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"snapshot_id": snapshotID,
		"events":      events,
		"count":       len(events),
	})
}

// GetDriftTimeline returns drift events from the most recent snapshot.
// GET /api/discovery/drift/timeline?limit=100
// GetDriftTimeline godoc
// @Summary Get full drift timeline
// @Tags discovery
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /discovery/drift/timeline [get]
func (h *DriftHandler) GetDriftTimeline(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	// Find the most recent snapshot to anchor the timeline.
	snaps, err := h.repo.ListSnapshots(c.Request.Context(), 1, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(snaps) == 0 {
		c.JSON(http.StatusOK, gin.H{"events": []interface{}{}, "count": 0})
		return
	}

	events, err := h.repo.ListDriftSinceSnapshot(c.Request.Context(), snaps[0].ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"snapshot_id": snaps[0].ID,
		"events":      events,
		"count":       len(events),
	})
}
