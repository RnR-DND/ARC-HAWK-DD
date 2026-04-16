package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", HealthHandler)
	r.POST("/scan", ScanHandler)
	return r
}

func TestHealthHandler_Returns200(t *testing.T) {
	r := newTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /health = %d, want 200", w.Code)
	}
}

func TestHealthHandler_ResponseBody(t *testing.T) {
	r := newTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("health status = %q, want \"ok\"", body["status"])
	}
}

func TestScanHandler_MissingScanID_Returns400(t *testing.T) {
	r := newTestRouter()
	// ScanRequest with no ScanID
	payload, _ := json.Marshal(ScanRequest{Sources: []SourceConfig{}})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/scan", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /scan without scan_id = %d, want 400", w.Code)
	}
}

func TestScanHandler_InvalidJSON_Returns400(t *testing.T) {
	r := newTestRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/scan", bytes.NewBufferString("{not json}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST /scan with invalid JSON = %d, want 400", w.Code)
	}
}

func TestScanHandler_ValidRequest_Returns202(t *testing.T) {
	r := newTestRouter()
	payload, _ := json.Marshal(ScanRequest{
		ScanID:  "integration-test-scan-001",
		Sources: []SourceConfig{},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/scan", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Errorf("POST /scan with valid payload = %d, want 202", w.Code)
	}
}

func TestScanHandler_ValidRequest_ResponseBody(t *testing.T) {
	r := newTestRouter()
	scanID := "body-check-scan-001"
	payload, _ := json.Marshal(ScanRequest{
		ScanID:  scanID,
		Sources: []SourceConfig{},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/scan", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode scan response: %v", err)
	}
	if body["scan_id"] != scanID {
		t.Errorf("response scan_id = %q, want %q", body["scan_id"], scanID)
	}
	if body["status"] != "running" {
		t.Errorf("response status = %q, want \"running\"", body["status"])
	}
}
