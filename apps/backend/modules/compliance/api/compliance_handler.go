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

// GetComplianceOverview godoc
// @Summary Get DPDPA compliance overview
// @Description Returns the DPDPA compliance dashboard with summary metrics
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/overview [get]
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

// GetCriticalAssets godoc
// @Summary Get critical assets with PII exposure
// @Description Returns assets with critical PII exposure
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/critical [get]
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

// GetConsentViolations godoc
// @Summary Get consent violations
// @Description Returns assets violating consent rules
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/violations [get]
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
