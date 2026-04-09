package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// SpikeThreshold is the multiplier above the 7-day rolling average that triggers an alert.
// E.g. 2.5 means: if this scan has >2.5× the average weekly finding count, alert.
const SpikeThreshold = 2.5

// PiiSpikeAlert describes a PII volume spike detected for one asset.
type PiiSpikeAlert struct {
	AssetID        string    `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	CurrentCount   int       `json:"current_count"`
	RollingAverage float64   `json:"rolling_average_7d"`
	Ratio          float64   `json:"ratio"`
	DetectedAt     time.Time `json:"detected_at"`
}

// SpikeDetector compares current PII finding counts to a 7-day rolling average.
type SpikeDetector struct {
	db *sql.DB
}

// NewSpikeDetector creates a spike detector backed by the given DB.
func NewSpikeDetector(db *sql.DB) *SpikeDetector {
	return &SpikeDetector{db: db}
}

// Run queries the last completed scan's findings per asset and compares to the
// 7-day rolling average from risk_score_history. Returns any detected spikes.
//
// Phase 10 requirement: alert when a source's PII count jumps by ≥2.5× the 7-day baseline.
func (d *SpikeDetector) Run(ctx context.Context) ([]PiiSpikeAlert, error) {
	// Per-asset finding counts in the most recent completed scan
	const currentQuery = `
		SELECT a.id::text, a.name, COUNT(f.id) AS finding_count
		  FROM findings f
		  JOIN assets a ON a.id = f.asset_id
		  JOIN scan_runs sr ON sr.id = f.scan_run_id
		 WHERE sr.id = (
		     SELECT id FROM scan_runs
		      WHERE status = 'completed'
		      ORDER BY scan_completed_at DESC
		      LIMIT 1
		 )
		 GROUP BY a.id, a.name
	`
	rows, err := d.db.QueryContext(ctx, currentQuery)
	if err != nil {
		return nil, fmt.Errorf("current finding counts: %w", err)
	}
	defer rows.Close()

	type assetCount struct {
		id    string
		name  string
		count int
	}
	var current []assetCount
	for rows.Next() {
		var a assetCount
		if err := rows.Scan(&a.id, &a.name, &a.count); err != nil {
			continue
		}
		current = append(current, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(current) == 0 {
		return nil, nil
	}

	// 7-day rolling average from risk_score_history
	// We use finding_count proxy: count of findings per asset in the last 7 days
	const avgQuery = `
		SELECT asset_id::text, AVG(finding_count) AS avg_findings
		  FROM (
		      SELECT asset_id, COUNT(f.id) AS finding_count
		        FROM findings f
		        JOIN scan_runs sr ON sr.id = f.scan_run_id
		       WHERE sr.scan_completed_at >= NOW() - INTERVAL '7 days'
		         AND sr.status = 'completed'
		       GROUP BY asset_id, sr.id
		  ) sub
		 GROUP BY asset_id
	`
	avgRows, err := d.db.QueryContext(ctx, avgQuery)
	if err != nil {
		return nil, fmt.Errorf("rolling average query: %w", err)
	}
	defer avgRows.Close()

	rollingAvg := map[string]float64{}
	for avgRows.Next() {
		var assetID string
		var avg float64
		if err := avgRows.Scan(&assetID, &avg); err != nil {
			continue
		}
		rollingAvg[assetID] = avg
	}

	var alerts []PiiSpikeAlert
	for _, a := range current {
		avg, ok := rollingAvg[a.id]
		if !ok || avg < 10 {
			// No baseline or too few historical findings — skip
			continue
		}
		ratio := float64(a.count) / avg
		if ratio >= SpikeThreshold {
			alert := PiiSpikeAlert{
				AssetID:        a.id,
				AssetName:      a.name,
				CurrentCount:   a.count,
				RollingAverage: avg,
				Ratio:          ratio,
				DetectedAt:     time.Now().UTC(),
			}
			alerts = append(alerts, alert)
			log.Printf("[SPIKE] Asset %s: %d findings vs %.1f avg (ratio %.1fx) — SPIKE DETECTED",
				a.name, a.count, avg, ratio)
		}
	}
	return alerts, nil
}
