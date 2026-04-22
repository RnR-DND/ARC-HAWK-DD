package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
)

func newTenantLimiter(perT int64) *tenantScanLimiter {
	return &tenantScanLimiter{
		sems: make(map[uuid.UUID]*semaphore.Weighted),
		perT: perT,
	}
}

func TestTenantScanLimiter_TryAcquire_SucceedsUnderCap(t *testing.T) {
	l := newTenantLimiter(2)
	tenant := uuid.New()

	rel1, ok := l.TryAcquire(tenant)
	if !ok {
		t.Fatal("first acquire should succeed")
	}
	defer rel1()

	rel2, ok := l.TryAcquire(tenant)
	if !ok {
		t.Fatal("second acquire should succeed (cap is 2)")
	}
	defer rel2()
}

func TestTenantScanLimiter_TryAcquire_Rejects429(t *testing.T) {
	l := newTenantLimiter(1)
	tenant := uuid.New()

	rel, ok := l.TryAcquire(tenant)
	if !ok {
		t.Fatal("first acquire should succeed")
	}

	if _, ok := l.TryAcquire(tenant); ok {
		t.Error("second acquire must fail when cap is 1 (simulates 429)")
	}

	rel() // release first slot
	if _, ok := l.TryAcquire(tenant); !ok {
		t.Error("after release, acquire should succeed again")
	}
}

func TestTenantScanLimiter_PerTenantIsolation(t *testing.T) {
	l := newTenantLimiter(1)
	tenantA := uuid.New()
	tenantB := uuid.New()

	relA, ok := l.TryAcquire(tenantA)
	if !ok {
		t.Fatal("tenant A acquire should succeed")
	}
	defer relA()

	// Tenant A is saturated but tenant B must still get a slot —
	// one tenant's load must not affect another.
	relB, ok := l.TryAcquire(tenantB)
	if !ok {
		t.Fatal("tenant B acquire should succeed while tenant A is saturated")
	}
	defer relB()
}

func TestTenantScanLimiter_ReleaseIsIdempotent(t *testing.T) {
	l := newTenantLimiter(1)
	tenant := uuid.New()

	rel, ok := l.TryAcquire(tenant)
	if !ok {
		t.Fatal("acquire should succeed")
	}

	// Calling release twice must not panic or double-release (which would
	// let the next acquire through when the bucket is actually full).
	rel()
	rel()

	// Double-release + double-re-acquire would allow 2 slots if release
	// were not idempotent. Try to acquire twice to verify.
	r1, ok := l.TryAcquire(tenant)
	if !ok {
		t.Fatal("post-release first acquire should succeed")
	}
	defer r1()

	if _, ok := l.TryAcquire(tenant); ok {
		t.Error("double-release bug: second acquire succeeded when cap=1")
	}
}

func TestTenantScanLimiter_Acquire_BlocksThenUnblocks(t *testing.T) {
	l := newTenantLimiter(1)
	tenant := uuid.New()

	ctx := context.Background()
	rel1, ok := l.Acquire(ctx, tenant)
	if !ok {
		t.Fatal("first acquire should succeed")
	}

	// Second Acquire should block. Release after a brief delay so it unblocks.
	var wg sync.WaitGroup
	wg.Add(1)
	var got bool
	go func() {
		defer wg.Done()
		_, ok := l.Acquire(ctx, tenant)
		got = ok
	}()

	time.Sleep(50 * time.Millisecond)
	rel1()
	wg.Wait()

	if !got {
		t.Error("blocked acquire should succeed after release")
	}
}

func TestTenantScanLimiter_Acquire_TimesOut(t *testing.T) {
	l := newTenantLimiter(1)
	tenant := uuid.New()

	rel, _ := l.TryAcquire(tenant)
	defer rel()

	// Use a tight parent context so the 30s internal timeout never kicks in.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, ok := l.Acquire(ctx, tenant)
	elapsed := time.Since(start)

	if ok {
		t.Error("Acquire should fail when cap is exhausted and ctx deadline passes")
	}
	if elapsed > time.Second {
		t.Errorf("Acquire should honor parent ctx deadline; took %v", elapsed)
	}
}

func TestGetTenantScanLimiter_Singleton(t *testing.T) {
	a := getTenantScanLimiter()
	b := getTenantScanLimiter()
	if a != b {
		t.Error("getTenantScanLimiter must return the same instance")
	}
	if a.perT < 1 {
		t.Errorf("perTenant default should be >= 1, got %d", a.perT)
	}
}
