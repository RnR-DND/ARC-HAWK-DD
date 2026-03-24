package api

import (
	"net/http"

	"github.com/arc-platform/backend/modules/compliance/service"
	"github.com/gin-gonic/gin"
)

// ComplianceHandler handles DPDPA compliance endpoints
type ComplianceHandler struct {
	service *service.ComplianceService
}

// NewComplianceHandler creates a new compliance handler
func NewComplianceHandler(service *service.ComplianceService) *ComplianceHandler {
	return &ComplianceHandler{
		service: service,
	}
}

// GetComplianceOverview returns the DPDPA compliance dashboard
// GET /api/v1/compliance/overview
func (h *ComplianceHandler) GetComplianceOverview(c *gin.Context) {
	overview, err := h.service.GetComplianceOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, overview)
}

// GetCriticalAssets returns assets with critical PII exposure
// GET /api/v1/compliance/critical
func (h *ComplianceHandler) GetCriticalAssets(c *gin.Context) {
	assets, err := h.service.GetCriticalAssets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"assets": assets,
	})
}

// GetConsentViolations returns assets violating consent rules
// GET /api/v1/compliance/violations
func (h *ComplianceHandler) GetConsentViolations(c *gin.Context) {
	violations, err := h.service.GetConsentViolations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"violations": violations,
	})
}
