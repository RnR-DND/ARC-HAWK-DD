package orchestrator

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// speedUpBackoff is a test hook: the retry loop sleeps between attempts, which
// slows tests. We can't easily override ingestBackoffBase (const), so tests
// that exercise multi-attempt paths accept the ~6s real-world delay by
// running with a small chunk count. For fast tests, rely on 2xx/4xx paths
// which never sleep.

func TestSendIngestChunkWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	err := sendIngestChunkWithRetry("tenant-1", srv.URL, []byte(`{"k":"v"}`), 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts: got %d, want 1", got)
	}
}

func TestSendIngestChunkWithRetry_FailFastOn4xx(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	err := sendIngestChunkWithRetry("tenant-1", srv.URL, []byte(`{}`), 0, 5)
	if err == nil {
		t.Fatal("expected error on 400")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("4xx should fail fast without retrying; attempts=%d", got)
	}
}

func TestSendIngestChunkWithRetry_RetriesOn5xxThenSucceeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	start := time.Now()
	err := sendIngestChunkWithRetry("tenant-1", srv.URL, []byte(`{}`), 0, 5)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("attempts: got %d, want 2", got)
	}
	// First retry backoff is ingestBackoffBase (2s). Success should take >= 2s.
	if elapsed < 2*time.Second {
		t.Errorf("expected ~2s backoff before second attempt; got %v", elapsed)
	}
}

func TestSendIngestChunkWithRetry_ExhaustsRetries(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := sendIngestChunkWithRetry("tenant-1", srv.URL, []byte(`{}`), 0, 5)
	if err == nil {
		t.Fatal("expected error after max attempts")
	}
	if got := atomic.LoadInt32(&attempts); got != int32(ingestMaxAttempts) {
		t.Errorf("attempts: got %d, want %d (ingestMaxAttempts)", got, ingestMaxAttempts)
	}
}

func TestSendIngestChunkWithRetry_ForwardsHeaders(t *testing.T) {
	// Restore package-level token after test.
	orig := scannerServiceToken
	scannerServiceToken = "test-token-abc"
	defer func() { scannerServiceToken = orig }()

	type capture struct {
		tenant string
		token  string
	}
	captured := make(chan capture, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- capture{
			tenant: r.Header.Get("X-Tenant-ID"),
			token:  r.Header.Get("X-Scanner-Token"),
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	err := sendIngestChunkWithRetry("tenant-42", srv.URL, []byte(`{}`), 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := <-captured
	if c.tenant != "tenant-42" {
		t.Errorf("X-Tenant-ID: got %q, want tenant-42", c.tenant)
	}
	if c.token != "test-token-abc" {
		t.Errorf("X-Scanner-Token: got %q, want test-token-abc", c.token)
	}
}

func TestSendIngestChunkWithRetry_TransportErrorRetries(t *testing.T) {
	// Closed server → transport error on every attempt. Should loop until
	// exhaustion and return the last error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // connections to this URL will now fail

	err := sendIngestChunkWithRetry("tenant-1", url, []byte(`{}`), 0, 5)
	if err == nil {
		t.Fatal("expected transport error")
	}
}
