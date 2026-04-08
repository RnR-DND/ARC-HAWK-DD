// Package worker contains background jobs for the discovery module.
//
// SnapshotWorker runs on a fixed interval (default 24h, configurable via
// DISCOVERY_SNAPSHOT_INTERVAL env var). On each tick, it iterates through all
// active tenants serially (cap 1 concurrent per E4 in autoplan eng review),
// taking a snapshot for each and running drift detection against the prior snapshot.
//
// Pattern reference: apps/backend/modules/scanning/module.go:97
// (ticker + stop chan + per-iteration timeout).
package worker

import (
	"context"
	"log"
	"time"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/discovery/service"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
)

// SnapshotWorker periodically snapshots inventory and detects drift for every active tenant.
type SnapshotWorker struct {
	snapshotService *service.SnapshotService
	driftDetector   *service.DriftDetectionService
	tenantLister    service.TenantLister
	interval        time.Duration
}

// NewSnapshotWorker creates a new background snapshot worker.
func NewSnapshotWorker(
	snapshotService *service.SnapshotService,
	driftDetector *service.DriftDetectionService,
	tenantLister service.TenantLister,
	interval time.Duration,
) *SnapshotWorker {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &SnapshotWorker{
		snapshotService: snapshotService,
		driftDetector:   driftDetector,
		tenantLister:    tenantLister,
		interval:        interval,
	}
}

// Run is the worker loop. Cancel via the stop channel.
func (w *SnapshotWorker) Run(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run once 30s after start to populate the first snapshot quickly. Without this,
	// brand-new deployments stare at an empty dashboard for 24 hours.
	initialDelay := time.NewTimer(30 * time.Second)
	defer initialDelay.Stop()

	for {
		select {
		case <-stop:
			log.Printf("📸 Discovery snapshot worker stopping")
			return
		case <-ctx.Done():
			log.Printf("📸 Discovery snapshot worker context cancelled")
			return
		case <-initialDelay.C:
			w.runOnce(ctx)
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

// runOnce iterates active tenants and takes one snapshot per tenant. Errors per
// tenant are logged but do not stop the loop.
func (w *SnapshotWorker) runOnce(ctx context.Context) {
	tenants, err := w.tenantLister.ListActiveTenants(ctx)
	if err != nil {
		log.Printf("⚠️  discovery worker: list tenants failed: %v", err)
		return
	}
	if len(tenants) == 0 {
		log.Printf("📸 Discovery worker: no active tenants")
		return
	}

	log.Printf("📸 Discovery worker: starting snapshot pass for %d tenant(s)", len(tenants))
	for _, tid := range tenants {
		// Per-tenant timeout — 5 min cap (matches scanning module's pattern).
		tenantCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		tenantCtx = context.WithValue(tenantCtx, persistence.TenantIDKey, tid)

		snap, err := w.snapshotService.TakeSnapshot(tenantCtx, domain.TriggerCron, nil)
		if err != nil {
			log.Printf("⚠️  discovery worker: snapshot failed for tenant %s: %v", tid, err)
			cancel()
			continue
		}

		// Drift detection against the previous snapshot.
		count, err := w.driftDetector.DetectDrift(tenantCtx, snap.ID)
		if err != nil {
			log.Printf("⚠️  discovery worker: drift detection failed for tenant %s: %v", tid, err)
		} else if count > 0 {
			log.Printf("📊 Discovery worker: %d drift events for tenant %s", count, tid)
		}

		cancel()
	}
	log.Printf("📸 Discovery worker: snapshot pass complete")
}
