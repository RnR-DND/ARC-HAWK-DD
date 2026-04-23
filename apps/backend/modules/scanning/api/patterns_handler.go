package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PatternsHandler handles CRUD for custom PII patterns.
type PatternsHandler struct {
	svc         *service.PatternsService
	auditLogger interfaces.AuditLogger
}

// NewPatternsHandler creates a PatternsHandler.
func NewPatternsHandler(svc *service.PatternsService, auditLogger interfaces.AuditLogger) *PatternsHandler {
	return &PatternsHandler{svc: svc, auditLogger: auditLogger}
}

func tenantFromCtx(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("tenant_id"); ok {
		switch t := v.(type) {
		case uuid.UUID:
			return t
		case string:
			if id, err := uuid.Parse(t); err == nil {
				return id
			}
		}
	}
	return persistence.DevSystemTenantID
}

// ListPatterns godoc
// @Summary List custom PII patterns
// @Tags patterns
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Success 200 {object} gin.H "data: []CustomPattern"
// @Security BearerAuth
// @Router /patterns [get]
func (h *PatternsHandler) ListPatterns(c *gin.Context) {
	tenantID := tenantFromCtx(c)
	patterns, err := h.svc.ListPatterns(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if patterns == nil {
		patterns = []*service.CustomPattern{}
	}
	c.JSON(http.StatusOK, gin.H{"data": patterns})
}

// CreatePattern godoc
// @Summary Create a custom PII pattern
// @Tags patterns
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param body body service.CustomPattern true "Pattern definition (name, regex, pii_type required)"
// @Success 201 {object} gin.H
// @Failure 400 {object} gin.H
// @Security BearerAuth
// @Router /patterns [post]
func (h *PatternsHandler) CreatePattern(c *gin.Context) {
	var body service.CustomPattern
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if body.Name == "" || body.Regex == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and regex are required"})
		return
	}
	if body.DisplayName == "" {
		body.DisplayName = body.Name
	}

	tenantID := tenantFromCtx(c)
	createdBy, _ := c.Get("user_email")
	createdByStr, _ := createdBy.(string)

	p, err := h.svc.CreatePattern(c.Request.Context(), tenantID, createdByStr, &body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Record(c.Request.Context(), "PATTERN_CREATED", "custom_pattern", p.ID.String(), map[string]interface{}{
			"name":       p.Name,
			"tenant_id":  tenantID.String(),
			"created_by": createdByStr,
		})
	}

	c.JSON(http.StatusCreated, gin.H{"data": p})
}

// UpdatePattern godoc
// @Summary Update a custom pattern
// @Tags patterns
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Pattern UUID"
// @Success 200 {object} gin.H
// @Failure 404 {object} gin.H
// @Security BearerAuth
// @Router /patterns/{id} [put]
func (h *PatternsHandler) UpdatePattern(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern id"})
		return
	}
	var body service.CustomPattern
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	tenantID := tenantFromCtx(c)
	p, err := h.svc.UpdatePattern(c.Request.Context(), tenantID, id, &body)
	if err != nil {
		if err.Error() == "pattern not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "pattern not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Record(c.Request.Context(), "PATTERN_UPDATED", "custom_pattern", id.String(), map[string]interface{}{
			"name":      p.Name,
			"tenant_id": tenantID.String(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": p})
}

// DeletePattern godoc
// @Summary Delete a custom pattern
// @Tags patterns
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Pattern UUID"
// @Success 200 {object} gin.H
// @Failure 404 {object} gin.H
// @Security BearerAuth
// @Router /patterns/{id} [delete]
func (h *PatternsHandler) DeletePattern(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern id"})
		return
	}
	tenantID := tenantFromCtx(c)
	if err := h.svc.DeletePattern(c.Request.Context(), tenantID, id); err != nil {
		if err.Error() == "pattern not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "pattern not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Record(c.Request.Context(), "PATTERN_DELETED", "custom_pattern", id.String(), map[string]interface{}{
			"tenant_id": tenantID.String(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "pattern deleted"})
}

// RecordFalsePositive godoc
// @Summary Record a false positive hit for a pattern
// @Description Increments false-positive counter; auto-deactivates if rate > 30%%
// @Tags patterns
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Pattern UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /patterns/{id}/false-positive [post]
func (h *PatternsHandler) RecordFalsePositive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern id"})
		return
	}
	tenantID := tenantFromCtx(c)

	p, err := h.svc.RecordFalsePositive(c.Request.Context(), tenantID, id)
	if err != nil {
		if err.Error() == "pattern not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "pattern not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Record(c.Request.Context(), "PATTERN_FALSE_POSITIVE", "custom_pattern", id.String(), map[string]interface{}{
			"tenant_id":           tenantID.String(),
			"false_positive_rate": p.FalsePositiveRate,
			"auto_deactivated":    p.AutoDeactivated,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": p})
}

// GetPatternStats godoc
// @Summary Get match and false-positive stats for a pattern
// @Tags patterns
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Pattern UUID"
// @Success 200 {object} gin.H
// @Security BearerAuth
// @Router /patterns/{id}/stats [get]
func (h *PatternsHandler) GetPatternStats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern id"})
		return
	}
	tenantID := tenantFromCtx(c)

	stats, err := h.svc.GetPatternStats(c.Request.Context(), tenantID, id)
	if err != nil {
		if err.Error() == "pattern not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "pattern not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// TestPattern godoc
// @Summary Test a pattern against sample values
// @Tags patterns
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Param id path string true "Pattern UUID"
// @Param body body object true "{test_cases: [{value, should_match}]}"
// @Success 200 {object} gin.H "pass_rate, results per case"
// @Security BearerAuth
// @Router /patterns/{id}/test [post]
func (h *PatternsHandler) TestPattern(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern id"})
		return
	}
	tenantID := tenantFromCtx(c)

	var body struct {
		TestCases []service.TestCase `json:"test_cases"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if len(body.TestCases) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "test_cases must not be empty"})
		return
	}

	result, err := h.svc.TestPattern(c.Request.Context(), tenantID, id, body.TestCases)
	if err != nil {
		if err.Error() == "pattern not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "pattern not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}
