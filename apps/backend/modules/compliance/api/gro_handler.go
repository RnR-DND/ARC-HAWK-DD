package api

// GROHandler serves DPDP Act 2023 Sec 11 — Grievance Redressal Officer settings.
//
// Routes:
//   GET /compliance/gro-settings — return current tenant's GRO info
//   PUT /compliance/gro-settings — update gro_name, gro_email, gro_phone, is_significant_data_fiduciary

import (
	"database/sql"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

// GROHandler handles Grievance Redressal Officer configuration.
type GROHandler struct {
	db *sql.DB
}

// NewGROHandler creates a new GROHandler backed by the provided *sql.DB.
func NewGROHandler(db *sql.DB) *GROHandler {
	return &GROHandler{db: db}
}

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type groSettings struct {
	GROName                    *string `json:"gro_name"`
	GROEmail                   *string `json:"gro_email"`
	GROPhone                   *string `json:"gro_phone"`
	IsSignificantDataFiduciary bool    `json:"is_significant_data_fiduciary"`
}

// GetSettings godoc
// @Summary Get GRO settings
// @Description Returns the current tenant's Grievance Redressal Officer settings
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Success 200 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/gro-settings [get]
func (h *GROHandler) GetSettings(c *gin.Context) {
	tenantID, ok := extractTenantID(c)
	if !ok {
		return
	}

	var s groSettings
	var name, email, phone sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), `
		SELECT gro_name, gro_email, gro_phone, is_significant_data_fiduciary
		FROM tenants WHERE id = $1
	`, tenantID).Scan(&name, &email, &phone, &s.IsSignificantDataFiduciary)
	if err == sql.ErrNoRows {
		// GRO settings are optional — a tenant that has not been configured
		// yet returns an empty settings object rather than 404. The UI
		// page that loads this endpoint would otherwise break on every
		// fresh tenant (including the dev-mode default tenant).
		c.JSON(http.StatusOK, groSettings{})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	if name.Valid {
		s.GROName = &name.String
	}
	if email.Valid {
		s.GROEmail = &email.String
	}
	if phone.Valid {
		s.GROPhone = &phone.String
	}
	c.JSON(http.StatusOK, s)
}

// UpdateSettings godoc
// @Summary Update GRO settings
// @Description Updates the Grievance Redressal Officer configuration for the tenant
// @Tags compliance
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer {token}"
// @Security BearerAuth
// @Param body body groSettings true "GRO settings payload"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/v1/compliance/gro-settings [put]
func (h *GROHandler) UpdateSettings(c *gin.Context) {
	tenantID, ok := extractTenantID(c)
	if !ok {
		return
	}

	var req struct {
		GROName                    *string `json:"gro_name"`
		GROEmail                   *string `json:"gro_email"`
		GROPhone                   *string `json:"gro_phone"`
		IsSignificantDataFiduciary *bool   `json:"is_significant_data_fiduciary"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	if req.GROEmail != nil && *req.GROEmail != "" {
		if !emailRegexp.MatchString(*req.GROEmail) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid gro_email format"})
			return
		}
	}

	_, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE tenants
		SET gro_name = COALESCE($1, gro_name),
		    gro_email = COALESCE($2, gro_email),
		    gro_phone = COALESCE($3, gro_phone),
		    is_significant_data_fiduciary = COALESCE($4, is_significant_data_fiduciary),
		    updated_at = NOW()
		WHERE id = $5
	`, req.GROName, req.GROEmail, req.GROPhone, req.IsSignificantDataFiduciary, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}
