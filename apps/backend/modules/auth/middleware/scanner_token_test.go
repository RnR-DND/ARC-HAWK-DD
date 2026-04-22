package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// These tests cover the global Authenticate() middleware's scanner-token
// recognition path added by the system audit (P0-2). The scanner holds a
// shared secret and uses X-Scanner-Token to call backend endpoints that are
// normally gated behind JWT. Without this branch, the global middleware
// would 401 every scanner call before per-route middleware could run.

func init() { gin.SetMode(gin.TestMode) }

// newBareAuthMiddleware builds an AuthMiddleware with only the public/skip
// path maps set. authenticateAPIKey returns false early when no X-API-Key
// header is present, so the DB-backed fields are never accessed along the
// code path these tests exercise.
func newBareAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{
		skipAuthPaths: map[string]bool{},
		publicPaths:   map[string]bool{},
	}
}

func TestAuthenticate_ScannerToken_ValidPassesThrough(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "shared-secret-xyz")

	m := newBareAuthMiddleware()
	r := gin.New()
	r.Use(m.Authenticate())
	r.GET("/x", func(c *gin.Context) {
		auth, _ := c.Get("service_auth")
		c.JSON(http.StatusOK, gin.H{"service_auth": auth})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Scanner-Token", "shared-secret-xyz")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("valid scanner token should pass: got %d", w.Code)
	}
}

func TestAuthenticate_ScannerToken_WrongTokenFallsThroughToJWTPath(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "shared-secret-xyz")
	// Make dev-mode anon fallback explicit so the request completes rather
	// than 401ing for an unrelated reason.
	t.Setenv("AUTH_REQUIRED", "false")

	m := newBareAuthMiddleware()
	r := gin.New()

	var capturedServiceAuth any
	r.Use(m.Authenticate())
	r.GET("/x", func(c *gin.Context) {
		capturedServiceAuth, _ = c.Get("service_auth")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Scanner-Token", "WRONG")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("wrong token should not 401 here (dev-mode anon fallback): got %d", w.Code)
	}
	// service_auth must NOT be set on a wrong-token request — otherwise
	// an attacker could impersonate the scanner by sending any string.
	if capturedServiceAuth != nil {
		t.Errorf("wrong scanner token set service_auth = %v; expected nil (unauthenticated as scanner)", capturedServiceAuth)
	}
}

func TestAuthenticate_ScannerToken_MissingExpectedEnvFallsThrough(t *testing.T) {
	t.Setenv("SCANNER_SERVICE_TOKEN", "")
	t.Setenv("AUTH_REQUIRED", "false")

	m := newBareAuthMiddleware()
	r := gin.New()
	r.Use(m.Authenticate())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Scanner-Token", "anything")
	r.ServeHTTP(w, req)

	// When SCANNER_SERVICE_TOKEN is unset the scanner branch is a no-op —
	// the request must fall through to the regular auth path. With
	// AUTH_REQUIRED=false that is anon-allowed.
	if w.Code != http.StatusOK {
		t.Fatalf("missing expected token should not grant scanner auth; got %d", w.Code)
	}
}
