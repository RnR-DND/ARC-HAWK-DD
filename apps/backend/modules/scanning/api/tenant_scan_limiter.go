package api

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
)

// tenantScanLimiter bounds concurrent scan executor goroutines per tenant.
// Without this, a single tenant can spawn N unbounded goroutines (each holding
// ~500MB of findings in worst case) and OOM the backend for every tenant.
//
// The per-tenant cap is read from TENANT_MAX_CONCURRENT_SCANS (default 5).
type tenantScanLimiter struct {
	mu    sync.Mutex
	sems  map[uuid.UUID]*semaphore.Weighted
	perT  int64
}

var (
	tenantLimiterOnce sync.Once
	tenantLimiter     *tenantScanLimiter
)

func getTenantScanLimiter() *tenantScanLimiter {
	tenantLimiterOnce.Do(func() {
		perTenant := int64(5)
		if v := os.Getenv("TENANT_MAX_CONCURRENT_SCANS"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
				perTenant = n
			}
		}
		tenantLimiter = &tenantScanLimiter{
			sems: make(map[uuid.UUID]*semaphore.Weighted),
			perT: perTenant,
		}
	})
	return tenantLimiter
}

// Acquire blocks (up to 30s) for a scan slot on the given tenant. On success the
// caller MUST invoke the returned release fn when the scan goroutine exits.
// If the acquire times out, ok=false and release is a no-op.
func (l *tenantScanLimiter) Acquire(ctx context.Context, tenantID uuid.UUID) (release func(), ok bool) {
	sem := l.semFor(tenantID)
	acqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := sem.Acquire(acqCtx, 1); err != nil {
		return func() {}, false
	}
	var once sync.Once
	return func() { once.Do(func() { sem.Release(1) }) }, true
}

// TryAcquire returns immediately. Useful for the HTTP path where we want to
// 429 the client rather than have them wait.
func (l *tenantScanLimiter) TryAcquire(tenantID uuid.UUID) (release func(), ok bool) {
	sem := l.semFor(tenantID)
	if !sem.TryAcquire(1) {
		return func() {}, false
	}
	var once sync.Once
	return func() { once.Do(func() { sem.Release(1) }) }, true
}

func (l *tenantScanLimiter) semFor(tenantID uuid.UUID) *semaphore.Weighted {
	l.mu.Lock()
	defer l.mu.Unlock()
	if s, ok := l.sems[tenantID]; ok {
		return s
	}
	s := semaphore.NewWeighted(l.perT)
	l.sems[tenantID] = s
	return s
}
