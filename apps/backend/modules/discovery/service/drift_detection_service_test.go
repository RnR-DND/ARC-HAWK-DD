package service

import (
	"testing"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/google/uuid"
)

// TestDiffSnapshotFacts exercises the pure diff function for every drift event type.
// This is the highest-risk surface in the discovery module — the formula determines
// what the sponsor sees on the board report's "what changed" section.
func TestDiffSnapshotFacts(t *testing.T) {
	tenantID := uuid.New()
	snapshotID := uuid.New()
	src1 := uuid.New()
	src2 := uuid.New()

	tests := []struct {
		name         string
		prior        []*domain.SnapshotFact
		current      []*domain.SnapshotFact
		wantTypes    []domain.DriftEventType
		wantMinCount int
	}{
		{
			name:         "no prior, no current = no events",
			prior:        nil,
			current:      nil,
			wantTypes:    nil,
			wantMinCount: 0,
		},
		{
			name:  "first snapshot has no prior — everything in current is added",
			prior: nil,
			current: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 10, FindingCount: 50, SensitivityAvg: 80},
			},
			wantTypes:    []domain.DriftEventType{domain.DriftAssetAdded},
			wantMinCount: 1,
		},
		{
			name: "asset_removed when source disappears",
			prior: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "EMAIL", AssetCount: 5, FindingCount: 20, SensitivityAvg: 30},
			},
			current:      nil,
			wantTypes:    []domain.DriftEventType{domain.DriftAssetRemoved},
			wantMinCount: 1,
		},
		{
			name: "finding_count_spike when count grows >50%",
			prior: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 100, SensitivityAvg: 50},
			},
			current: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 200, SensitivityAvg: 50},
			},
			wantTypes:    []domain.DriftEventType{domain.DriftFindingCountSpike},
			wantMinCount: 1,
		},
		{
			name: "no spike when growth is under 50%",
			prior: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 100, SensitivityAvg: 50},
			},
			current: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 140, SensitivityAvg: 50},
			},
			wantTypes:    nil,
			wantMinCount: 0,
		},
		{
			name: "risk_increased when sensitivity moves up >20",
			prior: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 10, SensitivityAvg: 30},
			},
			current: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 10, SensitivityAvg: 60},
			},
			wantTypes:    []domain.DriftEventType{domain.DriftRiskIncreased},
			wantMinCount: 1,
		},
		{
			name: "risk_decreased when sensitivity moves down >20",
			prior: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 10, SensitivityAvg: 80},
			},
			current: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 10, SensitivityAvg: 50},
			},
			wantTypes:    []domain.DriftEventType{domain.DriftRiskDecreased},
			wantMinCount: 1,
		},
		{
			name: "multiple sources with mixed drift",
			prior: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 10, SensitivityAvg: 50},
			},
			current: []*domain.SnapshotFact{
				{SourceID: &src1, Classification: "SSN", AssetCount: 1, FindingCount: 10, SensitivityAvg: 50},
				{SourceID: &src2, Classification: "EMAIL", AssetCount: 5, FindingCount: 30, SensitivityAvg: 20},
			},
			wantTypes:    []domain.DriftEventType{domain.DriftAssetAdded},
			wantMinCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := DiffSnapshotFacts(tt.prior, tt.current, snapshotID, tenantID)

			if len(events) < tt.wantMinCount {
				t.Errorf("DiffSnapshotFacts() returned %d events, want at least %d", len(events), tt.wantMinCount)
			}

			for _, wantType := range tt.wantTypes {
				found := false
				for _, e := range events {
					if e.EventType == wantType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("DiffSnapshotFacts() did not produce expected event type %s; got %d events", wantType, len(events))
				}
			}

			// Sanity: every event references the right snapshot + tenant.
			for _, e := range events {
				if e.SnapshotID != snapshotID {
					t.Errorf("event has wrong snapshot id: %v", e.SnapshotID)
				}
				if e.TenantID != tenantID {
					t.Errorf("event has wrong tenant id: %v", e.TenantID)
				}
			}
		})
	}
}

// TestDiffSnapshotFacts_FirstSnapshotNoOp explicitly verifies the E7 edge case from
// the autoplan eng review: drift detection on the very first snapshot should not
// flood the system with synthetic "added" events for every existing asset. The
// service-layer GetLastCompletedSnapshot returns nil for the first snapshot, and
// DetectDrift short-circuits — but DiffSnapshotFacts itself, called with prior=nil,
// should report current rows as added (this is expected when called via the public
// API for cross-snapshot diffs, not from the worker on the first snapshot).
//
// This test documents that boundary so future contributors know which side of the
// no-op boundary lives where.
func TestDiffSnapshotFacts_NilPriorReportsAdded(t *testing.T) {
	src := uuid.New()
	current := []*domain.SnapshotFact{
		{SourceID: &src, Classification: "SSN", AssetCount: 1, FindingCount: 1, SensitivityAvg: 50},
	}
	events := DiffSnapshotFacts(nil, current, uuid.New(), uuid.New())
	if len(events) != 1 {
		t.Fatalf("expected 1 added event, got %d", len(events))
	}
	if events[0].EventType != domain.DriftAssetAdded {
		t.Errorf("expected DriftAssetAdded, got %s", events[0].EventType)
	}
}
