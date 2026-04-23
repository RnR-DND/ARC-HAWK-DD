package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	authmiddleware "github.com/arc-platform/backend/modules/auth/middleware"
	"github.com/arc-platform/backend/modules/scanning/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var scanHandlerTenantID = uuid.MustParse("55555555-5555-5555-5555-555555555555")

func newScanStatusRouter(t *testing.T) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	repo := persistence.NewPostgresRepository(db)
	scanSvc := service.NewScanService(repo)
	h := NewScanStatusHandler(scanSvc, nil, repo, nil, nil)

	r := gin.New()
	injectCtx := func(c *gin.Context) {
		c.Set("tenant_context", authmiddleware.TenantContext{
			TenantID: scanHandlerTenantID,
			UserID:   uuid.New(),
			Role:     "admin",
		})
		c.Next()
	}

	r.GET("/scans/:id/status", injectCtx, h.GetScanStatus)
	r.GET("/scans/:id", injectCtx, h.GetScan)
	r.POST("/scans/:id/complete", h.CompleteScan) // no auth — uses X-Scanner-Token
	r.POST("/scans/:id/cancel", injectCtx, h.CancelScan)
	r.DELETE("/scans/:id", injectCtx, h.DeleteScan)
	return r, mock
}

// ─── GetScanStatus ────────────────────────────────────────────────────────────

func TestGetScanStatus_InvalidUUID_Returns400(t *testing.T) {
	r, _ := newScanStatusRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/scans/not-a-uuid/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestGetScanStatus_NotFound_Returns404(t *testing.T) {
	r, mock := newScanStatusRouter(t)

	scanID := uuid.New()
	// DB returns no rows → GetScanRunByID returns error → 404
	mock.ExpectQuery(`SELECT`).
		WithArgs(scanID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}))

	req := httptest.NewRequest(http.MethodGet, "/scans/"+scanID.String()+"/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── CompleteScan ─────────────────────────────────────────────────────────────

func TestCompleteScan_InvalidUUID_Returns400(t *testing.T) {
	r, _ := newScanStatusRouter(t)

	body, _ := json.Marshal(map[string]string{"status": "completed"})
	req := httptest.NewRequest(http.MethodPost, "/scans/not-a-uuid/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestCompleteScan_InvalidStatus_Returns400(t *testing.T) {
	r, _ := newScanStatusRouter(t)

	scanID := uuid.New()
	body, _ := json.Marshal(map[string]string{"status": "in-progress"}) // not a valid terminal status
	req := httptest.NewRequest(http.MethodPost, "/scans/"+scanID.String()+"/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCompleteScan_MissingBody_Returns400(t *testing.T) {
	r, _ := newScanStatusRouter(t)

	scanID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/scans/"+scanID.String()+"/complete", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing body, got %d", w.Code)
	}
}

// ─── CancelScan ───────────────────────────────────────────────────────────────

func TestCancelScan_InvalidUUID_Returns400(t *testing.T) {
	r, _ := newScanStatusRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/scans/not-a-uuid/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ─── DeleteScan ───────────────────────────────────────────────────────────────

func TestDeleteScan_InvalidUUID_Returns400(t *testing.T) {
	r, _ := newScanStatusRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/scans/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
