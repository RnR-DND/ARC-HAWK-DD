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

// RecordConsent godoc
// @Summary Record a new consent
// @Description Records a new consent entry for a data subject
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param body body service.ConsentRequest true "Consent request payload"
// @Success 201 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/consent/records [post]
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

// ListConsentRecords godoc
// @Summary List consent records
// @Description Returns consent records with optional filters
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param asset_id query string false "Filter by asset ID"
// @Param pii_type query string false "Filter by PII type"
// @Param status query string false "Filter by consent status"
// @Param limit query int false "Maximum records to return"
// @Param offset query int false "Number of records to skip"
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/consent/records [get]
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

// WithdrawConsent godoc
// @Summary Withdraw an existing consent
// @Description Withdraws a previously recorded consent by ID
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param id path string true "Consent ID"
// @Param body body service.ConsentWithdrawalRequest true "Withdrawal request payload"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/consent/withdraw/{id} [post]
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

// GetConsentStatus godoc
// @Summary Get consent status for an asset and PII type
// @Description Returns the consent status for a specific asset and PII type combination
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param assetId path string true "Asset ID"
// @Param piiType path string true "PII type"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/consent/status/{assetId}/{piiType} [get]
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

// GetConsentViolations godoc
// @Summary Get consent violations
// @Description Returns assets that have consent violations
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/consent/violations [get]
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
