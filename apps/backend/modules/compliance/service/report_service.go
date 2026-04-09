package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"
)

// ReportService generates DPDPA compliance gap reports.
type ReportService struct {
	obligationSvc *DPDPAObligationService
}

// NewReportService creates a new report service.
func NewReportService(obligationSvc *DPDPAObligationService) *ReportService {
	return &ReportService{obligationSvc: obligationSvc}
}

// GenerateHTMLReport builds a DPDPA compliance gap report as an HTML document.
// The HTML is styled for print: the caller can return it with Content-Type text/html
// and the browser's print-to-PDF flow produces a board-ready PDF.
func (s *ReportService) GenerateHTMLReport(ctx context.Context) ([]byte, error) {
	report, err := s.obligationSvc.BuildGapReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate report: %w", err)
	}

	tmpl, err := template.New("dpdpa_report").Funcs(template.FuncMap{
		"now": time.Now,
		"statusClass": func(status ObligationStatus) string {
			switch status {
			case StatusPass:
				return "pass"
			case StatusFail:
				return "fail"
			default:
				return "unknown"
			}
		},
		"sectionTitle": func(o DPDPAObligation) string {
			titles := map[DPDPAObligation]string{
				ObligationSec4LawfulProcessing:  "Section 4 — Lawful Processing",
				ObligationSec5PurposeLimitation: "Section 5 — Purpose Limitation",
				ObligationSec6Consent:           "Section 6 — Consent",
				ObligationSec8DataAccuracy:      "Section 8 — Data Accuracy",
				ObligationSec9ChildrensData:     "Section 9 — Children's Data",
				ObligationSec10DataFiduciary:    "Section 10 — Data Fiduciary Duties",
				ObligationSec17Retention:        "Section 17 — Retention",
			}
			if t, ok := titles[o]; ok {
				return t
			}
			return string(o)
		},
	}).Parse(dpdpaReportTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse report template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, report); err != nil {
		return nil, fmt.Errorf("execute report template: %w", err)
	}
	return buf.Bytes(), nil
}

var dpdpaReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>DPDPA 2023 Compliance Gap Report</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Segoe UI', Arial, sans-serif; font-size: 13px; color: #1a1a2e; background: #fff; }
  .cover { padding: 60px 48px; border-bottom: 3px solid #2563eb; }
  .cover h1 { font-size: 28px; font-weight: 700; color: #1e3a8a; }
  .cover p { margin-top: 8px; color: #64748b; font-size: 14px; }
  .section { padding: 32px 48px; border-bottom: 1px solid #e2e8f0; }
  .section h2 { font-size: 16px; font-weight: 600; color: #1e3a8a; margin-bottom: 16px; }
  .summary-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 24px; }
  .metric { background: #f8fafc; border-radius: 8px; padding: 16px; text-align: center; }
  .metric .value { font-size: 32px; font-weight: 700; color: #1e3a8a; }
  .metric .label { font-size: 12px; color: #64748b; margin-top: 4px; }
  .metric.fail .value { color: #dc2626; }
  .metric.pass .value { color: #16a34a; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 16px; font-size: 12px; }
  th { background: #1e3a8a; color: #fff; text-align: left; padding: 8px 12px; }
  td { padding: 8px 12px; border-bottom: 1px solid #e2e8f0; }
  tr:nth-child(even) td { background: #f8fafc; }
  .badge { display: inline-block; border-radius: 4px; padding: 2px 8px; font-size: 11px; font-weight: 600; }
  .badge.pass { background: #dcfce7; color: #166534; }
  .badge.fail { background: #fee2e2; color: #991b1b; }
  .badge.unknown { background: #fef3c7; color: #92400e; }
  .footer { padding: 24px 48px; color: #94a3b8; font-size: 11px; }
  @media print {
    .section { page-break-inside: avoid; }
    body { font-size: 11px; }
  }
</style>
</head>
<body>

<div class="cover">
  <h1>DPDPA 2023 Compliance Gap Report</h1>
  <p>Generated: {{.GeneratedAt.Format "02 Jan 2006, 15:04 UTC"}} &nbsp;|&nbsp; Total Assets Assessed: {{.TotalAssets}}</p>
  <p style="margin-top:16px;color:#374151;">
    This report maps each data asset against the obligations of India's Digital Personal Data Protection Act 2023.
    Gaps require remediation action. Evidence IDs link to specific findings in the ARC-HAWK-DD platform.
  </p>
</div>

<div class="section">
  <h2>Executive Summary</h2>
  <div class="summary-grid">
    <div class="metric fail">
      <div class="value">{{.Summary.FailCount}}</div>
      <div class="label">Compliance Gaps</div>
    </div>
    <div class="metric pass">
      <div class="value">{{.Summary.PassCount}}</div>
      <div class="label">Obligations Met</div>
    </div>
    <div class="metric">
      <div class="value">{{.Summary.UnknownCount}}</div>
      <div class="label">Requires Review</div>
    </div>
    <div class="metric">
      <div class="value">{{.TotalAssets}}</div>
      <div class="label">Assets Assessed</div>
    </div>
  </div>
</div>

{{range $obligation, $gaps := .GapsBySection}}
<div class="section">
  <h2>{{sectionTitle $obligation}}</h2>
  {{if $gaps}}
  <table>
    <thead>
      <tr><th>Asset</th><th>Status</th><th>Detail</th></tr>
    </thead>
    <tbody>
      {{range $gaps}}
      <tr>
        <td>{{.AssetName}}</td>
        <td><span class="badge {{.Status}}">{{.Status}}</span></td>
        <td>{{.Detail}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
  {{else}}
  <p style="color:#64748b;">No assets assessed for this obligation.</p>
  {{end}}
</div>
{{end}}

<div class="footer">
  ARC-HAWK-DD &mdash; Confidential &mdash; For internal DPO use only &mdash; {{.GeneratedAt.Format "2006"}}
</div>
</body>
</html>`
