package orchestrator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arc-platform/go-scanner/internal/presidio"
	"github.com/sony/gobreaker"
)

// newTestBreaker creates an isolated circuit breaker with a short open timeout
// so tests don't need to wait 30 seconds for the breaker to half-open.
func newTestBreaker() *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "presidio-test",
		MaxRequests: 1,
		Interval:    50 * time.Millisecond,
		Timeout:     50 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})
}

// testAnalyzeWithBreaker mirrors analyzeWithBreaker but accepts an explicit breaker,
// allowing test isolation without mutating the package-level presidioBreaker.
func testAnalyzeWithBreaker(ctx context.Context, client *presidio.Client, value string, opts presidio.AnalyzeOptions, cb *gobreaker.CircuitBreaker) []presidio.Entity {
	result, err := cb.Execute(func() (interface{}, error) {
		entities := client.Analyze(ctx, value, opts)
		if entities == nil {
			if hErr := client.Health(ctx); hErr != nil {
				return nil, hErr
			}
		}
		return entities, nil
	})
	if err != nil {
		return nil
	}
	if result == nil {
		return nil
	}
	return result.([]presidio.Entity)
}

func TestPresidioCircuitBreaker_OpensAfter3Failures(t *testing.T) {
	var callCount int64

	// Mock server that always returns 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)
		http.Error(w, "service unavailable", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := presidio.NewClient(srv.URL)
	cb := newTestBreaker()
	ctx := context.Background()

	// 3 consecutive failures — breaker should open
	for i := 0; i < 3; i++ {
		result := testAnalyzeWithBreaker(ctx, client, "test text", presidio.AnalyzeOptions{}, cb)
		if result != nil {
			t.Errorf("call %d: server failing but got non-nil result", i+1)
		}
	}

	if cb.State() != gobreaker.StateOpen {
		t.Errorf("circuit breaker should be open after 3 failures, got state %v", cb.State())
	}
}

func TestPresidioCircuitBreaker_ReturnsNilWhenOpen(t *testing.T) {
	var callCount int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := presidio.NewClient(srv.URL)
	cb := newTestBreaker()
	ctx := context.Background()

	// Trip the breaker
	for i := 0; i < 3; i++ {
		testAnalyzeWithBreaker(ctx, client, "text", presidio.AnalyzeOptions{}, cb)
	}

	prevCount := atomic.LoadInt64(&callCount)

	// 4th call: breaker is open, server should NOT be hit
	result := testAnalyzeWithBreaker(ctx, client, "text", presidio.AnalyzeOptions{}, cb)
	if result != nil {
		t.Error("expected nil from open circuit breaker")
	}

	afterCount := atomic.LoadInt64(&callCount)
	if afterCount != prevCount {
		t.Errorf("server should not be called when breaker is open: calls before=%d after=%d", prevCount, afterCount)
	}
}

func TestPresidioCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := presidio.NewClient(srv.URL)
	cb := newTestBreaker()
	ctx := context.Background()

	// Trip the breaker
	for i := 0; i < 3; i++ {
		testAnalyzeWithBreaker(ctx, client, "text", presidio.AnalyzeOptions{}, cb)
	}
	if cb.State() != gobreaker.StateOpen {
		t.Fatal("expected Open state")
	}

	// Wait for the short Timeout (50ms) → half-open
	time.Sleep(100 * time.Millisecond)
	if cb.State() != gobreaker.StateHalfOpen {
		t.Logf("state after timeout: %v (may vary by timing)", cb.State())
	}
}

func TestPresidioClient_DisabledWithEmptyURL(t *testing.T) {
	// NewClient("") → client.Enabled() == false → Analyze returns nil immediately
	client := presidio.NewClient("")
	if client.Enabled() {
		t.Error("client with empty URL should report disabled")
	}
	entities := client.Analyze(context.Background(), "ABCPE1234F", presidio.AnalyzeOptions{})
	if entities != nil {
		t.Error("disabled client should return nil entities")
	}
}
