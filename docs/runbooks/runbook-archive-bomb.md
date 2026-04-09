# Runbook: Archive Bomb

## Trigger

The scanner encounters a compressed archive (ZIP, tar.gz, etc.) whose decompressed size exceeds the configured safety threshold (default: 1 GB or 1000x compression ratio). This is a defence against zip bomb / decompression bomb attacks that would exhaust disk space or memory.

Activates when the scanner logs `error_code: ARCHIVE_BOMB` or when an asset shows `scan_status: rejected_archive_bomb`.

---

## Symptoms

**Logs (scanner worker):**
```
ERROR  hawk_scanner  file=dataset.zip  compressed_bytes=1024  estimated_uncompressed_bytes=10737418240  ratio=10485760.0  error="archive bomb detected – rejecting (ratio exceeds 1000x)"
```

**Dashboard:** The archive appears in the **Rejected Files** list with reason `archive_bomb`. The asset record shows `scan_status = 'rejected_archive_bomb'`.

**Metrics:** `hawk_scanner_rejected_files_total{reason="archive_bomb"}` counter increments.

**No** disk space is consumed because the archive is never fully decompressed.

---

## Automated Response

The scanner reads archive metadata (stored headers) to estimate the uncompressed size before decompressing any content. If the estimated size exceeds the threshold or the compression ratio exceeds the limit, the archive is immediately rejected and logged. The archive is **not** decompressed and **not** deleted — it is left in place.

The scan job continues with remaining files.

---

## Manual Steps

If you believe the archive is legitimate and must be scanned:

1. **Verify the archive** is from a trusted source and inspect its contents header:
   ```bash
   # Check ZIP contents without extracting
   unzip -l dataset.zip | tail -5

   # Check total uncompressed size
   unzip -l dataset.zip | awk '{sum += $1} END {print sum " bytes uncompressed"}'
   ```

2. **Verify disk space** is sufficient to handle full decompression:
   ```bash
   df -h /data/scan-staging
   ```

3. **Extract to a temporary location manually** (not to a scanned directory) to avoid triggering another archive bomb rejection:
   ```bash
   mkdir -p /tmp/scan-extract
   unzip dataset.zip -d /tmp/scan-extract/
   ```

4. **Trigger a filesystem scan** on the extracted directory:
   ```bash
   curl -X POST http://localhost:8080/api/v1/scan/trigger \
     -H "Content-Type: application/json" \
     -d '{"source_type": "filesystem", "path": "/tmp/scan-extract/"}'
   ```

5. **Clean up** the extraction directory after scanning:
   ```bash
   rm -rf /tmp/scan-extract/
   ```

6. If the archive is legitimate but large and this will be a recurring situation, **raise the scanner threshold** in `hawk_scanner/config.yml`:
   ```yaml
   archive:
     max_uncompressed_bytes: 5368709120  # 5 GB (increased from 1 GB)
     max_compression_ratio: 5000         # (increased from 1000x)
   ```
   Apply the config change and restart the scanner worker.

**If the archive is malicious or from an unknown source:**

1. **Quarantine** the file immediately by moving it out of the scan path to an isolated location.
2. **Notify your security team** with the file path, source, and upload timestamp.
3. **Review access logs** for the storage system to identify who uploaded the file.
4. **Do not delete** the file until the security team has completed their investigation.

---

## Resolution Criteria

- The archive has either been:
  - Legitimately scanned (extracted to a temp location, scanned, temp cleaned up), or
  - Confirmed as malicious and quarantined by the security team.
- The asset's `scan_status` has been updated to `completed` or `excluded`.

---

## Prevention

- **Input validation on upload endpoints:** Reject compressed files above a size threshold at the point of upload (before they reach the scan directory). Validate the `Content-Length` header and reject suspiciously high compression ratios.
- **File type allowlisting:** If your environment does not need ZIP files in data pipelines, exclude them from scanned directories.
- **Staged extraction:** If large archives are routine, implement a pre-processing stage that extracts archives in a resource-limited sandbox (e.g., a container with cgroup memory/disk limits) before making the contents available to the scanner.
- **Alerting on rejection events:** Set a Prometheus alert on `hawk_scanner_rejected_files_total{reason="archive_bomb"} > 0` to catch any occurrence immediately.
