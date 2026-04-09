# Runbook: Corrupt or Zero-Byte File

## Trigger

The scanner encounters a file that cannot be read because it is zero bytes, truncated, has an unrecognised format header, or is otherwise corrupt.

Activates when the scanner logs `error_code: CORRUPT_FILE` or `error_code: ZERO_BYTE_FILE`, or when an asset shows `scan_status: skipped_corrupt`.

---

## Symptoms

**Logs (scanner worker):**
```
ERROR  hawk_scanner  file=export_2026.csv  size_bytes=0  error="zero-byte file – skipping"
ERROR  hawk_scanner  file=backup_dump.parquet  error="unexpected EOF reading Parquet footer"
```

**Temporal UI:** Scan activity shows `ActivityTaskFailed` with reason `corrupt_file` or `zero_byte_file`.

**Dashboard:** The file appears in the **Skipped Files** list with the reason `corrupt` or `zero_byte`. The asset record shows `scan_status = 'skipped_corrupt'`.

**Metrics:** `hawk_scanner_skipped_files_total{reason="corrupt"}` and `hawk_scanner_skipped_files_total{reason="zero_byte"}` counters increment.

---

## Automated Response

The scanner catches the read error or detects zero size before attempting to parse the file. It logs the error with the file path, size, and scan job ID, marks the file as `skipped`, and continues with remaining files. The scan job itself is not failed.

A finding record with `scan_status = 'skipped_corrupt'` is written to the database so the file is surfaced in the UI.

---

## Manual Steps

**For zero-byte files:**

1. **Identify the file** from the scan error log or skipped files list.

2. **Check if the file was being written at the time of the scan:**
   ```bash
   stat export_2026.csv  # Check modification time vs scan time
   lsof export_2026.csv  # Check if another process has it open
   ```

3. **Restore from backup** if the file should contain data:
   ```bash
   # Example: restore from S3 backup
   aws s3 cp s3://backups/export_2026.csv /data/exports/export_2026.csv
   ```

4. **Delete the zero-byte file** if it is an orphaned artifact with no backup:
   ```bash
   rm export_2026.csv
   ```

5. **Trigger a re-scan** after restoration or cleanup.

**For corrupt/truncated files:**

1. **Check disk health:**
   ```bash
   df -h      # Check disk space
   dmesg | grep -i "i/o error"  # Check for disk errors
   smartctl -a /dev/sda  # SMART status
   ```

2. **Attempt to repair the file** using format-specific tools:
   - **CSV:** Open in a text editor, identify where the truncation starts, and truncate cleanly at the last complete row.
   - **Parquet:** Use `parquet-tools` to inspect: `parquet-tools inspect corrupt.parquet`
   - **ZIP/archive:** `zip -FF corrupt.zip --out repaired.zip`

3. **Restore from backup** if repair is not feasible.

4. **Mark the asset as permanently skipped** in the dashboard if the file cannot be recovered and does not contain PII (to prevent repeated scan warnings):
   ```bash
   curl -X PUT http://localhost:8080/api/v1/assets/{asset_id} \
     -H "Content-Type: application/json" \
     -d '{"scan_status": "excluded", "exclusion_reason": "corrupt_unrecoverable"}'
   ```

---

## Resolution Criteria

- The corrupt or zero-byte file has been repaired, restored, or deliberately excluded from future scans.
- If restored, a re-scan has completed successfully with `scan_status = 'completed'`.
- The skipped file count for the scan job has been acknowledged in the dashboard.

---

## Prevention

- **Write-to-temp-then-rename pattern:** Ensure all processes that write files to scanned directories use an atomic write pattern (write to a `.tmp` file, then rename). This prevents the scanner from seeing partially written files.
- **File integrity checks on ingest:** Add a post-write checksum verification step in data pipelines before files reach the scan directory.
- **Disk monitoring:** Alert on disk I/O errors and low disk space proactively. Corrupt files are often symptoms of failing hardware.
- **Exclude known transient paths:** Configure the scanner to skip directories used for temporary file staging (e.g., `/tmp`, `*.partial`, `*.tmp`).
