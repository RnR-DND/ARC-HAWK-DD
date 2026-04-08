package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/arc-platform/backend/modules/discovery/domain"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/google/uuid"
)

// ReportService generates board-readable reports asynchronously via an in-process
// worker pool (per E6 in the autoplan eng review). Jobs are persisted to
// discovery_reports; workers pick them up and write the generated content back to the
// row.
//
// v1 supports html, csv, json natively. "pdf" format returns HTML with a print-to-PDF
// hint — v1.5 will add real PDF via gofpdf.
type ReportService struct {
	repo *Repo

	// In-process job queue.
	jobs    chan reportJob
	wg      sync.WaitGroup
	stopped chan struct{}
	once    sync.Once
}

type reportJob struct {
	ctx      context.Context
	reportID uuid.UUID
	format   domain.ReportFormat
	tenantID uuid.UUID
}

// NewReportService creates a new report service. Call StartWorkers to begin processing.
func NewReportService(repo *Repo) *ReportService {
	return &ReportService{
		repo:    repo,
		jobs:    make(chan reportJob, 32),
		stopped: make(chan struct{}),
	}
}

// StartWorkers spins up `n` background goroutines that consume report jobs.
func (s *ReportService) StartWorkers(n int) {
	for i := 0; i < n; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
}

// StopWorkers signals all workers to drain and exit. Blocks until they're done.
func (s *ReportService) StopWorkers() {
	s.once.Do(func() {
		close(s.stopped)
		close(s.jobs)
	})
	s.wg.Wait()
}

func (s *ReportService) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case <-s.stopped:
			return
		case job, ok := <-s.jobs:
			if !ok {
				return
			}
			s.processJob(job)
		}
	}
}

// EnqueueReport creates a report row in 'pending' and queues it for the worker pool.
// Returns the new report ID. The caller polls GET /reports/:id for status.
func (s *ReportService) EnqueueReport(ctx context.Context, snapshotID *uuid.UUID, format domain.ReportFormat, requestedBy *uuid.UUID) (*domain.Report, error) {
	tenantID, err := persistence.EnsureTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("enqueue report: %w", err)
	}

	rep := &domain.Report{
		TenantID:    tenantID,
		SnapshotID:  snapshotID,
		Format:      format,
		RequestedBy: requestedBy,
	}
	if err := s.repo.CreateReport(ctx, rep); err != nil {
		return nil, err
	}

	// Detach the context for the background job — the request context will be
	// cancelled when the HTTP handler returns. We pass tenantID explicitly via
	// a fresh background context with the tenant key set.
	bgCtx := context.WithValue(context.Background(), persistence.TenantIDKey, tenantID)
	select {
	case s.jobs <- reportJob{ctx: bgCtx, reportID: rep.ID, format: format, tenantID: tenantID}:
		return rep, nil
	default:
		// Queue full — fail fast rather than blocking.
		_ = s.repo.FailReport(ctx, rep.ID, "report queue full; try again later")
		rep.Status = domain.ReportFailed
		rep.Error = "report queue full"
		return rep, nil
	}
}

func (s *ReportService) processJob(job reportJob) {
	ctx, cancel := context.WithTimeout(job.ctx, 2*time.Minute)
	defer cancel()

	content, contentType, err := s.generateContent(ctx, job)
	if err != nil {
		log.Printf("⚠️  discovery report %s failed: %v", job.reportID.String()[:8], err)
		_ = s.repo.FailReport(ctx, job.reportID, err.Error())
		return
	}
	if err := s.repo.CompleteReport(ctx, job.reportID, content, contentType); err != nil {
		log.Printf("⚠️  discovery report %s persist failed: %v", job.reportID.String()[:8], err)
	}
}

// generateContent produces the report bytes for the requested format.
func (s *ReportService) generateContent(ctx context.Context, job reportJob) ([]byte, string, error) {
	rep, err := s.repo.GetReport(ctx, job.reportID)
	if err != nil {
		return nil, "", err
	}

	// Pull data: most recent completed snapshot if not specified.
	var snap *domain.Snapshot
	if rep.SnapshotID != nil {
		snap, err = s.repo.GetSnapshot(ctx, *rep.SnapshotID)
		if err != nil {
			return nil, "", err
		}
	} else {
		snap, err = s.repo.GetLastCompletedSnapshot(ctx, nil)
		if err != nil {
			return nil, "", err
		}
		if snap == nil {
			return nil, "", fmt.Errorf("no completed snapshot available")
		}
	}

	facts, err := s.repo.ListFactsForSnapshot(ctx, snap.ID)
	if err != nil {
		return nil, "", err
	}
	hotspots, _ := s.repo.ListTopRiskHotspots(ctx, 10)

	switch job.format {
	case domain.ReportJSON:
		return s.renderJSON(snap, facts, hotspots)
	case domain.ReportCSV:
		return s.renderCSV(snap, facts)
	case domain.ReportHTML, domain.ReportPDF:
		// PDF currently produces HTML — see v1.5 TODO.
		return s.renderHTML(snap, facts, hotspots)
	default:
		return nil, "", fmt.Errorf("unsupported report format: %s", job.format)
	}
}

func (s *ReportService) renderJSON(snap *domain.Snapshot, facts []*domain.SnapshotFact, hotspots []*domain.RiskHotspot) ([]byte, string, error) {
	payload := map[string]interface{}{
		"snapshot":  snap,
		"facts":     facts,
		"hotspots":  hotspots,
		"generated": time.Now().UTC(),
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return b, "application/json", nil
}

func (s *ReportService) renderCSV(snap *domain.Snapshot, facts []*domain.SnapshotFact) ([]byte, string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"snapshot_id", "taken_at", "source_id", "source_name", "classification", "asset_count", "finding_count", "sensitivity_avg"})
	for _, f := range facts {
		srcID := ""
		if f.SourceID != nil {
			srcID = f.SourceID.String()
		}
		_ = w.Write([]string{
			snap.ID.String(),
			snap.TakenAt.Format(time.RFC3339),
			srcID,
			f.SourceName,
			f.Classification,
			strconv.Itoa(f.AssetCount),
			strconv.Itoa(f.FindingCount),
			fmt.Sprintf("%.2f", f.SensitivityAvg),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "text/csv; charset=utf-8", nil
}

const boardReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Board Data Risk Report — {{.SnapshotDate}}</title>
<style>
  body { font-family: -apple-system, system-ui, sans-serif; max-width: 880px; margin: 2rem auto; color: #1a1a1a; padding: 0 1.5rem; }
  h1 { font-size: 1.8rem; border-bottom: 2px solid #1a1a1a; padding-bottom: 0.4rem; }
  h2 { font-size: 1.2rem; margin-top: 2rem; color: #444; }
  table { border-collapse: collapse; width: 100%; margin: 1rem 0; font-size: 0.9rem; }
  th, td { text-align: left; padding: 0.4rem 0.6rem; border-bottom: 1px solid #ddd; }
  th { background: #f5f5f5; font-weight: 600; }
  .kpi-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1rem; margin: 1rem 0; }
  .kpi { border: 1px solid #ddd; border-radius: 6px; padding: 1rem; text-align: center; }
  .kpi-label { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; color: #666; }
  .kpi-value { font-size: 1.6rem; font-weight: 700; margin-top: 0.3rem; }
  .footer { margin-top: 3rem; padding-top: 1rem; border-top: 1px solid #ddd; color: #666; font-size: 0.8rem; }
  @media print { body { max-width: 100%; padding: 0; } }
</style>
</head>
<body>

<h1>Board Data Risk Report</h1>
<p><strong>Snapshot:</strong> {{.SnapshotDate}} &nbsp;|&nbsp; <strong>Snapshot ID:</strong> {{.SnapshotID}}</p>

<h2>Executive Summary</h2>
<div class="kpi-grid">
  <div class="kpi"><div class="kpi-label">Sources</div><div class="kpi-value">{{.SourceCount}}</div></div>
  <div class="kpi"><div class="kpi-label">Assets</div><div class="kpi-value">{{.AssetCount}}</div></div>
  <div class="kpi"><div class="kpi-label">Findings</div><div class="kpi-value">{{.FindingCount}}</div></div>
  <div class="kpi"><div class="kpi-label">Composite Risk</div><div class="kpi-value">{{printf "%.1f" .RiskScore}}</div></div>
</div>

<h2>High-Risk Hotspots</h2>
{{if .Hotspots}}
<table>
  <thead><tr><th>Asset</th><th>Classification</th><th>Findings</th><th>Risk Score</th></tr></thead>
  <tbody>
  {{range .Hotspots}}
    <tr>
      <td>{{.AssetName}}</td>
      <td>{{.Classification}}</td>
      <td>{{.FindingCount}}</td>
      <td>{{printf "%.1f" .Score}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
{{else}}
<p>No risk hotspots recorded for this snapshot.</p>
{{end}}

<h2>Inventory by Classification × Source</h2>
{{if .Facts}}
<table>
  <thead><tr><th>Source</th><th>Classification</th><th>Assets</th><th>Findings</th><th>Avg Sensitivity</th></tr></thead>
  <tbody>
  {{range .Facts}}
    <tr>
      <td>{{if .SourceName}}{{.SourceName}}{{else}}(unknown){{end}}</td>
      <td>{{.Classification}}</td>
      <td>{{.AssetCount}}</td>
      <td>{{.FindingCount}}</td>
      <td>{{printf "%.1f" .SensitivityAvg}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
{{else}}
<p>No inventory facts available for this snapshot.</p>
{{end}}

<div class="footer">
  Generated by ARC-HAWK-DD Data Discovery Module · {{.GeneratedAt}}<br>
  This report is auto-generated. To produce a PDF, use your browser's Print → Save as PDF.
</div>

</body>
</html>`

type boardReportData struct {
	SnapshotID   string
	SnapshotDate string
	SourceCount  int
	AssetCount   int
	FindingCount int
	RiskScore    float64
	Hotspots     []*domain.RiskHotspot
	Facts        []*domain.SnapshotFact
	GeneratedAt  string
}

func (s *ReportService) renderHTML(snap *domain.Snapshot, facts []*domain.SnapshotFact, hotspots []*domain.RiskHotspot) ([]byte, string, error) {
	tmpl, err := template.New("board").Parse(boardReportTemplate)
	if err != nil {
		return nil, "", err
	}
	data := boardReportData{
		SnapshotID:   snap.ID.String(),
		SnapshotDate: snap.TakenAt.Format("2006-01-02"),
		SourceCount:  snap.SourceCount,
		AssetCount:   snap.AssetCount,
		FindingCount: snap.FindingCount,
		RiskScore:    snap.CompositeRiskScore,
		Hotspots:     hotspots,
		Facts:        facts,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "text/html; charset=utf-8", nil
}
