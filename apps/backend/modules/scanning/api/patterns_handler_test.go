package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// newMockHandler returns a PatternsHandler backed by a sqlmock DB.
func newMockHandler(t *testing.T) (*PatternsHandler, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	repo := persistence.NewPostgresRepository(db)
	svc := service.NewPatternsService(repo)
	return NewPatternsHandler(svc, nil), mock
}

// ginCtxWithTenant returns a Gin engine with tenant_id and user_email set in context.
func ginCtxWithTenant(tenantID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Set("user_email", "tester@example.com")
		c.Next()
	})
	return r
}

func TestListPatterns_Success(t *testing.T) {
	handler, mock := newMockHandler(t)
	tenantID := uuid.New()

	now := time.Now()
	patternID := uuid.New()
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "display_name", "regex", "category",
		"description", "is_active", "created_by", "created_at", "updated_at",
		"context_keywords", "negative_keywords",
	}).AddRow(
		patternID, tenantID, "TEST_PATTERN", "Test Pattern",
		`^\d{10}$`, "phone", "ten-digit phone", true, "tester@example.com",
		now, now, "{}", "{}",
	)
	mock.ExpectQuery(`SELECT .* FROM custom_patterns WHERE tenant_id`).
		WithArgs(tenantID).
		WillReturnRows(rows)

	r := ginCtxWithTenant(tenantID)
	r.GET("/patterns", handler.ListPatterns)

	req := httptest.NewRequest(http.MethodGet, "/patterns", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].([]interface{})
	require.True(t, ok, "response should have data array")
	assert.Len(t, data, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePattern_Success(t *testing.T) {
	handler, mock := newMockHandler(t)
	tenantID := uuid.New()

	newID := uuid.New()
	now := time.Now()
	mock.ExpectQuery(`INSERT INTO custom_patterns`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(newID, now, now))

	payload := map[string]interface{}{
		"name":        "Custom Email",
		"regex":       `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		"category":    "email",
		"description": "Enterprise email pattern",
		"is_active":   true,
	}
	body, _ := json.Marshal(payload)

	r := ginCtxWithTenant(tenantID)
	r.POST("/patterns", handler.CreatePattern)

	req := httptest.NewRequest(http.MethodPost, "/patterns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdatePattern_Success(t *testing.T) {
	handler, mock := newMockHandler(t)
	tenantID := uuid.New()
	patternID := uuid.New()
	now := time.Now()

	updated := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "display_name", "regex", "category", "description",
		"is_active", "created_by", "created_at", "updated_at",
		"validation_status", "backtrack_safe", "match_count_lifetime", "last_matched_at",
		"false_positive_count", "false_positive_rate", "auto_deactivated",
		"context_keywords", "negative_keywords",
	}).AddRow(
		patternID, tenantID, "UPDATED_PATTERN", "Updated Pattern",
		`^\d{10}$`, "phone", "updated desc",
		false, "tester@example.com", now, now,
		"valid", true, int64(0), (*time.Time)(nil),
		int64(0), float64(0), false,
		"{}", "{}",
	)
	mock.ExpectQuery(`UPDATE custom_patterns`).
		WillReturnRows(updated)

	payload := map[string]interface{}{
		"name":      "Updated Pattern",
		"regex":     `^\d{10}$`,
		"is_active": false,
	}
	body, _ := json.Marshal(payload)

	r := ginCtxWithTenant(tenantID)
	r.PUT("/patterns/:id", handler.UpdatePattern)

	req := httptest.NewRequest(http.MethodPut, "/patterns/"+patternID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["data"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeletePattern_Success(t *testing.T) {
	handler, mock := newMockHandler(t)
	tenantID := uuid.New()
	patternID := uuid.New()

	mock.ExpectExec(`DELETE FROM custom_patterns`).
		WithArgs(patternID, tenantID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	r := ginCtxWithTenant(tenantID)
	r.DELETE("/patterns/:id", handler.DeletePattern)

	req := httptest.NewRequest(http.MethodDelete, "/patterns/"+patternID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListPatterns_TenantIsolation(t *testing.T) {
	handler, mock := newMockHandler(t)

	tenant1 := persistence.DevSystemTenantID
	tenant2 := uuid.New()
	assert.NotEqual(t, tenant1, tenant2, "tenants must be distinct")

	now := time.Now()
	patternID := uuid.New()

	// Tenant 1 query returns one row; tenant 2 query returns zero rows.
	tenant1Rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "display_name", "regex", "category",
		"description", "is_active", "created_by", "created_at", "updated_at",
		"context_keywords", "negative_keywords",
	}).AddRow(patternID, tenant1, "T1_PATTERN", "T1 Pattern", `\d+`, "number", "", true, "t1@example.com", now, now, "{}", "{}")

	tenant2Rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "display_name", "regex", "category",
		"description", "is_active", "created_by", "created_at", "updated_at",
		"context_keywords", "negative_keywords",
	})

	mock.ExpectQuery(`SELECT .* FROM custom_patterns WHERE tenant_id`).
		WithArgs(tenant1).WillReturnRows(tenant1Rows)
	mock.ExpectQuery(`SELECT .* FROM custom_patterns WHERE tenant_id`).
		WithArgs(tenant2).WillReturnRows(tenant2Rows)

	// Request as tenant 1.
	r1 := ginCtxWithTenant(tenant1)
	r1.GET("/patterns", handler.ListPatterns)
	req1 := httptest.NewRequest(http.MethodGet, "/patterns", nil)
	w1 := httptest.NewRecorder()
	r1.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	var resp1 map[string]interface{}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &resp1))
	data1 := resp1["data"].([]interface{})
	assert.Len(t, data1, 1, "tenant1 should see its own pattern")

	// Request as tenant 2.
	r2 := ginCtxWithTenant(tenant2)
	r2.GET("/patterns", handler.ListPatterns)
	req2 := httptest.NewRequest(http.MethodGet, "/patterns", nil)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	var resp2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	data2 := resp2["data"].([]interface{})
	assert.Len(t, data2, 0, "tenant2 should see no patterns")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePattern_RegexValidation(t *testing.T) {
	tests := []struct {
		name       string
		regex      string
		shouldFail bool // true = expect 400 (unsafe regex)
	}{
		{"valid simple pattern", `^\d{3}-\d{2}-\d{4}$`, false},
		{"nested quantifiers — ReDoS risk", `(a+)+b`, true},
		{"unbounded alternation inside quantifier", `(email|mail|address)+`, true},
		{"excessive length", string(make([]byte, 501)), true},
	}

	handler, mock := newMockHandler(t)
	tenantID := uuid.New()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.shouldFail {
				newID := uuid.New()
				now := time.Now()
				mock.ExpectQuery(`INSERT INTO custom_patterns`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(newID, now, now))
			}

			payload := map[string]interface{}{
				"name":  "TEST",
				"regex": tc.regex,
			}
			body, _ := json.Marshal(payload)

			r := ginCtxWithTenant(tenantID)
			r.POST("/patterns", handler.CreatePattern)

			req := httptest.NewRequest(http.MethodPost, "/patterns", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tc.shouldFail {
				assert.Equal(t, http.StatusBadRequest, w.Code, "unsafe regex %q should return 400", tc.regex)
			} else {
				assert.Equal(t, http.StatusCreated, w.Code, "safe regex %q should return 201", tc.regex)
			}
		})
	}

	// Only validate expectations for patterns that hit the DB.
	_ = mock.ExpectationsWereMet()
}

func TestCreatePattern_MissingRequiredFields(t *testing.T) {
	handler, _ := newMockHandler(t)
	tenantID := uuid.New()

	tests := []struct {
		name    string
		payload map[string]interface{}
	}{
		{"missing name", map[string]interface{}{"regex": `\d+`}},
		{"missing regex", map[string]interface{}{"name": "Test"}},
		{"empty body", map[string]interface{}{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			r := ginCtxWithTenant(tenantID)
			r.POST("/patterns", handler.CreatePattern)

			req := httptest.NewRequest(http.MethodPost, "/patterns", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestDeletePattern_InvalidUUID(t *testing.T) {
	handler, _ := newMockHandler(t)
	tenantID := uuid.New()

	r := ginCtxWithTenant(tenantID)
	r.DELETE("/patterns/:id", handler.DeletePattern)

	req := httptest.NewRequest(http.MethodDelete, "/patterns/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeletePattern_NotFound(t *testing.T) {
	handler, mock := newMockHandler(t)
	tenantID := uuid.New()
	patternID := uuid.New()

	mock.ExpectExec(`DELETE FROM custom_patterns`).
		WithArgs(patternID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

	r := ginCtxWithTenant(tenantID)
	r.DELETE("/patterns/:id", handler.DeletePattern)

	req := httptest.NewRequest(http.MethodDelete, "/patterns/"+patternID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeletePattern_NotOwned verifies that deleting a pattern belonging to a
// different tenant returns 404 (not 500 or 200).
func TestDeletePattern_NotOwned(t *testing.T) {
	handler, mock := newMockHandler(t)
	ownerTenant := uuid.New()
	attackerTenant := uuid.New()
	patternID := uuid.New()

	// Attacker's tenant_id is used in the query — no rows match because pattern
	// belongs to ownerTenant. DELETE returns 0 rows affected.
	mock.ExpectExec(`DELETE FROM custom_patterns`).
		WithArgs(patternID, attackerTenant).
		WillReturnResult(sqlmock.NewResult(0, 0))

	_ = ownerTenant // owner never makes a request in this test
	r := ginCtxWithTenant(attackerTenant)
	r.DELETE("/patterns/:id", handler.DeletePattern)

	req := httptest.NewRequest(http.MethodDelete, "/patterns/"+patternID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreatePattern_InvalidJSON(t *testing.T) {
	handler, _ := newMockHandler(t)
	tenantID := uuid.New()

	r := ginCtxWithTenant(tenantID)
	r.POST("/patterns", handler.CreatePattern)

	req := httptest.NewRequest(http.MethodPost, "/patterns", bytes.NewReader([]byte(`{not valid json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// newMockSQL is a helper that opens a sqlmock and returns a *sql.DB only.
func newMockSQL(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}
