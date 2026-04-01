# ARC-HAWK-DD — April 1 Full Stabilization Plan

> **Scope:** Complete pipeline stabilization — scanner → ingest → classify → display  
> **Source audits:** `full_codebase_audit.md` (54 issues, expanded edition) + Compiled Bug Report (62 bugs)  
> **Status at session start:** 49/54 first-audit issues fixed; 62 second-audit issues unaddressed  

---

## Part A — All Fixes Done (Previous Sessions)

### Session 1 — Functional Bugs Fixed
| ID | Fix | Files Changed |
|----|-----|---------------|
| BUG-01 | match_strings() PAN-only filter removed | system.py |
| BUG-02 | Table name scans→scan_runs in 3 Go files | health_handler.go, dashboard_handler.go, scan_cleanup_service.go |
| BUG-03 | IngestSDKVerified uses scan_id from request payload | ingest_sdk_verified.go |
| BUG-04 | CORS ALLOWED_ORIGINS fixed to localhost:3000 | apps/backend/.env |
| BUG-05 | BACKEND_URL double /api/v1 removed | apps/backend/.env |
| BUG-07 | scans.api.ts double-unwrap removed | scans.api.ts |
| BUG-08+API-01 | connections.api.ts rewritten to use apiClient + username→user | connections.api.ts |
| BUG-09+API-04+DUP-06 | dashboard.api.ts double-unwrap + page_size 50 + removed getFallbackFindings | dashboard.api.ts |
| BUG-10 | python3→python fallback added | scan_orchestration_service.go |
| BUG-11 | sample_text added to scanner findings | system.py |
| BUG-12 | MySQL/MongoDB/S3/Redis/Slack/GCS added to HawkeyeScanInput | ingestion_service.go |

### Session 1 — Security Fixed
| ID | Fix | Files Changed |
|----|-----|---------------|
| SEC-01 | safeJoinPath() added to filesystem.go | filesystem.go |
| SEC-02 | Real AES-GCM encryption in all 4 connectors | filesystem.go, postgresql.go, mysql.go, mongodb.go |
| SEC-03 | Debug log with DB password removed | connection_handler.go |
| SEC-04 | ENCRYPTION_KEY warning comments added | .env, apps/backend/.env |
| SEC-05 | hex.EncodeToString instead of raw bytes in HashToken | jwt_service.go |
| SEC-06 | IDOR fixed — userID from JWT context not request body | remediation_handler.go |
| SEC-07 | sync.Map JWT blacklist added to JWTService | jwt_service.go |
| SEC-08 | Goroutines use context.Background() not gin.Context | connection_handler.go |
| SEC-09 | url.QueryEscape for MongoDB authDB | mongodb.go |
| ARCH-02 | tenant_id type consistency (uuid.UUID, not string) | auth_middleware.go |

### Session 1 — Schema / Frontend Fixed
| ID | Fix | Files Changed |
|----|-----|---------------|
| SCHEMA-07 | ANONYMIZE→ENCRYPT in RemediationEvent type | remediation.api.ts |
| SCHEMA-04/05 | Added 7 missing fields to Finding + metadata to ScanRun | types/index.ts |
| DUP-07 | Connection interface centralized | types/index.ts |
| API-03 (FE) | assetId param added to getRemediationHistory | remediation.api.ts |

### Session 1 — Files Deleted / Moved
| ID | Fix |
|----|-----|
| ARCH-04/DUP-02 | components/Sidebar.tsx deleted (dead code) |
| PLACE-01 | apps/backend/pkg/validation/verhoeff.py deleted |
| PLACE-02 | apps/backend/tests/test_api_snapshot.py deleted |
| PLACE-03 | apps/backend/invalid_findings.go moved → cmd/debug/main.go |
| PLACE-04 | scripts/automation/fingerprint.yml deleted |

### Session 2 — Completed
| ID | Fix | Files Changed |
|----|-----|---------------|
| DUP-01 | get_fingerprint_file() checks config/fingerprint.yml first | system.py |
| DUP-05 | normalize_for_matching() dead function removed | system.py |
| PLACE-05 | .env.production + **/.env.production added to .gitignore | .gitignore |
| SCHEMA-08 | 000010 + 000011 .down.sql created | migrations_versioned/ |
| SCHEMA-02 | Migration 000016 for masked_value column | migrations_versioned/ |
| SCHEMA-06 | Migration 000017 for classification score columns | migrations_versioned/ |
| API-03 (BE) | history_handler.go already existed — verified in place | — |
| ARCH-07 | Dashboard limit 100000 → 1000 | dashboard_handler.go |
| ARCH-05 | .env files documented with usage comments | .env, apps/backend/.env |

---

## Part B — Remaining Fixes (This Plan)

### Phase 1 — Scan Engine (Root Cause of Zero Findings)

#### P1.1 — Integrate ContextAwareScanner into match_strings()
**File:** `apps/scanner/hawk_scanner/internals/system.py:863`  
**Audit:** BUG-001 (P0) + BUG-015 (P1)  
**Problem:** `match_strings()` loads 90+ fingerprint patterns but immediately discards them. Uses only Presidio's `SharedAnalyzerEngine` which detects only PAN. `ContextAwareScanner` exists in `scanner_engine.py` with full regex+entropy+context logic but is never called.  
**Fix:**
```python
from hawk_scanner.internals.scanner_engine import ContextAwareScanner

def match_strings(args, content, source='text'):
    patterns = get_fingerprint_file(args)
    
    # Run ContextAwareScanner with fingerprint patterns (regex-based)
    ctx_scanner = ContextAwareScanner()
    regex_findings = ctx_scanner.scan(content, patterns, source)
    
    # Also run Presidio for AI-powered detection
    from sdk.engine import SharedAnalyzerEngine
    presidio_results = SharedAnalyzerEngine().get_engine().analyze(text=content, entities=None, language="en")
    
    # Merge, dedup by (pattern_name, start, end)
    seen = set()
    all_findings = []
    for f in regex_findings:
        key = (f['pattern_name'], f.get('start', 0), f.get('end', 0))
        if key not in seen:
            seen.add(key)
            all_findings.append(f)
    for r in presidio_results:
        key = (r.entity_type, r.start, r.end)
        if key not in seen:
            seen.add(key)
            all_findings.append({
                "pattern_name": r.entity_type,
                "confidence_score": r.score,  # REAL score, not hardcoded
                "matches": [content[r.start:r.end].strip()],
                "sample_text": content[r.start:r.end].strip(),
                "start": r.start,
                "end": r.end
            })
    return all_findings
```

#### P1.2 — Fix severity casing mismatch
**File:** `apps/scanner/hawk_scanner/internals/system.py:663-687`  
**Audit:** BUG-016 (P1)  
**Problem:** Scanner returns `'CRITICAL'`, `'HIGH'` — backend `dashboard_handler.go:97` checks `"Critical"`, `"High"`. Match never happens → `highRiskFindings` always 0.  
**Fix:** Change `evaluate_severity()` to Title Case: `'Critical'`, `'High'`, `'Medium'`, `'Low'`

#### P1.3 — Fix map_pattern_to_pii_type passthrough → rejected by backend
**File:** `apps/scanner/scanner_api.py:276-291`  
**Audit:** BUG-005 / BUG-005 (P0)  
**Problem:** Presidio entity types like `PERSON`, `LOCATION`, `DATE_TIME` pass through as `PERSON`, `LOCATION` → backend `IsLockedPIIType()` returns false → silently dropped.  
**Fix:** For unmapped Presidio types, map to nearest India type or skip (don't send unknown types that will be rejected):
```python
PRESIDIO_TO_INDIA = {
    'PERSON': None,       # skip — no India equivalent  
    'LOCATION': None,     # skip
    'DATE_TIME': None,    # skip
    'CREDIT_CARD': 'CREDIT_CARD',
    'EMAIL_ADDRESS': 'EMAIL',
    'PHONE_NUMBER': 'IN_PHONE',
    'IN_PAN': 'IN_PAN',
    'IN_AADHAAR': 'IN_AADHAAR',
    'IN_VOTER': 'IN_VOTER',
    'IN_PASSPORT': 'IN_PASSPORT',
}
```

#### P1.4 — Fix hardcoded ml_confidence: 0.95
**File:** `apps/scanner/scanner_api.py:236-238`  
**Audit:** BUG-004 (P0)  
**Fix:** Use actual Presidio `r.score` as `ml_confidence` value when available.

#### P1.5 — Delete dead match_fix.py
**File:** `apps/scanner/hawk_scanner/internals/match_fix.py`  
**Audit:** BUG-013, BUG-049 (P3)  
**Fix:** Delete file — incompatible schema (`match` singular vs `matches` list), dead code risk.

---

### Phase 2 — Backend Ingestion Fixes

#### P2.1 — Fix IngestSDKVerified always creates new ScanRun
**File:** `apps/backend/modules/scanning/service/ingest_sdk_verified.go:31-49`  
**Audit:** BUG-002 (P0)  
**Fix:** Look up existing ScanRun by ID first; only create if not found:
```go
var scanRun *entity.ScanRun
if existing, err := s.repo.GetScanRunByID(ctx, scanRunID); err == nil {
    existing.Status = "completed"
    if err := tx.UpdateScanRun(ctx, existing); err != nil {
        return fmt.Errorf("failed to update scan run: %w", err)
    }
    scanRun = existing
} else {
    scanRun = &entity.ScanRun{ID: scanRunID, Status: "completed", ...}
    if err := tx.CreateScanRun(ctx, scanRun); err != nil {
        return fmt.Errorf("failed to create scan run: %w", err)
    }
}
```

#### P2.2 — Fix IngestScan defer/recover swallows panic silently
**File:** `apps/backend/modules/scanning/service/ingestion_service.go:96-103`  
**Audit:** BUG-012 (P0)  
**Fix:** Use named return value so recover block can set error:
```go
func (s *IngestionService) IngestScan(ctx context.Context, input HawkeyeScanInput) (result *ScanResult, retErr error) {
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            retErr = fmt.Errorf("panic during ingestion: %v", r)
        }
    }()
```

#### P2.3 — Fix IngestScan slice mutation
**File:** `apps/backend/modules/scanning/service/ingestion_service.go:106`  
**Audit:** BUG-008 (P0)  
**Fix:**
```go
allFindings := make([]HawkeyeFinding, 0,
    len(input.FS)+len(input.PostgreSQL)+len(input.MongoDB)+
    len(input.MySQL)+len(input.S3)+len(input.Redis)+len(input.Slack))
allFindings = append(allFindings, input.FS...)
allFindings = append(allFindings, input.PostgreSQL...)
// ... rest of source types
```

#### P2.4 — Fix ClassificationSummaryService unsafe type assertions (crashes server)
**File:** `apps/backend/modules/scanning/service/classification_summary_service.go:48-59`  
**Audit:** BUG-003 (P0)  
**Fix:** GORM returns `int64`, not `int`. Use safe assertion pattern throughout:
```go
total, _ := rawSummary["total"].(int64)
byTypeRaw, _ := rawSummary["by_type"].(map[string]interface{})
// in loop:
dataMap, _ := data.(map[string]interface{})
count, _ := dataMap["count"].(int64)
avgConf, _ := dataMap["avg_confidence"].(float64)
```

#### P2.5 — Replace fmt.Printf debug spam in ingest_sdk_verified.go
**File:** `apps/backend/modules/scanning/service/ingest_sdk_verified.go`  
**Audit:** BUG-050 (P3)  
**Fix:** Remove 5 `fmt.Printf` calls with emoji; replace error logs with `log.Printf`.

---

### Phase 3 — Security / Tenant Isolation

#### P3.1 — Fix TenantID context key collision (CRITICAL security bug)
**Files:** `apps/backend/cmd/server/main.go:284` + `apps/backend/modules/shared/infrastructure/persistence/tenant_helper.go:15`  
**Audit:** BUG-007 (P0), SEC-01  
**Problem:** `main.go` stores `contextKeyTenantID` (typed key); `tenant_helper.go` reads `"tenant_id"` (string key). Go context.Value never matches → all DB queries return ALL tenants' data.  
**Fix — Preferred:** Change `tenant_helper.go` to use `c.Value("tenant_id")` via gin context, since all paths have `*gin.Context`. Create a shared `contextKey` type in a package both can import, OR — simpler — pass tenant via gin context only and read it from there consistently in all handlers.  
**Minimum fix:** In `tenant_helper.go`, import the same `contextKey` type from wherever it's defined in main.go and use it for lookup.

#### P3.2 — Fix CORS origins not split by comma
**File:** `apps/backend/cmd/server/main.go:231-233`  
**Audit:** BUG-006 (P0)  
**Fix:**
```go
import "strings"
AllowOrigins: strings.Split(strings.TrimSpace(allowedOrigins), ","),
```

#### P3.3 — Fix raw SQL LastScanTime bypasses tenant isolation
**File:** `apps/backend/modules/scanning/api/dashboard_handler.go:117-119`  
**Audit:** BUG-020 (P1), SEC-03  
**Fix:** Add tenant_id WHERE clause; extract tenant from gin context:
```go
tenantIDVal, _ := c.Get("tenant_id")
// Use in query: WHERE status = 'completed' AND ($1 = '00000000-...' OR tenant_id = $1)
```

#### P3.4 — Fix dashboard env filter defaulting to PROD (dev shows zero data)
**File:** `apps/backend/modules/scanning/api/dashboard_handler.go:49-52`  
**Audit:** BUG-021 (P1)  
**Fix:** Default to empty string (all environments):
```go
envFilter := c.Query("env") // no default — empty = all
```
Then in the loop, skip the environment check when `envFilter == ""`.

---

### Phase 4 — Connections & Config

#### P4.1 — Fix ProfileName alphanum validator rejects underscores
**File:** `apps/backend/modules/connections/api/connection_handler.go:32`  
**Audit:** BUG-010 (P0)  
**Fix:** Replace `alphanum` with a regex that allows underscores and hyphens:
```go
ProfileName string `json:"profile_name" binding:"required,min=1,max=50"`
// Add manual validation:
matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, req.ProfileName)
if !matched { c.JSON(400, ...) }
```

#### P4.2 — Fix PostgreSQL connector type assertion panic (float64 vs int)
**Files:** `apps/backend/modules/remediation/connectors/postgresql.go:32-36` + mysql.go, mongodb.go  
**Audit:** BUG-011 (P0), SEC-04  
**Fix:** All JSON-unmarshaled numbers are `float64`:
```go
func getConfigString(config map[string]interface{}, key string) string {
    v, _ := config[key].(string)
    return v
}
func getConfigInt(config map[string]interface{}, key string) int {
    if v, ok := config[key].(float64); ok { return int(v) }
    if v, ok := config[key].(int); ok { return v }
    return 0
}
port := getConfigInt(config, "port")
host := getConfigString(config, "host")
```

#### P4.3 — Document ENCRYPTION_KEY in .env.example
**File:** `apps/backend/.env.example` (create if missing)  
**Audit:** BUG-029 (P1)  
**Fix:** Add `ENCRYPTION_KEY=` entry with generation instruction:
```
# Required: 32-byte or 64-char hex key. Generate: openssl rand -hex 32
ENCRYPTION_KEY=
```

---

### Phase 5 — Frontend Dashboard Fixes

#### P5.1 — Fix dashboard metrics response wrapper mismatch
**File:** `apps/backend/modules/scanning/api/dashboard_handler.go:122-125`  
**Audit:** BUG-014 (P0)  
**Problem:** Backend returns `{"data": metrics}` but frontend reads `metricsRes.total_pii` (skips `.data` level).  
**Fix (preferred — backend):** Change backend to return flat response, consistent with other endpoints:
```go
c.JSON(http.StatusOK, metrics)  // remove gin.H{"data": metrics} wrapper
```

#### P5.2 — Fix assetsHit counting PII types not assets
**File:** `apps/frontend/services/dashboard.api.ts:45`  
**Audit:** BUG-030 (P1)  
**Fix:** Backend already returns `AssetsHit int` in DashboardMetrics — use that directly:
```typescript
assetsHit: metricsRes.assets_hit || 0,
```

#### P5.3 — Fix confidence score fabrication
**File:** `apps/frontend/services/dashboard.api.ts:66`  
**Audit:** BUG-034 (P2)  
**Fix:** Don't fabricate 0.85 default — use 0 or null:
```typescript
confidence: f.confidence_score || f.confidence || 0,
```

---

### Phase 6 — Scanner Code Quality / Safety

#### P6.1 — Fix enhance_and_ocr fixed temp filename (race condition)
**File:** `apps/scanner/hawk_scanner/internals/system.py:698`  
**Audit:** BUG-024 (P1)  
**Fix:** Use unique temp file:
```python
import tempfile
tmp = tempfile.NamedTemporaryFile(suffix='.png', delete=False)
enhanced_image.save(tmp.name)
```

#### P6.2 — Fix find_pii_in_archive double-delete temp dir
**File:** `apps/scanner/hawk_scanner/internals/system.py:544-567`  
**Audit:** BUG-025 (P1)  
**Fix:** Remove explicit `shutil.rmtree(tmp_dir)` — the `with tempfile.TemporaryDirectory()` context manager handles it.

#### P6.3 — Fix tarfile.extractall path traversal (CVE-2007-4559)
**File:** `apps/scanner/hawk_scanner/internals/system.py:554`  
**Audit:** BUG-026 (P1), SEC-02  
**Fix:**
```python
tar.extractall(tmp_dir, filter='data')  # Python 3.12+ safe
# For older Python: manual member check before extractall
```

#### P6.4 — Fix pwd/grp Unix-only imports crash on Windows
**File:** `apps/scanner/hawk_scanner/internals/system.py:573-574`  
**Audit:** BUG-027 (P1)  
**Fix:** Wrap in try/except:
```python
try:
    import pwd, grp
    owner = pwd.getpwuid(stat.st_uid).pw_name
    group = grp.getgrgid(stat.st_gid).gr_name
except (ImportError, KeyError):
    owner, group = "", ""  # Windows or permission error
```

#### P6.5 — Fix Excel formatter treating None as string "None"
**File:** `apps/scanner/hawk_scanner/internals/system.py:534`  
**Audit:** BUG-032 (P2)  
**Fix:**
```python
if cell.value is not None:
    content += str(cell.value) + '\n'
```

#### P6.6 — Fix process_frame args.debug without null check
**File:** `apps/scanner/hawk_scanner/internals/system.py:444`  
**Audit:** BUG-033 (P2)  
**Fix:** `if args and hasattr(args, 'debug') and args.debug:`

---

### Phase 7 — Logging & Code Quality

#### P7.1 — Fix println() in connection_handler.go
**File:** `apps/backend/modules/connections/api/connection_handler.go:62, 103`  
**Audit:** BUG-051 (P3)  
**Fix:** `log.Printf("⚠️ sync error: %v", err)`

#### P7.2 — Fix ReadTimeout too short for scan operations
**File:** `apps/backend/cmd/server/main.go:391`  
**Audit:** BUG-031 (P2)  
**Fix:** Increase to 120s or use per-handler context timeout instead of global server timeout.

---

### Phase 8 — Skipped (Architectural / Too Complex)

These are real bugs but require larger refactoring. Document for future milestones:

| ID | Description | Why Skipped |
|----|-------------|-------------|
| BUG-009 | Scanner thread no cleanup on crash | Needs task queue (Celery/RQ) refactor |
| BUG-028 | active_scans in-memory lost on restart | Needs Redis or DB-backed scan state |
| BUG-017 | EnrichmentService.HistoricalCount hardcoded 0 | fplearning module not wired — full feature |
| BUG-022 | Mutex held during DB query in ScanOrchestration | Correctness vs performance trade-off |
| BUG-041 | ProcessPoolExecutor args not picklable | Video OCR edge case, not in main pipeline |
| ARCH-01 | Three competing scan trigger paths | Requires unified scan service |
| ARCH-03 | ContextAwareScanner never in main path | Addressed via P1.1 integration |
| ARCH-06 | WebSocket module not wired | Requires full websocket implementation |
| PLACE-06 | scanner_api.py at wrong level | Would break existing imports |

---

## Part C — Bug Patterns Discovered

After analyzing all 62 bugs, 5 systemic patterns emerge. **Each pattern suggests scanning additional files for the same issue:**

### Pattern 1: Unsafe Go Type Assertions on Interface Maps
**Found in:** `classification_summary_service.go`, `postgresql.go`, `mysql.go`, `mongodb.go`  
**Pattern:** `value.(int)` or `value.(string)` on `map[string]interface{}` values from JSON/DB without `ok` check  
**Danger:** Guaranteed runtime panic when type doesn't match (JSON numbers are always `float64`, DB COUNT is `int64`)  
**Scan for:** All `.go` files containing `\.\(int\)` or `\.\(string\)` or `\.\(map\[` — especially in handlers and services that receive external data  
**Files to check:**
- `apps/backend/modules/scanning/service/*.go`
- `apps/backend/modules/connections/service/*.go`
- `apps/backend/modules/remediation/connectors/*.go`

### Pattern 2: Go Context Key Type Collision
**Found in:** `main.go` (typed key) vs `tenant_helper.go` (string key)  
**Pattern:** Value stored in context with typed key, retrieved with string key (or vice versa) — always returns nil  
**Danger:** Silent authorization bypass (all tenants see all data)  
**Scan for:** All `context.WithValue` and `ctx.Value()` calls — verify key types match at storage and retrieval sites  
**Files to check:**
- `apps/backend/cmd/server/main.go` — all context.WithValue calls
- `apps/backend/modules/shared/infrastructure/persistence/tenant_helper.go`
- Any middleware that stores values in context

### Pattern 3: Silent Error Swallowing in defer/recover
**Found in:** `ingestion_service.go:96-103`  
**Pattern:** `recover()` block logs panic but function returns `(nil, nil)` — caller sees "success"  
**Danger:** Data loss with 200 OK response  
**Scan for:** All `defer func()` blocks containing `recover()` — verify they set named return error  
**Files to check:**
- `apps/backend/modules/scanning/service/*.go`
- `apps/backend/modules/remediation/service/*.go`

### Pattern 4: Response Wrapper Inconsistency (Frontend/Backend Contract)
**Found in:** `dashboard_handler.go` (`{"data": ...}`) vs most other endpoints (flat response)  
**Pattern:** Different endpoints use different JSON envelope shapes; frontend has to guess  
**Danger:** Double-unwrap bugs, always-zero metrics  
**Scan for:** All `c.JSON(200, gin.H{"data":` patterns in backend → verify frontend handles the wrapper  
**Files to check:**
- All `*_handler.go` files in `apps/backend/modules/*/api/`
- Corresponding `*.api.ts` files in `apps/frontend/services/`

### Pattern 5: Hardcoded Defaults That Break Dev/Prod Parity
**Found in:** Dashboard env filter `"PROD"` (BUG-021), temp file names (BUG-024), CORS single string (BUG-006)  
**Pattern:** Defaults that work in production silently break dev workflow  
**Danger:** Developers see zero data locally, can't debug  
**Scan for:** All `DefaultQuery("...", "PROD")` or similar hard-coded environment-specific defaults  
**Files to check:**
- All query-param defaults in handlers
- All `getEnv("...", "hardcoded-prod-value")` calls in main.go

---

## Part D — Complete File Impact Map

### Backend (Go)
| File | Changes |
|------|---------|
| `apps/backend/cmd/server/main.go` | P3.1 (context key), P3.2 (CORS split), P7.2 (timeout) |
| `apps/backend/modules/scanning/service/ingest_sdk_verified.go` | P2.1 (upsert ScanRun), P2.5 (remove fmt.Printf) |
| `apps/backend/modules/scanning/service/ingestion_service.go` | P2.2 (named return recover), P2.3 (slice mutation) |
| `apps/backend/modules/scanning/service/classification_summary_service.go` | P2.4 (safe type assertions) |
| `apps/backend/modules/scanning/api/dashboard_handler.go` | P3.3 (tenant SQL), P3.4 (env filter), P5.1 (flat response) |
| `apps/backend/modules/shared/infrastructure/persistence/tenant_helper.go` | P3.1 (context key) |
| `apps/backend/modules/connections/api/connection_handler.go` | P4.1 (alphanum), P7.1 (println) |
| `apps/backend/modules/remediation/connectors/postgresql.go` | P4.2 (float64 type assertion) |
| `apps/backend/modules/remediation/connectors/mysql.go` | P4.2 (same pattern) |
| `apps/backend/modules/remediation/connectors/mongodb.go` | P4.2 (same pattern) |
| `apps/backend/.env.example` | P4.3 (document ENCRYPTION_KEY) |

### Scanner (Python)
| File | Changes |
|------|---------|
| `apps/scanner/hawk_scanner/internals/system.py` | P1.1 (ContextAwareScanner), P1.2 (severity casing), P6.1-P6.6 |
| `apps/scanner/scanner_api.py` | P1.3 (pii_type mapping), P1.4 (real confidence) |
| `apps/scanner/hawk_scanner/internals/match_fix.py` | DELETE (P1.5) |

### Frontend (TypeScript)
| File | Changes |
|------|---------|
| `apps/frontend/services/dashboard.api.ts` | P5.1 (wrapper), P5.2 (assetsHit), P5.3 (confidence) |

---

## Part E — Optimal Execution Strategy

Based on dependency analysis, execute in this order (each phase unblocks the next):

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6+7
 (scan)    (ingest)  (security)  (config)  (frontend)  (cleanup)
```

**After each phase, verify:**
1. After Phase 1: Run scanner CLI against test data, confirm >1 PII type detected
2. After Phase 2: `go build ./...` must pass; POST to `/scans/ingest-verified` returns 200 with data in DB
3. After Phase 3: Multi-tenant queries return only own data; CORS works with comma-separated origins
4. After Phase 4: Can create connection named `postgresql_test_db`; no panic on remediation
5. After Phase 5: Dashboard shows actual counts, not zeros
6. After Phase 6+7: No Python warnings on Windows; no race conditions in concurrent scans

---

## Part F — Verification Checklist (End-to-End)

```
□ go build ./... — zero errors
□ scanner_api.py starts without import errors (including Windows)
□ POST /api/v1/connections — create a filesystem connection named "test_fs"
□ POST /api/v1/scans/trigger — trigger scan on directory with known PII
□ GET /api/v1/findings — returns non-empty list with multiple PII types (not just PAN)
□ GET /api/v1/dashboard/metrics — returns non-zero total_pii, high_risk_findings
□ GET /api/v1/classification/summary — returns without 500 error
□ GET /api/v1/remediation/history — returns 200 (not 404)
□ Dashboard UI shows correct asset count (not PII type count)
□ High-risk count > 0 when Critical/High findings exist
□ Second scan on same source updates existing ScanRun (not creates duplicate)
□ CORS works: frontend on localhost:3000 can call backend on localhost:8080
```
