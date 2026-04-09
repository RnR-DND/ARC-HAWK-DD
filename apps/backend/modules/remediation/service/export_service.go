package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/xuri/excelize/v2"
)

// RemediationExportRow represents one row in the export.
type RemediationExportRow struct {
	FindingID    string
	AssetName    string
	AssetPath    string
	PIIType      string
	FieldName    string
	Severity     string
	Status       string
	RemediatedAt string
	RemediatedBy string
	ActionTaken  string
}

// ExportReport fetches the remediation history and writes it in the requested format.
// format: "pdf" (HTML/print-to-PDF) | "xlsx"
func (s *RemediationService) ExportReport(ctx context.Context, format string) ([]byte, string, error) {
	rows, err := s.fetchExportRows(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("fetch remediation rows: %w", err)
	}

	switch format {
	case "xlsx":
		data, err := renderXLSX(rows)
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", err
	default: // "pdf" or unrecognized → HTML
		data, err := renderHTML(rows)
		return data, "text/html; charset=utf-8", err
	}
}

func (s *RemediationService) fetchExportRows(ctx context.Context) ([]RemediationExportRow, error) {
	const q = `
		SELECT
			f.id::text,
			COALESCE(a.name, ''),
			COALESCE(a.path, ''),
			COALESCE(f.pattern_name, ''),
			COALESCE(f.file_path, ''),
			COALESCE(f.severity, ''),
			COALESCE(f.status, ''),
			COALESCE(f.updated_at::text, ''),
			'',
			''
		FROM findings f
		LEFT JOIN assets a ON a.id = f.asset_id
		WHERE f.status IN ('remediated', 'approved', 'rejected')
		ORDER BY f.updated_at DESC
		LIMIT 5000
	`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RemediationExportRow
	for rows.Next() {
		var r RemediationExportRow
		if err := rows.Scan(
			&r.FindingID, &r.AssetName, &r.AssetPath,
			&r.PIIType, &r.FieldName, &r.Severity,
			&r.Status, &r.RemediatedAt, &r.RemediatedBy, &r.ActionTaken,
		); err != nil {
			continue
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// renderXLSX builds a multi-column spreadsheet using excelize.
func renderXLSX(rows []RemediationExportRow) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Remediation Report"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Finding ID", "Asset", "Path", "PII Type", "Field",
		"Severity", "Status", "Remediated At", "Remediated By", "Action Taken",
	}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Header style — bold, blue bg
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2563EB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A1", "J1", style)

	for i, r := range rows {
		row := i + 2
		vals := []any{
			r.FindingID, r.AssetName, r.AssetPath, r.PIIType, r.FieldName,
			r.Severity, r.Status, r.RemediatedAt, r.RemediatedBy, r.ActionTaken,
		}
		for col, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheet, cell, v)
		}
	}

	// Auto-fit approximate column widths
	colWidths := []float64{38, 24, 32, 18, 18, 10, 12, 22, 20, 24}
	for col, w := range colWidths {
		name, _ := excelize.ColumnNumberToName(col + 1)
		f.SetColWidth(sheet, name, name, w)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderHTML renders a print-ready HTML report (open in browser → Print → Save as PDF).
func renderHTML(rows []RemediationExportRow) ([]byte, error) {
	const tpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Remediation Report — ARC-HAWK-DD</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 0; padding: 32px; color: #111; font-size: 13px; }
  h1   { font-size: 22px; font-weight: 700; margin-bottom: 4px; }
  .meta { color: #6b7280; margin-bottom: 24px; font-size: 12px; }
  table { width: 100%; border-collapse: collapse; margin-top: 16px; }
  th { background: #2563eb; color: #fff; padding: 10px 12px; text-align: left; font-size: 11px; text-transform: uppercase; letter-spacing: 0.04em; }
  td { padding: 9px 12px; border-bottom: 1px solid #e5e7eb; vertical-align: top; }
  tr:nth-child(even) td { background: #f9fafb; }
  .badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
  .remediated { background: #d1fae5; color: #065f46; }
  .approved   { background: #dbeafe; color: #1e40af; }
  .rejected   { background: #fee2e2; color: #991b1b; }
  .critical   { background: #fef2f2; color: #b91c1c; }
  .high       { background: #fff7ed; color: #c2410c; }
  .medium     { background: #fefce8; color: #854d0e; }
  @media print { body { padding: 0; } }
</style>
</head>
<body>
<h1>Remediation Report</h1>
<div class="meta">ARC-HAWK-DD · Generated {{.Generated}} · {{.Total}} records</div>
<table>
  <thead>
    <tr>
      <th>Asset</th><th>PII Type</th><th>Field</th><th>Severity</th>
      <th>Status</th><th>Remediated At</th><th>Action</th>
    </tr>
  </thead>
  <tbody>
  {{range .Rows}}
  <tr>
    <td><strong>{{.AssetName}}</strong><br><span style="color:#6b7280;font-family:monospace;font-size:11px">{{.AssetPath}}</span></td>
    <td>{{.PIIType}}</td>
    <td>{{.FieldName}}</td>
    <td><span class="badge {{lower .Severity}}">{{.Severity}}</span></td>
    <td><span class="badge {{lower .Status}}">{{.Status}}</span></td>
    <td style="white-space:nowrap">{{.RemediatedAt}}</td>
    <td>{{.ActionTaken}}</td>
  </tr>
  {{else}}
  <tr><td colspan="7" style="text-align:center;padding:32px;color:#6b7280">No remediated findings yet.</td></tr>
  {{end}}
  </tbody>
</table>
</body>
</html>`

	funcMap := template.FuncMap{
		"lower": func(s string) string {
			result := ""
			for _, c := range s {
				if c >= 'A' && c <= 'Z' {
					result += string(c + 32)
				} else {
					result += string(c)
				}
			}
			return result
		},
	}

	t, err := template.New("report").Funcs(funcMap).Parse(tpl)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	data := struct {
		Generated string
		Total     int
		Rows      []RemediationExportRow
	}{
		Generated: time.Now().Format("2006-01-02 15:04 MST"),
		Total:     len(rows),
		Rows:      rows,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render template: %w", err)
	}
	return buf.Bytes(), nil
}
