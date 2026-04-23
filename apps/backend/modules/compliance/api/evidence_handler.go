package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/arc-platform/backend/modules/compliance/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/audit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EvidenceHandler exposes the DPDP evidence package endpoints.
type EvidenceHandler struct {
	svc    *service.EvidencePackageService
	logger *audit.LedgerLogger
}

// NewEvidenceHandler creates a new EvidenceHandler.
func NewEvidenceHandler(svc *service.EvidencePackageService, logger *audit.LedgerLogger) *EvidenceHandler {
	return &EvidenceHandler{svc: svc, logger: logger}
}

// GeneratePackage godoc
// @Summary Generate DPDP evidence package
// @Description Generates a ZIP containing all DPDP evidence and streams it to the client
// @Tags compliance
// @Accept json
// @Produce application/zip
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/evidence-package [post]
func (h *EvidenceHandler) GeneratePackage(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	actorIDStr, _ := c.Get("user_id")
	actorID, _ := uuid.Parse(fmt.Sprintf("%v", actorIDStr))
	actorEmailVal, _ := c.Get("user_email")
	actorEmail := fmt.Sprintf("%v", actorEmailVal)

	pkg, err := h.svc.Generate(c.Request.Context(), tenantID, actorID, actorEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+pkg.Filename+`"`)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Length", fmt.Sprintf("%d", len(pkg.ZipBytes)))
	c.Data(http.StatusOK, "application/zip", pkg.ZipBytes)
}

// GetAuditTrail godoc
// @Summary Get compliance audit trail
// @Description Returns paginated audit ledger rows for the tenant
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param event_type query string false "Filter by event type (repeatable)"
// @Param from query string false "Start time in RFC3339 format"
// @Param to query string false "End time in RFC3339 format"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/audit-trail [get]
func (h *EvidenceHandler) GetAuditTrail(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)

	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			now = t
		}
	}

	var eventTypes []string
	if et := c.QueryArray("event_type"); len(et) > 0 {
		eventTypes = et
	}

	results, err := h.logger.Query(c.Request.Context(), tenantID, eventTypes, from, now, 500)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_trail": results,
		"count":       len(results),
	})
}
