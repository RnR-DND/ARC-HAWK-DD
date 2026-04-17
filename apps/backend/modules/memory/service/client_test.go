package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestClient_AddDocument(t *testing.T) {
	var seenAuth, seenBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/v3/documents" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		seenBody = string(buf)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AddDocumentResponse{ID: "doc-1", Status: "queued"})
	}))
	defer srv.Close()

	os.Setenv("SUPERMEMORY_API_URL", srv.URL)
	os.Setenv("SUPERMEMORY_API_KEY", "test-key")
	os.Setenv("SUPERMEMORY_ENABLED", "true")
	t.Cleanup(func() {
		os.Unsetenv("SUPERMEMORY_API_URL")
		os.Unsetenv("SUPERMEMORY_API_KEY")
		os.Unsetenv("SUPERMEMORY_ENABLED")
	})

	c := NewClientFromEnv()
	if !c.Enabled() {
		t.Fatal("client should be enabled")
	}
	resp, err := c.AddDocument(context.Background(), Document{Content: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "doc-1" || resp.Status != "queued" {
		t.Errorf("bad resp: %+v", resp)
	}
	if seenAuth != "Bearer test-key" {
		t.Errorf("bad auth header: %q", seenAuth)
	}
	if !strings.Contains(seenBody, `"content":"hello"`) {
		t.Errorf("bad body: %s", seenBody)
	}
}

func TestClient_Disabled(t *testing.T) {
	os.Unsetenv("SUPERMEMORY_API_KEY")
	os.Unsetenv("SUPERMEMORY_ENABLED")
	c := NewClientFromEnv()
	if c.Enabled() {
		t.Fatal("client should be disabled without env")
	}
	_, err := c.AddDocument(context.Background(), Document{Content: "x"})
	if err != ErrDisabled {
		t.Errorf("want ErrDisabled, got %v", err)
	}
}

func TestClient_Search(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var q SearchQuery
		json.NewDecoder(r.Body).Decode(&q)
		if q.Limit != 5 {
			t.Errorf("expected limit=5, got %d", q.Limit)
		}
		json.NewEncoder(w).Encode(SearchResponse{
			Results: []SearchResult{{ID: "r1", Score: 0.9, Content: "match"}},
			Total:   1, TimingMs: 42,
		})
	}))
	defer srv.Close()

	os.Setenv("SUPERMEMORY_API_URL", srv.URL)
	os.Setenv("SUPERMEMORY_API_KEY", "test-key")
	os.Setenv("SUPERMEMORY_ENABLED", "true")
	t.Cleanup(func() {
		os.Unsetenv("SUPERMEMORY_API_URL")
		os.Unsetenv("SUPERMEMORY_API_KEY")
		os.Unsetenv("SUPERMEMORY_ENABLED")
	})

	c := NewClientFromEnv()
	resp, err := c.Search(context.Background(), SearchQuery{Q: "pan card", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 || resp.Results[0].ID != "r1" {
		t.Errorf("bad search resp: %+v", resp)
	}
}
