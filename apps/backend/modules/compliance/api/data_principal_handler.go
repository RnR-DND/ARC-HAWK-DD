package api

// DataPrincipalHandler serves DPDP Act 2023 Sec 7 — Data Principal Rights.
//
// Routes:
//   POST   /compliance/dpr              — submit a new data principal request
//   GET    /compliance/dpr              — list requests (paginated; filter by ?status=)
//   PATCH  /compliance/dpr/:id/status  — update status (admin)
//   GET    /compliance/dpr/stats        — counts by status + overdue compliance flag

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DataPrincipalHandler handles data principal rights request CRUD.
type DataPrincipalHandler struct {
	db *sql.DB
}

// NewDataPrincipalHandler creates a new DataPrincipalHandler backed by the provided *sql.DB.
func NewDataPrincipalHandler(db *sql.DB) *DataPrincipalHandler {
	return &DataPrincipalHandler{db: db}
}

type dprRow struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	RequestType        string     `json:"request_type"`
	Status             string     `json:"status"`
	DataPrincipalID    string     `json:"data_principal_id"`
	DataPrincipalEmail *string    `json:"data_principal_email,omitempty"`
	RequestDetails     *string    `json:"request_details,omitempty"`
	ResponseDetails    *string    `json:"response_details,omitempty"`
	DueDate            time.Time  `json:"due_date"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

var validRequestTypes = map[string]bool{
	"ACCESS": true, "CORRECTION": true, "ERASURE": true, "NOMINATION": true, "GRIEVANCE": true,
}

var validStatuses = map[string]bool{
	"PENDING": true, "IN_PROGRESS": true, "COMPLETED": true, "REJECTED": true,
}

// SubmitRequest handles POST /compliance/dpr.
func (h *DataPrincipalHandler) SubmitRequest(c *gin.Context) {
	tenantID, ok := extractTenantID(c)
	if !ok {
		return
	}

	var req struct {
		RequestType        string  `json:"request_type" binding:"required"`
		DataPrincipalID    string  `json:"data_principal_id" binding:"required"`
		DataPrincipalEmail *string `json:"data_principal_email"`
		RequestDetails     *string `json:"request_details"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validRequestTypes[req.RequestType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request_type; must be one of ACCESS, CORRECTION, ERASURE, NOMINATION, GRIEVANCE"})
		return
	}

	var r dprRow
	var email sql.NullString
	var details sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), `
		INSERT INTO data_principal_requests
		    (tenant_id, request_type, data_principal_id, data_principal_email, request_details)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, request_type, status, data_principal_id, data_principal_email,
		          request_details, response_details, due_date, resolved_at, created_at, updated_at
	`, tenantID, req.RequestType, req.DataPrincipalID, req.DataPrincipalEmail, req.RequestDetails,
	).Scan(
		&r.ID, &r.TenantID, &r.RequestType, &r.Status, &r.DataPrincipalID, &email,
		&details, new(sql.NullString), &r.DueDate, new(sql.NullTime), &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if email.Valid {
		r.DataPrincipalEmail = &email.String
	}
	if details.Valid {
		r.RequestDetails = &details.String
	}
	c.JSON(http.StatusCreated, r)
}

// ListRequests handles GET /compliance/dpr.
func (h *DataPrincipalHandler) ListRequests(c *gin.Context) {
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

	statusFilter := c.Query("status")
	var (
		rows *sql.Rows
		err  error
	)
	if statusFilter != "" {
		rows, err = h.db.QueryContext(c.Request.Context(), `
			SELECT id, tenant_id, request_type, status, data_principal_id, data_principal_email,
			       request_details, response_details, due_date, resolved_at, created_at, updated_at
			FROM data_principal_requests
			WHERE tenant_id = $1 AND status = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`, tenantID, statusFilter, limit, offset)
	} else {
		rows, err = h.db.QueryContext(c.Request.Context(), `
			SELECT id, tenant_id, request_type, status, data_principal_id, data_principal_email,
			       request_details, response_details, due_date, resolved_at, created_at, updated_at
			FROM data_principal_requests
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`, tenantID, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var requests []dprRow
	for rows.Next() {
		var r dprRow
		var email, requestDetails, responseDetails sql.NullString
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.RequestType, &r.Status, &r.DataPrincipalID, &email,
			&requestDetails, &responseDetails, &r.DueDate, &resolvedAt, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if email.Valid {
			r.DataPrincipalEmail = &email.String
		}
		if requestDetails.Valid {
			r.RequestDetails = &requestDetails.String
		}
		if responseDetails.Valid {
			r.ResponseDetails = &responseDetails.String
		}
		if resolvedAt.Valid {
			r.ResolvedAt = &resolvedAt.Time
		}
		requests = append(requests, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if requests == nil {
		requests = []dprRow{}
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests, "total": len(requests)})
}

// UpdateStatus handles PATCH /compliance/dpr/:id/status.
func (h *DataPrincipalHandler) UpdateStatus(c *gin.Context) {
	tenantID, ok := extractTenantID(c)
	if !ok {
		return
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	var req struct {
		Status          string  `json:"status" binding:"required"`
		ResponseDetails *string `json:"response_details"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status; must be one of PENDING, IN_PROGRESS, COMPLETED, REJECTED"})
		return
	}

	var resolvedAt interface{}
	if req.Status == "COMPLETED" || req.Status == "REJECTED" {
		resolvedAt = time.Now().UTC()
	}

	result, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE data_principal_requests
		SET status = $1,
		    response_details = COALESCE($2, response_details),
		    resolved_at = COALESCE($3::TIMESTAMPTZ, resolved_at),
		    updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5
	`, req.Status, req.ResponseDetails, resolvedAt, requestID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated", "request_id": requestID})
}

// GetStats handles GET /compliance/dpr/stats.
func (h *DataPrincipalHandler) GetStats(c *gin.Context) {
	tenantID, ok := extractTenantID(c)
	if !ok {
		return
	}

	rows, err := h.db.QueryContext(c.Request.Context(), `
		SELECT status, COUNT(*) FROM data_principal_requests
		WHERE tenant_id = $1 GROUP BY status
	`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	counts := map[string]int{"PENDING": 0, "IN_PROGRESS": 0, "COMPLETED": 0, "REJECTED": 0}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		counts[status] = count
	}

	var overdueCount int
	_ = h.db.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(*) FROM data_principal_requests
		WHERE tenant_id = $1 AND status = 'PENDING' AND created_at < NOW() - INTERVAL '30 days'
	`, tenantID).Scan(&overdueCount)

	compliant := overdueCount == 0
	msg := "No pending requests older than 30 days — Sec 7 compliant"
	if !compliant {
		msg = "Pending requests older than 30 days violate DPDPA Sec 7 response timeline"
	}
	c.JSON(http.StatusOK, gin.H{
		"counts":        counts,
		"overdue_count": overdueCount,
		"compliant":     compliant,
		"message":       msg,
	})
}
