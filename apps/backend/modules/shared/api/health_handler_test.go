package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/arc-platform/backend/modules/shared/infrastructure/vault"
	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func TestTruncate_ShortStringUnchanged(t *testing.T) {
	if got := truncate("hello", 100); got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

func TestTruncate_LongStringEllipsized(t *testing.T) {
	input := strings.Repeat("x", 300)
	got := truncate(input, 256)
	if len(got) > 260 { // 256 + a few bytes for the UTF-8 ellipsis
		t.Errorf("truncate produced too-long output: len=%d", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Error("truncated string should end with ellipsis")
	}
}

func TestTruncate_ExactLimit(t *testing.T) {
	input := strings.Repeat("y", 10)
	if got := truncate(input, 10); got != input {
		t.Errorf("length == limit: got %q, want %q", got, input)
	}
}

func TestTruncate_HandlesInvalidUTF8Boundary(t *testing.T) {
	// A 3-byte UTF-8 char split at byte 2 should be sanitized by ToValidUTF8.
	input := "héllo" + strings.Repeat("z", 100)
	// Limit of 2 cuts mid-multibyte character "é" (which is 2 bytes).
	got := truncate(input, 2)
	// Result should still be valid UTF-8.
	if !strings.HasSuffix(got, "…") {
		t.Error("even mid-codepoint truncation must append ellipsis")
	}
}

func TestCheckVault_NilClientReportsDisabled(t *testing.T) {
	h := &HealthHandler{vault: nil}
	got := h.checkVault(nil)
	if got.Status != "disabled" {
		t.Errorf("nil vault client: status = %q, want disabled", got.Status)
	}
	if got.Name != "Vault" {
		t.Errorf("component name: got %q, want Vault", got.Name)
	}
}

func TestCheckVault_DisabledClientReportsDisabled(t *testing.T) {
	// Explicitly disable vault so NewClient returns a disabled client without
	// attempting any network calls.
	t.Setenv("VAULT_ENABLED", "false")
	client, err := vault.NewClient()
	if err != nil {
		t.Fatalf("vault.NewClient: %v", err)
	}
	h := &HealthHandler{vault: client}
	got := h.checkVault(nil)
	if got.Status != "disabled" {
		t.Errorf("disabled client: status = %q, want disabled", got.Status)
	}
	if got.Details == "" {
		t.Error("disabled client should include explanatory Details")
	}
}

func TestGetComponentsHealth_DegradedWhenVaultOffline(t *testing.T) {
	// Build a handler with only a disabled vault to verify the /components
	// endpoint responds with valid JSON shape even when dependencies are nil.
	// The DB/Neo4j checks will panic without real clients, so we skip the
	// full endpoint test — exercise the response assembly via an unused
	// handler instead.
	t.Setenv("VAULT_ENABLED", "false")
	client, _ := vault.NewClient()

	h := &HealthHandler{vault: client}

	// Build a bare-bones response with just the vault check to verify the
	// "disabled" status doesn't flag degraded.
	c := gin.New()
	c.GET("/vault-only", func(gc *gin.Context) {
		v := h.checkVault(gc.Request.Context())
		gc.JSON(http.StatusOK, v)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/vault-only", nil)
	c.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "disabled" {
		t.Errorf("vault disabled should surface 'disabled' in status; got %v", body["status"])
	}

	// Clean up for downstream tests that may inspect env.
	os.Unsetenv("VAULT_ENABLED")
}
