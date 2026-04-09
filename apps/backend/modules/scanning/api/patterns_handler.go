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

// ListPatterns GET /api/v1/patterns
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

// CreatePattern POST /api/v1/patterns
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
			"name":      p.Name,
			"tenant_id": tenantID.String(),
			"created_by": createdByStr,
		})
	}

	c.JSON(http.StatusCreated, gin.H{"data": p})
}

// UpdatePattern PUT /api/v1/patterns/:id
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

// DeletePattern DELETE /api/v1/patterns/:id
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

// RecordFalsePositive POST /api/v1/patterns/:id/false-positive
// Increments the false-positive counter for the pattern and triggers auto-deactivation
// if the resulting false_positive_rate exceeds 0.30.
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

// GetPatternStats GET /api/v1/patterns/:id/stats
// Returns match-frequency and false-positive statistics for a single pattern.
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

// TestPattern POST /api/v1/patterns/:id/test
// Runs the pattern's compiled regex against caller-supplied test cases and returns
// pass/fail per case plus an overall pass rate.
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
