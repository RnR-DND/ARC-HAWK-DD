package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() { gin.SetMode(gin.TestMode) }

// handlerEcho reports the gin context values the middleware set, so tests
// can verify that tenant_id / user_id / user_role are seeded correctly when
// the scanner authenticates with a service token.
func handlerEcho(c *gin.Context) {
	tid, _ := c.Get("tenant_id")
	uid, _ := c.Get("user_id")
	urole, _ := c.Get("user_role")
	auth, _ := c.Get("authenticated")
	c.JSON(http.StatusOK, gin.H{
		"tenant_id":     tid,
		"user_id":       uid,
		"user_role":     urole,
		"authenticated": auth,
	})
}

func newCallbackRouter() *gin.Engine {
	r := gin.New()
	r.POST("/scans/:id/complete", ScannerCallbackAuth(), handlerEcho)
	return r
}

func TestScannerCallbackAuth_AllowsPriorAuth(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "secret")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := gin.New()
	// Simulate the global auth middleware running first and marking this
	// request as authenticated (e.g., valid JWT).
	r.Use(func(c *gin.Context) { c.Set("authenticated", true); c.Next() })
	r.POST("/scans/:id/complete", ScannerCallbackAuth(), handlerEcho)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/scans/abc/complete", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("prior auth should pass through: got %d", w.Code)
	}
}

func TestScannerCallbackAuth_RejectsMissingToken(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "secret")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newCallbackRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/scans/abc/complete", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token: got %d, want 401", w.Code)
	}
}

func TestScannerCallbackAuth_RejectsWrongToken(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "secret")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newCallbackRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/scans/abc/complete", nil)
	req.Header.Set("X-Scanner-Token", "nope")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token: got %d, want 401", w.Code)
	}
}

func TestScannerCallbackAuth_AcceptsValidToken_SeedsTenantFromHeader(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "secret")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	tenantID := uuid.New()
	r := newCallbackRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/scans/abc/complete", nil)
	req.Header.Set("X-Scanner-Token", "secret")
	req.Header.Set("X-Tenant-ID", tenantID.String())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("valid token: got %d, want 200", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["tenant_id"] != tenantID.String() {
		t.Errorf("tenant_id: got %v, want %s", body["tenant_id"], tenantID.String())
	}
	if body["user_role"] != "scanner" {
		t.Errorf("user_role: got %v, want scanner", body["user_role"])
	}
	if body["authenticated"] != true {
		t.Errorf("authenticated flag must be true after token auth; got %v", body["authenticated"])
	}
}

func TestScannerCallbackAuth_MissingTenantHeaderFallsBackToDev(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "secret")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newCallbackRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/scans/abc/complete", nil)
	req.Header.Set("X-Scanner-Token", "secret")
	// No X-Tenant-ID header.
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("valid token, no tenant: got %d, want 200", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["tenant_id"] == nil || body["tenant_id"] == "" {
		t.Error("tenant_id should fall back to DevSystemTenantID, got empty")
	}
}

func TestScannerCallbackAuth_Misconfigured(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newCallbackRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/scans/abc/complete", nil)
	req.Header.Set("X-Scanner-Token", "anything")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("misconfigured: got %d, want 503", w.Code)
	}
}

func TestScannerCallbackAuth_RejectsInvalidTenantUUID(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "secret")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newCallbackRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/scans/abc/complete", nil)
	req.Header.Set("X-Scanner-Token", "secret")
	req.Header.Set("X-Tenant-ID", "not-a-uuid")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("invalid uuid should still proceed via dev fallback; got %d", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["tenant_id"] == "not-a-uuid" {
		t.Error("invalid UUID should be rejected, not stored as tenant_id")
	}
}
