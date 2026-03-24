package api

import (
	"net/http"
	"strconv"

	"github.com/arc-platform/backend/modules/compliance/service"
	"github.com/gin-gonic/gin"
)

// ConsentHandler handles consent management API endpoints
type ConsentHandler struct {
	service *service.ConsentService
}

// NewConsentHandler creates a new consent handler
func NewConsentHandler(service *service.ConsentService) *ConsentHandler {
	return &ConsentHandler{service: service}
}

// RecordConsent records a new consent
// POST /api/v1/consent/records
func (h *ConsentHandler) RecordConsent(c *gin.Context) {
	var req service.ConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record, err := h.service.RecordConsent(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// ListConsentRecords lists consent records with optional filters
// GET /api/v1/consent/records
func (h *ConsentHandler) ListConsentRecords(c *gin.Context) {
	filters := service.ConsentFilters{
		AssetID: c.Query("asset_id"),
		PIIType: c.Query("pii_type"),
		Status:  service.ConsentStatus(c.Query("status")),
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}

	records, err := h.service.ListConsentRecords(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"total":   len(records),
	})
}

// WithdrawConsent withdraws an existing consent
// POST /api/v1/consent/withdraw/:id
func (h *ConsentHandler) WithdrawConsent(c *gin.Context) {
	consentID := c.Param("id")
	if consentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "consent_id is required"})
		return
	}

	var req service.ConsentWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.WithdrawConsent(c.Request.Context(), consentID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "withdrawn",
		"message":    "Consent withdrawn successfully",
		"consent_id": consentID,
	})
}

// GetConsentStatus gets the consent status for a specific asset and PII type
// GET /api/v1/consent/status/:assetId/:piiType
func (h *ConsentHandler) GetConsentStatus(c *gin.Context) {
	assetID := c.Param("assetId")
	piiType := c.Param("piiType")

	if assetID == "" || piiType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "asset_id and pii_type are required"})
		return
	}

	record, err := h.service.GetConsentStatus(c.Request.Context(), assetID, piiType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if record == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "MISSING",
			"message": "No consent record found",
		})
		return
	}

	c.JSON(http.StatusOK, record)
}

// GetConsentViolations returns assets with consent violations
// GET /api/v1/consent/violations
func (h *ConsentHandler) GetConsentViolations(c *gin.Context) {
	violations, err := h.service.GetConsentViolations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"violations": violations,
		"total":      len(violations),
	})
}
