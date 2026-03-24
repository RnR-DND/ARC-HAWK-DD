# ARC-Hawk Seamless Scanner Guide

## 🚀 One-Command Scan-to-Dashboard Workflow

The scanner now supports **auto-ingestion** - scan results automatically flow into the backend and appear in the frontend dashboard!

---

## Quick Start

### Option 1: Scan with Auto-Ingest (Built-in)

```bash
cd /Users/prathameshyadav/ARC-Hawk/apps/scanner

# Scan filesystem + auto-ingest to backend
python3 hawk_scanner/main.py fs \
  --connection config/connection.yml \
  --fingerprint /Users/prathameshyadav/ARC-Hawk/fingerprint.yml \
  --ingest-url http://localhost:8080/api/v1/scans/ingest

# Scan PostgreSQL + auto-ingest
python3 hawk_scanner/main.py postgresql \
  --connection config/connection.yml \
  --fingerprint /Users/prathameshyadav/ARC-Hawk/fingerprint.yml \
  --ingest-url http://localhost:8080/api/v1/scans/ingest

# Scan ALL sources (filesystem + PostgreSQL) + auto-ingest
python3 hawk_scanner/main.py all \
  --connection config/connection.yml \
  --fingerprint /Users/prathameshyadav/ARC-Hawk/fingerprint.yml \
  --ingest-url http://localhost:8080/api/v1/scans/ingest
```

### Option 2: Use Unified Scan Script

```bash
cd /Users/prathameshyadav/ARC-Hawk
python3 scripts/automation/unified-scan.py
```

---

## Auto-Ingest Parameters

### Required
- `--ingest-url`: Backend ingestion endpoint
  - Local: `http://localhost:8080/api/v1/scans/ingest`
  - Production: `https://your-domain.com/api/v1/scans/ingest`

### Optional
- `--ingest-retry 3`: Number of retry attempts (default: 3)
- `--ingest-timeout 30`: Timeout in seconds (default: 30)

---

## Complete Seamless Workflow

### Step 1: Start Backend
```bash
cd /Users/prathameshyadav/ARC-Hawk/apps/backend
go run cmd/server/main.go
```

### Step 2: Start Frontend
```bash
cd /Users/prathameshyadav/ARC-Hawk/apps/frontend
npm run dev
```

### Step 3: Run Scan with Auto-Ingest
```bash
cd /Users/prathameshyadav/ARC-Hawk/apps/scanner

python3 hawk_scanner/main.py fs \
  --connection config/connection.yml \
 --fingerprint /Users/prathameshyadav/ARC-Hawk/fingerprint.yml \
  --ingest-url http://localhost:8080/api/v1/scans/ingest
```

### Step 4: View Results
Open browser: `http://localhost:3000`

Results appear **immediately** on the dashboard!

---

## What Happens Automatically

1. **Scanner runs** → Detects PII/secrets
2. **Auto-ingest** → POSTs JSON to backend API
3. **Backend processes** → Multi-signal classification
4. **Database stores** → PostgreSQL + Neo4j (optional)
5. **Frontend refreshes** → Dashboard shows new findings

---

## Configuration (connection.yml)

Current scan targets:

```yaml
sources:
  postgresql:
    local_postgres:
      host: localhost
      port: 5432
      database: arc_platform
      limit_end: 10000  # Scan first 10K rows
    
    aiven_postgres:
      host: pg-c37a611-jainrajat5343-d96d.l.aivencloud.com
      port: 24100
      database: defaultdb
      limit_end: 10000
      
  fs:
    arc_hawk_project:
      path: /Users/prathameshyadav/ARC-Hawk
      exclude_patterns:
        - .git
        - node_modules
        - .next
```

---

## Troubleshooting

### Backend Not Running
```bash
Error: Connection refused to localhost:8080
```

**Fix**: Start backend first
```bash
cd apps/backend && go run cmd/server/main.go
```

### Slow Scans
- **Aiven Database**: Remote database scans are slower due to network latency
- **Solution**: Add `limit_end: 1000` in connection.yml
- **Or**: Scan locally only: `python3 hawk_scanner/main.py fs ...`

### No Auto-Ingest
- **Check**: Verify `--ingest-url` parameter is provided
- **Check**: Backend health: `curl http://localhost:8080/health`

---

## Example Output

```
🚀 Auto-ingesting scan results to http://localhost:8080/api/v1/scans/ingest
⏳ Sending 45 findings to backend...
✅ Successfully ingested scan results!
Response: {
  "scan_run_id": "a1b2c3d4-...",
  "total_findings": 45,
  "total_assets": 12,
  "assets_created": 5
}
```

**Then check frontend at `http://localhost:3000` - findings appear immediately!**

---

## Benefits

✅ **No manual steps** - Scan → Backend → Frontend (all automatic)  
✅ **Real-time updates** - Results appear as soon as scan completes  
✅ **Retry logic** - Automatic retries on network failures  
✅ **Error handling** - Graceful degradation if backend unavailable  
✅ **Production ready** - Works with remote backends  

---

## Next Steps

1. **Schedule scans**: Use cron or systemd timers
   ```cron
   0 */6 * * * cd /path/to/scanner && python3 hawk_scanner/main.py all --ingest-url ...
   ```

2. **CI/CD Integration**: Add to your pipeline
   ```yaml
   - name: Scan for PII
     run: |
       python3 hawk_scanner/main.py fs \
         --connection ...  \
         --ingest-url $BACKEND_URL/api/v1/scans/ingest
   ```

3. **Monitor**: Check dashboard for new findings after each scan

---

**Quick Demo Command:**

```bash
cd /Users/prathameshyadav/ARC-Hawk/apps/scanner && \
python3 hawk_scanner/main.py fs \
  --connection config/connection.yml \
  --fingerprint /Users/prathameshyadav/ARC-Hawk/fingerprint.yml \
  --ingest-url http://localhost:8080/api/v1/scans/ingest \
  --quiet
```

**Then open**: `http://localhost:3000`
