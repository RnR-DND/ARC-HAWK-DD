package api

// ConsentRecordsHandler handles the /compliance/consent endpoints backed by the
// consent_records table introduced in migration 000030.
//
// Routes:
//   GET    /compliance/consent          — list consent records (paginated)
//   POST   /compliance/consent          — create a consent record
//   DELETE /compliance/consent/:id      — withdraw consent (sets withdrawal_timestamp, is_active=false)

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ConsentRecordsHandler serves the compliance/consent REST surface.
type ConsentRecordsHandler struct {
	db *sql.DB
}

// NewConsentRecordsHandler creates a new handler backed by the provided *sql.DB.
func NewConsentRecordsHandler(db *sql.DB) *ConsentRecordsHandler {
	return &ConsentRecordsHandler{db: db}
}

// consentRow is the DB row model for consent_records (migration 000030 schema).
type consentRow struct {
	ID                  uuid.UUID  `json:"id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	DataSubjectID       string     `json:"data_subject_id"`
	AssetID             *uuid.UUID `json:"asset_id,omitempty"`
	Purpose             string     `json:"purpose"`
	ConsentGivenAt      time.Time  `json:"consent_given_at"`
	ConsentExpiresAt    *time.Time `json:"consent_expires_at,omitempty"`
	WithdrawalTimestamp *time.Time `json:"withdrawal_timestamp,omitempty"`
	ConsentMechanism    string     `json:"consent_mechanism"`
	IsActive            bool       `json:"is_active"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// createConsentRequest is the POST body for /compliance/consent.
type createConsentRequest struct {
	DataSubjectID    string     `json:"data_subject_id" binding:"required"`
	AssetID          *string    `json:"asset_id,omitempty"`
	Purpose          string     `json:"purpose" binding:"required"`
	ConsentGivenAt   time.Time  `json:"consent_given_at" binding:"required"`
	ConsentExpiresAt *time.Time `json:"consent_expires_at,omitempty"`
	// ConsentMechanism must be one of: explicit, implicit, legitimate_interest
	ConsentMechanism string `json:"consent_mechanism" binding:"required"`
}

// ListConsentRecords handles GET /compliance/consent.
// Query params: limit (default 50, max 200), offset (default 0).
func (h *ConsentRecordsHandler) ListConsentRecords(c *gin.Context) {
	tenantID, ok := extractTenantID(c)
	if !ok {
		return
	}

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	rows, err := h.db.QueryContext(c.Request.Context(), `
		SELECT id, tenant_id, data_subject_id, asset_id,
		       purpose, consent_given_at, consent_expires_at,
		       withdrawal_timestamp, consent_mechanism, is_active, created_at, updated_at
		FROM consent_records
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var records []consentRow
	for rows.Next() {
		var r consentRow
		var assetID sql.NullString
		var consentExpiresAt, withdrawalTS sql.NullTime
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.DataSubjectID, &assetID,
			&r.Purpose, &r.ConsentGivenAt, &consentExpiresAt,
			&withdrawalTS, &r.ConsentMechanism, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if assetID.Valid {
			if id, err := uuid.Parse(assetID.String); err == nil {
				r.AssetID = &id
			}
		}
		if consentExpiresAt.Valid {
			r.ConsentExpiresAt = &consentExpiresAt.Time
		}
		if withdrawalTS.Valid {
			r.WithdrawalTimestamp = &withdrawalTS.Time
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if records == nil {
		records = []consentRow{}
	}
	c.JSON(http.StatusOK, gin.H{"records": records, "total": len(records)})
}

// CreateConsentRecord handles POST /compliance/consent.
func (h *ConsentRecordsHandler) CreateConsentRecord(c *gin.Context) {
	tenantID, ok := extractTenantID(c)
	if !ok {
		return
	}

	var req createConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var assetIDParam interface{}
	if req.AssetID != nil && *req.AssetID != "" {
		parsed, err := uuid.Parse(*req.AssetID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset_id UUID"})
			return
		}
		assetIDParam = parsed
	}

	var r consentRow
	var assetID sql.NullString
	var consentExpiresAt sql.NullTime
	err := h.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO consent_records
		    (tenant_id, data_subject_id, asset_id, purpose,
		     consent_given_at, consent_expires_at, consent_mechanism, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
		RETURNING id, tenant_id, data_subject_id, asset_id,
		          purpose, consent_given_at, consent_expires_at,
		          withdrawal_timestamp, consent_mechanism, is_active, created_at, updated_at
	`, tenantID, req.DataSubjectID, assetIDParam, req.Purpose,
		req.ConsentGivenAt, req.ConsentExpiresAt, req.ConsentMechanism,
	).Scan(
		&r.ID, &r.TenantID, &r.DataSubjectID, &assetID,
		&r.Purpose, &r.ConsentGivenAt, &consentExpiresAt,
		new(sql.NullTime), &r.ConsentMechanism, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if assetID.Valid {
		if id, parseErr := uuid.Parse(assetID.String); parseErr == nil {
			r.AssetID = &id
		}
	}
	if consentExpiresAt.Valid {
		r.ConsentExpiresAt = &consentExpiresAt.Time
	}
	c.JSON(http.StatusCreated, r)
}

// WithdrawConsentRecord handles DELETE /compliance/consent/:id.
// Sets withdrawal_timestamp = NOW() and is_active = false.
func (h *ConsentRecordsHandler) WithdrawConsentRecord(c *gin.Context) {
	tenantID, ok := extractTenantID(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	consentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid consent record id"})
		return
	}

	result, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE consent_records
		SET withdrawal_timestamp = NOW(), is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND is_active = TRUE
	`, consentID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "consent record not found or already withdrawn"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":     "withdrawn",
		"message":    "Consent withdrawn successfully",
		"consent_id": consentID,
	})
}

// extractTenantID reads tenant_id from the Gin context (set by auth middleware).
// Returns uuid.Nil with ok=true when no tenant is present so that unauthenticated
// dev calls still work (empty result sets). Production deploys should always have
// the auth middleware in place so tenantID will be non-nil.
func extractTenantID(c *gin.Context) (uuid.UUID, bool) {
	var tenantID uuid.UUID

	if tid, exists := c.Get("tenant_id"); exists {
		switch v := tid.(type) {
		case uuid.UUID:
			tenantID = v
		case string:
			if parsed, err := uuid.Parse(v); err == nil {
				tenantID = parsed
			}
		}
	}
	return tenantID, true
}
