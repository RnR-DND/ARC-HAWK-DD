# Runbook: Low OCR Confidence

## Trigger

The scanner processes an image-based file (scanned PDF, TIFF, PNG) using OCR and the extracted text confidence score falls below the configured threshold (default: 60%). The result is marked unreliable.

Activates when the scanner logs `ocr_confidence < threshold` or when a finding shows `scan_status: low_ocr_confidence`.

---

## Symptoms

**Logs (scanner worker):**
```
WARN   hawk_scanner  file=scanned_contract.pdf  ocr_confidence=0.43  threshold=0.60  action=flagged_low_confidence
```

**Dashboard:** Asset shows `scan_warning: low_ocr_confidence`. Findings from the file are tagged `[LOW CONFIDENCE]` in the findings list.

**Metrics:** `hawk_scanner_ocr_confidence_histogram` bucket `<0.60` increments; `hawk_scanner_low_ocr_total` counter increments.

---

## Automated Response

The scanner extracts whatever text it can at the low confidence level and records findings tagged with `confidence: low`. It does **not** skip the file — partial extraction is better than no extraction for triage purposes. The asset is flagged in the database with `scan_warning = 'low_ocr_confidence'` so operators can prioritise re-review.

---

## Manual Steps

1. **Assess the source image quality.** Open the file and visually inspect it:
   - Is the scan resolution below 300 DPI? Low resolution is the most common cause.
   - Is the image skewed, rotated, or has significant noise?
   - Is the text handwritten rather than printed?

2. **Re-scan at higher resolution** if the source document is available:
   - Re-scan the physical document at 300 DPI or higher.
   - Save as a searchable PDF (PDF/A format preferred).
   - Replace the file in the source location.

3. **Pre-process the image** if you cannot re-scan:
   ```bash
   # Deskew and enhance contrast using ImageMagick
   convert -deskew 40% -normalize scanned_contract.pdf \
     /tmp/scanned_contract_enhanced.pdf
   ```

4. **Trigger a re-scan** on the improved file:
   ```bash
   curl -X POST http://localhost:8080/api/v1/scan/trigger \
     -H "Content-Type: application/json" \
     -d '{"source_type": "filesystem", "path": "/path/to/scanned_contract.pdf"}'
   ```

5. **Manually review** the low-confidence findings. For any finding tagged `[LOW CONFIDENCE]`, use the finding detail page to confirm or mark as false positive.

6. **Lower the confidence threshold** temporarily if the entire document corpus has inherently low-quality scans and the current threshold is generating too many warnings:
   ```yaml
   # In scanner config (hawk_scanner/config.yml)
   ocr:
     min_confidence: 0.45  # adjusted from 0.60
   ```
   This is a short-term measure — fix the underlying document quality issue.

---

## Resolution Criteria

- The asset's `scan_warning` field is cleared (either by a successful re-scan with high confidence, or by manual acknowledgement).
- All `[LOW CONFIDENCE]` findings have been reviewed and either confirmed or marked as false positive.
- OCR confidence on the re-processed file is above the threshold.

---

## Prevention

- **Enforce upload quality standards:** Apply a minimum DPI check on document ingest pipelines. Reject or auto-enhance images below 200 DPI before they reach the scan directory.
- **Use searchable PDFs at the source:** Configure document scanners and printers to produce searchable PDF/A output natively rather than image-only PDFs.
- **Alert on patterns:** If a particular source system or storage bucket consistently produces low-confidence results, investigate and fix the upstream digitisation process.
