# 📋 architecture/scanning-sop.md - Scanning SOP

**Version:** 1.0.0
**Date:** 2026-01-22
**Status:** Draft

---

## 🎯 Purpose

This SOP defines the scanning workflow for PII discovery across all supported data sources. The Scanner SDK is the **SOLE AUTHORITY** for PII validation.

---

## 📊 Input

### Scan Configuration

```json
{
  "name": "string",
  "sources": ["fs", "postgresql", "mysql", "mongodb", "s3", "gcs", "redis", "slack"],
  "pii_types": ["IN_AADHAAR", "IN_PAN", "EMAIL", "PHONE", ...],
  "execution_mode": "sequential|parallel",
  "connection_profile": "string",
  "options": {
    "quick_exit": false,
    "max_matches": 5
  }
}
```

### Connection Profiles

**Each source requires UNIQUE connection parameters (see connection.yml.sample):**

| Source | Required Parameters |
|--------|---------------------|
| **Filesystem** | `path`, `exclude_patterns[]` |
| **PostgreSQL** | `host`, `port`, `user`, `password`, `database`, `tables[]` |
| **MySQL** | `host`, `port`, `user`, `password`, `database`, `tables[]`, `exclude_columns[]` |
| **MongoDB** | `uri` OR `host`, `port`, `username`, `password`, `database`, `collections[]` |
| **AWS S3** | `access_key`, `secret_key`, `bucket_name`, `exclude_patterns[]` |
| **Google GCS** | `credentials_file`, `bucket_name`, `exclude_patterns[]` |
| **Redis** | `host`, `password` |
| **Slack** | `token`, `channel_types`, `channel_ids[]`, `limit_mins` |

---

## 🔄 Workflow Steps

### Step 1: Initialize Scanner

```python
from hawk_scanner.internals import system

scanner = system.Scanner(
    config_path="config/connection.yml",
    fingerprint_path="config/fingerprint.yml"
)
```

### Step 2: Load Connection Profile

Load the appropriate connection profile based on source type:

```python
profile = scanner.load_profile(source_type="fs", profile_name="fs_example")
# OR
profile = scanner.load_profile(source_type="postgresql", profile_name="postgresql_example")
```

### Step 3: Execute Scan

```python
results = scanner.scan(
    source_type="fs",
    profile=profile,
    pii_types=["IN_AADHAAR", "IN_PAN", "EMAIL"],
    options={"max_matches": 5}
)
```

### Step 4: Validate Results (CRITICAL)

All findings MUST be validated by the Scanner SDK:

```python
for finding in results:
    if finding.is_validated:
        # Validated by SDK (Verhoeff, Luhn, etc.)
        pass
    else:
        # Reject - only SDK can validate
        raise Exception("Unvalidated finding rejected")
```

### Step 5: Format Output

```json
{
  "data_source": "fs",
  "host": "localhost",
  "file_path": "/path/to/file.txt",
  "pattern_name": "IN_AADHAAR",
  "matches": ["hashed_value"],
  "sample_text": "Context snippet...",
  "severity": "Critical",
  "profile": "fs_example",
  "verified": true,
  "verification_method": "Verhoeff"
}
```

---

## ⚠️ Error Handling

| Error Type | Condition | Action |
|------------|-----------|--------|
| `ConnectionError` | Cannot connect to source | Log error, continue to next source |
| `ValidationError` | Finding not validated by SDK | Reject finding, log warning |
| `TimeoutError` | Scan exceeds timeout | Abort scan, log partial results |
| `PermissionError` | Insufficient permissions | Log error, skip asset |

---

## 🔧 Retry Logic

- **Max Retries:** 3
- **Backoff:** Exponential (2^n seconds)
- **Retryable Errors:** ConnectionError, TimeoutError
- **Non-Retryable Errors:** PermissionError, ValidationError

---

## 📈 Metrics

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Scan Throughput | 200-350 files/sec | < 100 files/sec |
| Validation Speed | 1,000 findings/sec | < 500 findings/sec |
| False Positive Rate | 0% | > 0.1% |

---

## 🔗 Related Documents

- `connection.yml.sample` - Connection schemas
- `fingerprint.yml` - PII detection patterns
- `scanning-sop.md` - This SOP
- `ingestion-sop.md` - Data ingestion workflow

---

*This SOP is the source of truth for scanning operations. Update before modifying scanner code.*
