package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func newRouterWithAuth() *gin.Engine {
	r := gin.New()
	r.Use(ServiceTokenAuth())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})
	return r
}

func TestServiceTokenAuth_RejectsMissingToken(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "expected-token")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newRouterWithAuth()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token: got %d, want 401", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "unauthorized" {
		t.Errorf("error code: got %v, want unauthorized", body["error"])
	}
}

func TestServiceTokenAuth_RejectsWrongToken(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "expected-token")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newRouterWithAuth()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-Scanner-Token", "wrong-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token: got %d, want 401", w.Code)
	}
}

func TestServiceTokenAuth_AcceptsCorrectToken(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "expected-token")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newRouterWithAuth()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-Scanner-Token", "expected-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("correct token: got %d, want 200", w.Code)
	}
}

func TestServiceTokenAuth_Misconfigured(t *testing.T) {
	// Required but no token set — scanner should refuse to run.
	t.Setenv("SCANNER_SERVICE_TOKEN", "")
	t.Setenv("SCANNER_AUTH_REQUIRED", "true")

	r := newRouterWithAuth()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-Scanner-Token", "whatever")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("misconfigured: got %d, want 503", w.Code)
	}
}

func TestServiceTokenAuth_DevModeBypass(t *testing.T) {
	// Explicit dev mode — auth is skipped.
	t.Setenv("SCANNER_SERVICE_TOKEN", "expected-token")
	t.Setenv("SCANNER_AUTH_REQUIRED", "false")

	r := newRouterWithAuth()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	// No token header.
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("dev bypass: got %d, want 200", w.Code)
	}
}
