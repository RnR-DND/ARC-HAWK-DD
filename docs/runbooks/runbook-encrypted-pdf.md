# Runbook: Encrypted PDF

## Trigger

A scan job encounters a PDF file that requires a password to open. The scanner cannot extract text from the file and logs a failure.

Activates when the scanner reports `error_code: ENCRYPTED_PDF` or when a finding shows `scan_status: skipped_encrypted` for a `.pdf` asset.

---

## Symptoms

**Logs (scanner worker):**
```
ERROR  hawk_scanner  file=invoice_march.pdf  error="PDF is encrypted or password-protected – skipping"
```

**Temporal UI (`localhost:8088`):** Scan activity shows `ActivityTaskFailed` with reason `encrypted_pdf`.

**Dashboard:** Asset shows status `scan_error` or the file appears in the **Skipped Files** section of the scan summary.

**Metrics:** `hawk_scanner_skipped_files_total{reason="encrypted"}` counter increments.

---

## Automated Response

The scanner catches the decryption error, logs it with the file path and scan job ID, marks the file as `skipped`, and continues processing the remaining files in the scan job. The scan job itself is **not** failed — only the individual file is skipped.

A finding record with `scan_status = 'skipped_encrypted'` is written to the database so the file is surfaced in the UI.

---

## Manual Steps

If the automated skip is not sufficient and you need to scan the encrypted file:

1. **Obtain the decryption password** from the document owner or IT/security team.

2. **Decrypt the file** and place the plaintext copy in a temporary location:
   ```bash
   qpdf --password=<password> --decrypt invoice_march.pdf /tmp/invoice_march_decrypted.pdf
   ```

3. **Trigger a targeted scan** on the decrypted copy via the API:
   ```bash
   curl -X POST http://localhost:8080/api/v1/scan/trigger \
     -H "Content-Type: application/json" \
     -d '{"source_type": "filesystem", "path": "/tmp/invoice_march_decrypted.pdf"}'
   ```

4. **Delete the plaintext copy** after the scan completes to avoid leaving unencrypted data on disk:
   ```bash
   shred -u /tmp/invoice_march_decrypted.pdf
   ```

5. **Mark the original asset** with a note in the dashboard indicating it was scanned via a decrypted copy.

---

## Resolution Criteria

- The decrypted version of the file has been scanned successfully (scan status `completed` in the dashboard).
- Any PII findings from the decrypted scan are visible in the findings list.
- The plaintext temporary file has been securely deleted.

---

## Prevention

- **Document management policy:** Establish a policy that files stored in scanned locations must not be password-protected, or that encryption passwords must be escrowed in a secrets manager (e.g., Vault, AWS Secrets Manager).
- **Pre-scan decryption layer:** For environments where encrypted PDFs are common, add a pre-scan decryption step in the scanner pipeline that fetches passwords from a secrets store.
- **Alert on accumulation:** Set an alert if `hawk_scanner_skipped_files_total{reason="encrypted"}` exceeds a threshold per scan run, which may indicate a systematic encryption policy change.
