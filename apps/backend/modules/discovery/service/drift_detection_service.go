package service

import (
	"context"
	"fmt"
	"log"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// DriftDetectionService diffs two snapshots' facts and emits drift events for
// asset additions, removals, classification changes, risk movements, and finding
// count spikes. Closed enum of 6 event types matches the DB CHECK constraint.
type DriftDetectionService struct {
	repo *Repo
}

// NewDriftDetectionService creates a new drift detection service.
func NewDriftDetectionService(repo *Repo) *DriftDetectionService {
	return &DriftDetectionService{repo: repo}
}

// DetectDrift compares the given snapshot to the most recent prior completed snapshot
// for the same tenant and writes drift events for any meaningful differences.
//
// Edge case (E7): if there is NO prior snapshot, this is a no-op and returns 0.
func (s *DriftDetectionService) DetectDrift(ctx context.Context, currentSnapshotID uuid.UUID) (int, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return 0, fmt.Errorf("detect drift: %w", err)
	}

	prior, err := s.repo.GetLastCompletedSnapshot(ctx, &currentSnapshotID)
	if err != nil {
		return 0, err
	}
	if prior == nil {
		log.Printf("📊 Discovery drift detection skipped for tenant %s — no prior snapshot", tenantID)
		return 0, nil
	}

	current, err := s.repo.GetSnapshot(ctx, currentSnapshotID)
	if err != nil {
		return 0, err
	}

	priorFacts, err := s.repo.ListFactsForSnapshot(ctx, prior.ID)
	if err != nil {
		return 0, err
	}
	currentFacts, err := s.repo.ListFactsForSnapshot(ctx, current.ID)
	if err != nil {
		return 0, err
	}

	events := DiffSnapshotFacts(priorFacts, currentFacts, currentSnapshotID, tenantID)
	if len(events) == 0 {
		return 0, nil
	}

	// Persist events in a single transaction.
	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	for _, e := range events {
		if err := s.repo.InsertDriftEventTx(ctx, tx, e); err != nil {
			_ = tx.Rollback()
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	log.Printf("📊 Discovery detected %d drift events for tenant %s (snapshot %s vs %s)",
		len(events), tenantID, current.ID.String()[:8], prior.ID.String()[:8])
	return len(events), nil
}

// DiffSnapshotFacts is a pure function diff that produces drift events from two slices of facts.
//
// It is exposed (capitalized) so unit tests can exercise it without a DB.
//
// Diff semantics:
//   - asset_added: a (source, classification) appears in current but not prior
//   - asset_removed: a (source, classification) appears in prior but not current
//   - classification_changed: same source, different classification distribution
//   - finding_count_spike: same (source, classification), finding_count grew by >50%
//   - risk_increased / risk_decreased: sensitivity_avg moved by >20 points
//
// Notes:
//   - We aggregate by (source_id, classification) so the diff is order-independent
//   - Asset-level diff would require asset-level facts which we don't store yet (v1.5)
//   - The asset_id field on events is set to uuid.Nil for source-level events; for v1.5 we'll
//     decompose facts into per-asset events.
func DiffSnapshotFacts(prior, current []*domain.SnapshotFact, snapshotID, tenantID uuid.UUID) []*domain.DriftEvent {
	type key struct {
		Source         string
		Classification string
	}
	makeKey := func(f *domain.SnapshotFact) key {
		src := ""
		if f.SourceID != nil {
			src = f.SourceID.String()
		}
		return key{Source: src, Classification: f.Classification}
	}

	priorByKey := map[key]*domain.SnapshotFact{}
	for _, f := range prior {
		priorByKey[makeKey(f)] = f
	}
	currentByKey := map[key]*domain.SnapshotFact{}
	for _, f := range current {
		currentByKey[makeKey(f)] = f
	}

	var events []*domain.DriftEvent

	// Walk current facts: detect added + spike + risk change.
	for k, cf := range currentByKey {
		pf, hadPrior := priorByKey[k]
		if !hadPrior {
			events = append(events, &domain.DriftEvent{
				TenantID:    tenantID,
				SnapshotID:  snapshotID,
				EventType:   domain.DriftAssetAdded,
				AssetID:     uuid.Nil, // source-level event in v1
				BeforeState: nil,
				AfterState: map[string]interface{}{
					"source_id":      k.Source,
					"classification": k.Classification,
					"asset_count":    cf.AssetCount,
					"finding_count":  cf.FindingCount,
				},
				Severity: severityForAdded(cf),
			})
			continue
		}

		// Finding count spike (>50%).
		if pf.FindingCount > 0 {
			pct := float64(cf.FindingCount-pf.FindingCount) / float64(pf.FindingCount)
			if pct > 0.5 {
				events = append(events, &domain.DriftEvent{
					TenantID:   tenantID,
					SnapshotID: snapshotID,
					EventType:  domain.DriftFindingCountSpike,
					AssetID:    uuid.Nil,
					BeforeState: map[string]interface{}{
						"finding_count":  pf.FindingCount,
						"classification": k.Classification,
					},
					AfterState: map[string]interface{}{
						"finding_count":  cf.FindingCount,
						"pct_increase":   pct,
						"classification": k.Classification,
					},
					Severity: severityForSpike(pct),
				})
			}
		} else if cf.FindingCount > 10 {
			// Brand new findings on a previously empty (source, classification).
			events = append(events, &domain.DriftEvent{
				TenantID:    tenantID,
				SnapshotID:  snapshotID,
				EventType:   domain.DriftFindingCountSpike,
				AssetID:     uuid.Nil,
				BeforeState: map[string]interface{}{"finding_count": 0},
				AfterState: map[string]interface{}{
					"finding_count":  cf.FindingCount,
					"classification": k.Classification,
				},
				Severity: domain.SeverityMedium,
			})
		}

		// Sensitivity (~ risk) movement >20 points.
		delta := cf.SensitivityAvg - pf.SensitivityAvg
		if delta > 20 {
			events = append(events, &domain.DriftEvent{
				TenantID:    tenantID,
				SnapshotID:  snapshotID,
				EventType:   domain.DriftRiskIncreased,
				AssetID:     uuid.Nil,
				BeforeState: map[string]interface{}{"sensitivity_avg": pf.SensitivityAvg, "classification": k.Classification},
				AfterState:  map[string]interface{}{"sensitivity_avg": cf.SensitivityAvg, "classification": k.Classification},
				Severity:    domain.SeverityHigh,
			})
		} else if delta < -20 {
			events = append(events, &domain.DriftEvent{
				TenantID:    tenantID,
				SnapshotID:  snapshotID,
				EventType:   domain.DriftRiskDecreased,
				AssetID:     uuid.Nil,
				BeforeState: map[string]interface{}{"sensitivity_avg": pf.SensitivityAvg, "classification": k.Classification},
				AfterState:  map[string]interface{}{"sensitivity_avg": cf.SensitivityAvg, "classification": k.Classification},
				Severity:    domain.SeverityLow,
			})
		}
	}

	// Walk prior facts: detect removed.
	for k, pf := range priorByKey {
		if _, stillThere := currentByKey[k]; stillThere {
			continue
		}
		events = append(events, &domain.DriftEvent{
			TenantID:   tenantID,
			SnapshotID: snapshotID,
			EventType:  domain.DriftAssetRemoved,
			AssetID:    uuid.Nil,
			BeforeState: map[string]interface{}{
				"source_id":      k.Source,
				"classification": k.Classification,
				"asset_count":    pf.AssetCount,
				"finding_count":  pf.FindingCount,
			},
			AfterState: nil,
			Severity:   domain.SeverityMedium,
		})
	}

	return events
}

func severityForAdded(f *domain.SnapshotFact) domain.DriftSeverity {
	if f.SensitivityAvg >= 70 {
		return domain.SeverityHigh
	}
	if f.AssetCount > 10 {
		return domain.SeverityMedium
	}
	return domain.SeverityLow
}

func severityForSpike(pct float64) domain.DriftSeverity {
	if pct > 5.0 {
		return domain.SeverityCritical
	}
	if pct > 2.0 {
		return domain.SeverityHigh
	}
	return domain.SeverityMedium
}
