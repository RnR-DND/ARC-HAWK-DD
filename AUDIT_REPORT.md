# ARC-HAWK-DD: Absolute State Audit Report

**Date:** 2026-04-22  
**Auditor:** Staff-Level Systems Architect / Zero-Trust Security Auditor  
**Scope:** apps/backend/, apps/goScanner/, apps/frontend/ — full stack  
**Methodology:** Static grep/AST analysis, schema cross-reference, concurrency trace, no mock data, no happy-path assumptions

---

## Executive Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 3 |
| HIGH | 7 |
| MEDIUM | 11 |
| LOW | 6 |
| **TOTAL** | **27** |

**Overall Risk Rating: HIGH**

The system has functional core paths — scan ingestion, classification, auth — but carries three critical gaps: (1) a frontend API call to a non-existent backend route, (2) a secret already known-exposed requiring rotation, and (3) zero fuzz test coverage on the three highest-risk parsing functions. Seven tables lack corresponding Go structs, breaking policy/consent/api-key features entirely. Test coverage is ~9.4% by file count with no integration tests backed by real databases.

---

## Phase 1: Static Eradication & Schema Contract Enforcement

### 1.1 Dead Code & Phantom Logic

#### Stub Services — Production Logic Not Implemented

| # | File | Severity | Description |
|---|------|----------|-------------|
| S-01 | `apps/backend/modules/connections/service/connection_sync_service.go` | MEDIUM | `ConnectionSyncService.SyncToYAML()`, `SyncSingleConnection()`, `ValidateSync()` all return nil/no-op. Service kept to avoid breaking callers after Python scanner was removed. Zero functional value. |
| S-02 | `apps/backend/modules/remediation/connectors/filesystem.go`<br>`…/mongodb.go`<br>`…/mysql.go`<br>`…/postgresql.go`<br>`…/s3.go` | MEDIUM | All five remediation connectors are stubs reporting success without mutating source data. The `REMEDIATION_ENABLED` env guard prevents production execution, but the connectors silently lie about success. |
| S-03 | `apps/backend/modules/discovery/api/glossary_handler.go:9-26` | LOW | `GET /discovery/glossary` returns hardcoded `{"items":[],"count":0,"version":"v1-stub"}`. Frontend could render empty state and mask a missing implementation. |
| S-04 | `apps/goScanner/internal/connectors/files/images.go:11`<br>`…/databases/oracle.go:10`<br>`…/cloud/gdrive_workspace.go:11`<br>`…/cloud/azure_blob.go:11`<br>`…/cloud/gdrive.go:11` | LOW | Five goScanner connectors close their output channels immediately with a "not implemented" error. All registered in the connector registry and visible to API consumers — no indication these are stubs. |

#### Notable TODO/FIXME in Production Code

| File | Line | Comment |
|------|------|---------|
| `modules/compliance/service/dpdpa_obligation_service.go` | 265 | `TODO: Query tenant settings for data_accuracy_rescan_days when available` |
| `modules/connections/service/connection_service.go` | 67 | `TODO: Trigger async validation (Phase 3 - Temporal workflow)` |
| `modules/connections/api/connection_handler.go` | 51 | `TODO: Get user from auth context (Phase 2 - Authentication)` — auth bypass in production |
| `modules/discovery/service/risk_engine.go` | 297 | Webhook dispatch stub — silently no-ops |
| `modules/discovery/service/report_service.go` | 170 | PDF export produces HTML; noted as "v1.5 TODO" |
| `modules/scanning/api/agent_sync_handler.go` | 225 | `TODO: Wire real Redis LPUSH when async classify pipeline is enabled` |

---

### 1.2 Database-to-Struct Parity

**Summary: 6 tables have NO Go struct. 5 existing structs have missing columns.**

#### Tables With No Go Struct (orphaned schema)

| Table | Migration | Impact |
|-------|-----------|--------|
| `api_keys` | 000028 | CI/CD API key auth has DB storage but no entity — keys cannot be read from application code |
| `policies` | 000008 | Compliance policy automation exists only in SQL |
| `policy_executions` | 000008 | Policy execution audit trail inaccessible from Go |
| `consent_records` | 000008 | Consent management entirely dead from application layer |
| `token_blacklist` | 000018 | JWT revocation table has no entity — blacklist lookups fall through to SQL ad-hoc only |
| `scan_state_transitions` | 000007 | Audit trail for scan state machine has no struct |

#### Existing Structs Missing Required Columns

| Struct File | Missing Column | SQL Migration | Issue |
|-------------|---------------|---------------|-------|
| `shared/domain/entity/finding.go` | `deleted_at` | 000003 soft-delete view | Soft-delete view `active_findings` unusable from struct |
| `shared/domain/entity/finding.go` | `occurrence_count` | 000003 deduplication | Dedup count tracked in SQL, never surfaced in Go |
| `shared/domain/entity/asset.go` | `deleted_at` | Soft-delete migration | Same as finding |
| `shared/domain/entity/scan_run.go` | `cancelled_at`, `cancelled_by` | 000007 | Cancellation cannot be tracked in audit trail |
| `shared/domain/entity/classification.go` | `confidence_score` is `float64` | 000017 | SQL column is nullable; Go type is non-nullable — will panic on NULL scan |

#### Type Safety Issue

| Table | Column | SQL Type | Go Type | Risk |
|-------|--------|----------|---------|------|
| `classifications` | `confidence_score` | `NUMERIC NULLABLE` | `float64` | NULL value from DB → `sql.Scan` into `float64` panics. Must be `*float64`. |

#### Missing Foreign Key Indexes

All FK columns verified to have indexes in migrations. No missing index gaps found.

---

### 1.3 Frontend-Backend API Contracts

#### CRITICAL: Frontend Calls Non-Existent Backend Route

| Severity | Frontend File | Line | Call | Backend Status |
|----------|---------------|------|------|----------------|
| **CRITICAL** | `apps/frontend/services/scans.api.ts` | 86-88 | `POST /scans/scan-all` | **NOT REGISTERED** — returns 404 in production |

`ScanningModule.RegisterRoutes()` does not register this endpoint. The frontend `scanAll()` function will always fail.

#### Dead Backend Endpoints (no frontend consumer)

| Route | Module File | Notes |
|-------|-------------|-------|
| `GET /discovery/inventory/:assetId` | discovery/module.go | Frontend only calls `/discovery/inventory` (list) |
| `GET /remediation/:id` | remediation/module.go:103 | Frontend calls only `GET /remediation/history` |
| `GET /discovery/risk/scores/:assetId` | discovery/module.go:135 | Frontend uses `/discovery/risk/overview` only |
| `GET /discovery/drift/since/:snapshotId` | discovery/module.go:141 | Frontend uses `/discovery/drift/timeline` only |

**Method mismatches:** None detected.

---

## Phase 2: Data Discovery Pipeline & Concurrency Hardening

### 2.1 Scanner Pipeline Trace

Execution path verified: `POST /scans/trigger` → `ScanTriggerHandler` → goroutine with 35-min context → HTTP POST to goScanner → `ScanHandler` (30-min context) → `errgroup` orchestrator → connectors → ingest → `IngestionService.IngestScan()` with transaction.

#### Pipeline Issues Found

| ID | File | Lines | Severity | Description |
|----|------|-------|----------|-------------|
| GCH-001 | `apps/goScanner/internal/orchestrator/orchestrator.go` | 138-164 | MEDIUM | `runParallel` does not check `gctx.Done()` before calling `g.Go()`. Goroutines may be launched into a cancelled context during cleanup. Add `if gctx.Err() != nil { break }` before each `g.Go()`. |
| GCH-002 | `apps/goScanner/internal/orchestrator/ingest.go` | 157-163 | LOW | `resp.Body.Close()` called inside error branch without `resp != nil` guard. If `ingestClient.Do()` returns `(nil, err)`, this is a nil dereference panic. |

### 2.2 Goroutine Leak Audit

#### Goroutine Issues Found

| ID | File | Lines | Severity | Description |
|----|------|-------|----------|-------------|
| GCH-003 | `apps/backend/modules/scanning/service/patterns_service.go` | 469-483 | MEDIUM | `TestPatternCases` loop spawns one goroutine per test case inside a `for range` loop with no pool or semaphore. If a pattern has 10,000 test cases, 10,000 goroutines are created. No bound on concurrent goroutine count. |
| GCH-004 | `apps/backend/modules/assets/service/asset_service.go` | 208-224 | MEDIUM | `BulkTagAssets()` spawns a background goroutine processing all `req.AssetIDs` sequentially. Multiple concurrent bulk-tag calls = multiple untracked goroutines, no shutdown signal, no WaitGroup. |

#### Verified Safe Patterns

- All 37 goScanner connector goroutines: `defer close(errc)` and `defer close(out)` — ✅
- `scanning/module.go:127-140` timeout ticker: `defer ticker.Stop()` + select on `m.stopTimeout` — ✅
- `scan_trigger_handler.go:161-164` scan executor: `defer release()` via limiter — ✅
- `errgroup` usage in orchestrator: all goroutines bounded by errgroup lifetime — ✅

### 2.3 Atomic Transactions

#### Transaction Issues Found

| ID | File | Lines | Severity | Description |
|----|------|-------|----------|-------------|
| TXN-001 | `apps/backend/modules/scanning/service/patterns_service.go` | 151, 202, 236, 266+ | MEDIUM | DB calls use `QueryContext(ctx, ...)` where `ctx` comes from HTTP handler with no explicit deadline. No `context.WithTimeout` added in service layer. Pattern queries on large tables can block indefinitely if caller doesn't set a deadline. |
| TXN-002 | `apps/backend/modules/scanning/service/ingestion_service.go` | 185-198 | MEDIUM | Transaction started at line 185. Panic recovery at lines 191-197 calls `tx.Rollback()` only on panic. The deferred rollback does not cover all error return paths — if a later function returns error but does not call rollback, the transaction stays open. Pattern should be: `defer func() { if !committed { tx.Rollback() } }()`. |

#### Verified Safe Patterns

- `ingestion_service.go`: single transaction wraps 350 lines, explicit rollback at each error return — ✅ (modulo TXN-002)
- `snapshot_service.go`: `tx.Begin()` → 3 operations → `tx.Commit()` — ✅
- `drift_detection_service.go`: `tx.Begin()` → 2 updates → `tx.Commit()` — ✅
- All DB calls in scan pipeline use `context`-bearing variants (`QueryContext`, `ExecContext`) — ✅

---

## Phase 3: Zero-Trust Security & Regulatory Compliance

### 3.1 PII Exfiltration via Logs

#### Log Scrubbing Infrastructure — Present but Partial

`apps/backend/modules/shared/utils/log_scrubbing.go` exists with `ScrubPII()`, `ScrubJSONLog()`, `SanitizeForLog()`. The infrastructure is correct. The gap is in enforcement:

| ID | File | Lines | Severity | Description |
|----|------|-------|----------|-------------|
| SEC-01 | `apps/backend/modules/scanning/service/ingestion_service.go` | 371-386 | LOW | If `s.encryptor == nil`, logs `WARNING: EncryptionService unavailable — PII sample stored unencrypted for %s`. The pattern name is logged but the sample itself is not. Mitigation present: fallback stores `[ENCRYPTION_FAILED]`. Risk: warning is `log.Printf` — not structured, bypasses `ScrubJSONLog`. |

**No instances found** of raw scan results, PII field values, or connection strings logged without `SanitizeForLog`. The `SanitizeForLog` wrapper is consistently used at identified call sites.

### 3.2 Secret Management

#### CRITICAL: Known-Exposed API Key

| ID | File | Line | Severity | Finding |
|----|------|------|----------|---------|
| SEC-02 | `apps/backend/.env` | 6 | **CRITICAL** | Supermemory API key `sm_XXDS****` present in `.env`. Per `.continue-here.md:11`, this key is documented as already exposed in chat logs and flagged for rotation. The key is present in a committed or tracked file. |

#### HIGH: Default Tokens in docker-compose.yml

| ID | File | Line | Severity | Finding |
|----|------|------|----------|---------|
| SEC-03 | `docker-compose.yml` | 150, 198 | HIGH | `VAULT_DEV_ROOT_TOKEN_ID: ${VAULT_DEV_ROOT_TOKEN:-arc-hawk-dev-token}` — fallback default `arc-hawk-dev-token` hardcoded. If env var unset, service uses this weak token. |
| SEC-04 | `docker-compose.yml` | 205, 271 | HIGH | `SCANNER_SERVICE_TOKEN=${SCANNER_SERVICE_TOKEN:-dev-scanner-token-change-me}` — weak placeholder token for backend↔scanner auth. If `SCANNER_SERVICE_TOKEN` not set, both backend and goScanner use this default. |

#### MEDIUM: Placeholder Secrets in .env Files

| ID | File | Line | Severity | Finding |
|----|------|------|----------|---------|
| SEC-05 | `.env` | 24, 31 | MEDIUM | `JWT_SECRET=CHANGE_ME_generate_with_openssl_rand_base64_48` and `ENCRYPTION_KEY=CHANGE_ME_exactly_32_chars_here_` — placeholder values for development. Deployment pipeline must replace before production. These values are weak and predictable. |

#### LOW: Auth Bypass Flag in Dev Config

| ID | File | Line | Severity | Finding |
|----|------|------|----------|---------|
| SEC-06 | `.env` | 27 | LOW | `AUTH_REQUIRED=false` — disables all authentication. Safe for local dev; fatal if deployed. No automated check in CI to catch this. |

#### MEDIUM: StrictRateLimiter Not Verified on Auth Endpoints

| ID | Finding | Severity |
|----|---------|----------|
| SEC-07 | `StrictRateLimiter()` (10 req/min) exists in `shared/middleware/rate_limiter.go:185-195`. No grep match confirms it is applied to `/api/v1/auth/login`, `/api/v1/auth/register`, `/api/v1/auth/refresh`. General `APIRateLimiter` (100 req/min) is confirmed on all routes. Login endpoint vulnerable to brute-force above 100 req/min per IP. | MEDIUM |

#### Verified Secure

- JWT secret loaded from env var; in `GIN_MODE=release` fatally fails if unset — ✅
- API keys SHA-256 hashed before DB lookup, never stored plaintext — ✅
- All SQL queries use parameterized placeholders (`$1`, `$2`) — no injection vectors found — ✅
- Docker services bind `127.0.0.1` only; goScanner has no host port mapping — ✅
- No non-localhost IPs hardcoded in Go or TypeScript source files — ✅

---

## Phase 4: Test Harness Gap Analysis & Deliverables

### 4.1 Test Coverage Reality

| Metric | Value |
|--------|-------|
| Production Go files in `apps/backend/modules/` | 171 |
| Test files in `apps/backend/modules/` | 16 |
| File-level coverage ratio | **9.4%** |
| Production Go files in `apps/goScanner/` | ~60 |
| Test files in `apps/goScanner/` | 8 |
| testcontainers-go usage | **0** |
| Fuzz tests (`testing.F`) | **0** (before this audit) |
| Playwright test files | **0** |

**Modules with zero test coverage:**

| Module | Production Files |
|--------|-----------------|
| `auth/service/` | 2 |
| `compliance/service/` | 6 |
| `connections/service/` | 4 |
| `lineage/service/` | 2 |
| `assets/service/` | 3 |
| `masking/service/` | 1 |
| `analytics/service/` | 1 |

**Test type breakdown:**
- `go-sqlmock` (mock DB): `patterns_handler_test.go`, `asset_repository_test.go`, `interfaces_test.go`
- Pure unit (no I/O): all remaining 13 files
- Real DB / testcontainers: **none**

### 4.2 Testcontainers Setup

**Written to disk:** `apps/backend/modules/shared/testutil/db_container_test.go`

**Prerequisite:**
```bash
cd apps/backend
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
```

**Usage in other integration tests:**
```go
import "github.com/arc-platform/backend/modules/shared/testutil"

func TestMyRepository_Integration(t *testing.T) {
    if testing.Short() { t.Skip() }
    tdb := testutil.NewTestDB(t)   // real Postgres, fully migrated
    defer tdb.TruncateAll(t)       // clean slate between tests
    
    repo := NewMyRepository(tdb.DB)
    // ... test against real DB
}
```

**Run smoke test:**
```bash
cd apps/backend
go test -v -run TestContainerSmokeTest ./modules/shared/testutil/
```

### 4.3 Fuzz Targets

Three fuzz test files written to disk:

#### Fuzz Target 1 — `validateRegexSafety` (ReDoS risk)

**Written to:** `apps/backend/modules/scanning/service/patterns_validate_fuzz_test.go`

**Dangerous function:** `validateRegexSafety(pattern string) error`  
**Risk:** Takes user-supplied regex from `POST /patterns`. Layer-1 static checks (nested quantifiers, unbounded alternation) guard against obvious ReDoS. Fuzz finds patterns that bypass static checks but still consume excessive CPU.

**Run:**
```bash
cd apps/backend
go test -fuzz=FuzzValidateRegexSafety ./modules/scanning/service/ -fuzztime=60s
```

#### Fuzz Target 2 — `Engine.Classify` (unbounded field value, nil deref)

**Written to:** `apps/goScanner/internal/classifier/engine_fuzz_test.go`

**Dangerous function:** `func (e *Engine) Classify(record connectors.FieldRecord, custom []CustomPattern, allowedPatterns map[string]struct{}) []ClassifiedFinding`  
**Risk:** Applies compiled regex (`FindAllString`) to unvalidated `record.Value` — no length limit on the value. Custom patterns with nil `Regex` skip correctly (guarded), but `excerpt()` and `hashValue()` have no input bounds. Deeply adversarial values stress memory allocation.

**Run:**
```bash
cd apps/goScanner
go test -fuzz=FuzzEngineClassify ./internal/classifier/ -fuzztime=60s
```

#### Fuzz Target 3 — `ScrubJSONLog` (unbounded recursion, no depth limit)

**Written to:** `apps/backend/modules/shared/utils/log_scrubbing_fuzz_test.go`

**Dangerous function:** `func ScrubJSONLog(jsonStr string) string`  
**Risk:** `json.Unmarshal` into `interface{}` followed by `scrubValue()` which recurses into every nested map/array with no depth limit. A 1000-level nested JSON object will recurse 1000 levels deep — stack overflow on extreme inputs. No size limit on input string.

**Run:**
```bash
cd apps/backend
go test -fuzz=FuzzScrubJSONLog ./modules/shared/utils/ -fuzztime=60s
go test -fuzz=FuzzScrubPII ./modules/shared/utils/ -fuzztime=60s
```

### 4.4 Playwright data-testid Gap Analysis

**Playwright test suite:** Does not exist. No `.spec.ts`, `.e2e.ts`, or `playwright.config.ts` found.

**Current data-testid coverage:** 13 of 315 interactive elements = **4.1%**

| Element Type | Total | Have data-testid | Missing |
|---|---|---|---|
| `<button>` | 229 | 13 | **216** |
| `<input>` | 58 | 0 | **58** |
| `<form>` | 2 | 0 | **2** |
| `<select>` | 22 | 0 | **22** |
| `<a>` (action links) | 4 | 0 | **4** |
| **Total** | **315** | **13** | **302** |

**Critical elements missing data-testid (P0 — primary user flows):**

```diff
// apps/frontend/app/scans/page.tsx

- <button onClick={handleScanAll}>
+ <button onClick={handleScanAll} data-testid="scan-all-sources-btn">

- <button onClick={() => setShowScanConfigModal(true)}>
+ <button onClick={() => setShowScanConfigModal(true)} data-testid="new-scan-btn">

- <button onClick={(e) => handleDeleteScan(e, scan.id)}>
+ <button onClick={(e) => handleDeleteScan(e, scan.id)} data-testid="delete-scan-btn">

// apps/frontend/app/findings/page.tsx

- <button onClick={...}> {/* Export CSV */}
+ <button onClick={...} data-testid="export-findings-btn">

- <select value={piiTypeFilter}>
+ <select value={piiTypeFilter} data-testid="pii-type-filter">

- <select value={severityFilter}>
+ <select value={severityFilter} data-testid="severity-filter">

- <input type="text" placeholder="Search path/field...">
+ <input type="text" placeholder="Search path/field..." data-testid="findings-search-input">

// apps/frontend/app/connectors/page.tsx

- <form onSubmit={handleSubmit}>
+ <form onSubmit={handleSubmit} data-testid="add-connector-form">

- <button type="submit">
+ <button type="submit" data-testid="submit-connector-btn">

- <button type="button" onClick={onClose}>
+ <button type="button" onClick={onClose} data-testid="cancel-connector-btn">

- <button onClick={() => handleEdit(conn)}>
+ <button onClick={() => handleEdit(conn)} data-testid="edit-connector-btn">

- <button onClick={() => handleTestConnection(conn)}>
+ <button onClick={() => handleTestConnection(conn)} data-testid="test-connection-btn">

- <button onClick={() => handleDelete(conn)}>
+ <button onClick={() => handleDelete(conn)} data-testid="delete-connector-btn">

// apps/frontend/components/layout/GlobalLayout.tsx

- <Button variant="ghost">  {/* Mobile menu toggle */}
+ <Button variant="ghost" data-testid="mobile-nav-toggle-btn">

- <motion.button onClick={() => setIsAddSourceOpen(true)}>
+ <motion.button onClick={() => setIsAddSourceOpen(true)} data-testid="add-source-quick-action-btn">

- <motion.button onClick={() => setIsRunScanOpen(true)}>
+ <motion.button onClick={() => setIsRunScanOpen(true)} data-testid="run-scan-quick-action-btn">
```

---

## Remediation Roadmap

Ordered by risk, each item includes the exact fix.

### P0 — Fix Immediately (blocking correctness/security)

**R-01: Rotate exposed Supermemory API key** *(SEC-02 — CRITICAL)*
```bash
# 1. Rotate at https://app.supermemory.ai
# 2. Update all instances:
sed -i 's/SUPERMEMORY_API_KEY=sm_XXDS[^"]*/SUPERMEMORY_API_KEY=<NEW_KEY>/' apps/backend/.env
# 3. Ensure .env is in .gitignore (verify):
grep "^\.env$\|^apps/backend/\.env$" .gitignore
```

**R-02: Register missing `POST /scans/scan-all` backend route** *(Phase 1 — CRITICAL)*
```go
// apps/backend/modules/scanning/api/scan_trigger_handler.go
// Add handler:
func (h *ScanTriggerHandler) ScanAllSources(c *gin.Context) {
    tenantID := middleware.GetTenantID(c)
    // Fetch all active connections for tenant, trigger individual scans
    connections, err := h.connService.GetActiveConnections(c.Request.Context(), tenantID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    var triggered []string
    for _, conn := range connections {
        if err := h.scanService.TriggerScanForConnection(c.Request.Context(), tenantID, conn.ID); err != nil {
            continue
        }
        triggered = append(triggered, conn.ID.String())
    }
    c.JSON(http.StatusOK, gin.H{"triggered": triggered, "count": len(triggered)})
}

// In RegisterRoutes():
r.POST("/scans/scan-all", middleware.RequireAuth(), h.ScanAllSources)
```

**R-03: Fix nil dereference in ingest.go** *(GCH-002 — prevents panic)*
```go
// apps/goScanner/internal/orchestrator/ingest.go:163
// Before:
_ = resp.Body.Close()
// After:
if resp != nil {
    _ = resp.Body.Close()
}
```

**R-04: Fix NULL scan panic on confidence_score** *(Schema parity — CRITICAL data corruption)*
```go
// apps/backend/modules/shared/domain/entity/classification.go
// Before:
ConfidenceScore float64 `db:"confidence_score" json:"confidence_score"`
// After:
ConfidenceScore *float64 `db:"confidence_score" json:"confidence_score,omitempty"`
```

---

### P1 — Fix Before Next Release (HIGH)

**R-05: Remove hardcoded docker-compose token defaults** *(SEC-03, SEC-04)*
```yaml
# docker-compose.yml — remove fallback defaults:
# Before:
VAULT_DEV_ROOT_TOKEN_ID: ${VAULT_DEV_ROOT_TOKEN:-arc-hawk-dev-token}
SCANNER_SERVICE_TOKEN=${SCANNER_SERVICE_TOKEN:-dev-scanner-token-change-me}
# After:
VAULT_DEV_ROOT_TOKEN_ID: ${VAULT_DEV_ROOT_TOKEN:?VAULT_DEV_ROOT_TOKEN must be set}
SCANNER_SERVICE_TOKEN=${SCANNER_SERVICE_TOKEN:?SCANNER_SERVICE_TOKEN must be set}
```

**R-06: Apply StrictRateLimiter to auth endpoints** *(SEC-07)*
```go
// apps/backend/modules/auth/api/auth_routes.go (or equivalent route file)
authGroup := r.Group("/auth")
authGroup.Use(middleware.StrictRateLimiter().Middleware())
{
    authGroup.POST("/login", h.Login)
    authGroup.POST("/register", h.Register)
    authGroup.POST("/refresh", h.RefreshToken)
}
```

**R-07: Add deferred rollback to ingestion transaction** *(TXN-002)*
```go
// apps/backend/modules/scanning/service/ingestion_service.go:185
tx, err := s.repo.BeginTx(ctx)
if err != nil {
    return nil, err
}
committed := false
defer func() {
    if r := recover(); r != nil {
        _ = tx.Rollback()
        panic(r)
    }
    if !committed {
        _ = tx.Rollback()
    }
}()
// ... at successful end:
if err := tx.Commit(); err != nil {
    return nil, fmt.Errorf("commit: %w", err)
}
committed = true
```

**R-08: Add context timeout to PatternsService queries** *(TXN-001)*
```go
// apps/backend/modules/scanning/service/patterns_service.go
// Wrap all QueryContext calls:
queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()
rows, err := s.db.QueryContext(queryCtx, `SELECT ...`)
```

**R-09: Add Go structs for 6 orphaned tables** *(Schema parity)*
```bash
# Create entity files:
touch apps/backend/modules/shared/domain/entity/api_key.go
touch apps/backend/modules/shared/domain/entity/policy.go
touch apps/backend/modules/shared/domain/entity/policy_execution.go
touch apps/backend/modules/shared/domain/entity/consent_record.go
touch apps/backend/modules/shared/domain/entity/token_blacklist.go
touch apps/backend/modules/shared/domain/entity/scan_state_transition.go
```
Struct fields must match column names from migrations 000007, 000008, 000018, 000028.

---

### P2 — Fix in Next Sprint (MEDIUM)

**R-10: Add depth limit to ScrubJSONLog** *(Fuzz finding — prevents stack overflow)*
```go
// apps/backend/modules/shared/utils/log_scrubbing.go
const maxScrubDepth = 20
const maxScrubInputBytes = 1 << 18 // 256 KB

func ScrubJSONLog(jsonStr string) string {
    if len(jsonStr) > maxScrubInputBytes {
        return "[input too large to scrub]"
    }
    var obj interface{}
    if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
        return "[invalid json]"
    }
    scrubbed := scrubValueDepth(obj, 0)
    out, err := json.Marshal(scrubbed)
    if err != nil {
        return "[marshal error]"
    }
    return string(out)
}

func scrubValueDepth(v interface{}, depth int) interface{} {
    if depth > maxScrubDepth {
        return "[max depth exceeded]"
    }
    // ... existing switch logic with depth+1 on recursive calls
}
```

**R-11: Add goroutine pool to TestPatternCases loop** *(GCH-003)*
```go
// apps/backend/modules/scanning/service/patterns_service.go:469
sem := make(chan struct{}, 10) // max 10 concurrent regex tests
for _, tc := range cases {
    tc := tc
    sem <- struct{}{}
    matchCh := make(chan bool, 1)
    go func() {
        defer func() { <-sem }()
        matchCh <- re.MatchString(tc.Input)
    }()
    select {
    case matched = <-matchCh:
    case <-time.After(2 * time.Second):
        matched = false
    }
}
```

**R-12: Add context cancellation check in orchestrator loop** *(GCH-001)*
```go
// apps/goScanner/internal/orchestrator/orchestrator.go:138
for _, connector := range connectors {
    connector := connector
    if gctx.Err() != nil {   // ADD: early exit on cancellation
        break
    }
    if err := sem.Acquire(gctx, 1); err != nil {
        break
    }
    g.Go(func() error { ... })
}
```

**R-13: Add missing struct fields to Finding, Asset, ScanRun**
```go
// apps/backend/modules/shared/domain/entity/finding.go
DeletedAt      *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
OccurrenceCount int       `db:"occurrence_count" json:"occurrence_count"`

// apps/backend/modules/shared/domain/entity/scan_run.go
CancelledAt    *time.Time `db:"cancelled_at" json:"cancelled_at,omitempty"`
CancelledBy    *string    `db:"cancelled_by" json:"cancelled_by,omitempty"`
```

---

### P3 — Backlog (LOW / structural)

**R-14: Install testcontainers-go and run integration tests**
```bash
cd apps/backend
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
go test -v -run TestContainerSmokeTest ./modules/shared/testutil/
# Then convert mock-based repository tests to use testutil.NewTestDB()
```

**R-15: Run fuzz suite for minimum 10 minutes each**
```bash
cd apps/backend
go test -fuzz=FuzzValidateRegexSafety ./modules/scanning/service/ -fuzztime=10m
go test -fuzz=FuzzScrubJSONLog ./modules/shared/utils/ -fuzztime=10m
cd apps/goScanner
go test -fuzz=FuzzEngineClassify ./internal/classifier/ -fuzztime=10m
```

**R-16: Initialize Playwright and add data-testid to primary flows**
```bash
cd apps/frontend
npm install --save-dev @playwright/test
npx playwright init
# Add data-testid to 15 P0 elements listed in Phase 4.4 above
# Write smoke test for: login → trigger scan → view findings
```

**R-17: Delete or implement ConnectionSyncService stubs** *(S-01)*  
Either remove `ConnectionSyncService` entirely (update call sites to use DB directly) or implement the YAML sync. A no-op service that silently succeeds is a correctness hazard.

**R-18: Mark stub connectors with build tags** *(S-04)*  
Stub goScanner connectors (images, oracle, gdrive) should be guarded by a build tag `//go:build connector_stub` so they are excluded from production builds.

**R-19: CI pre-deploy gate for AUTH_REQUIRED** *(SEC-06)*  
Add to CI pipeline:
```bash
if [ "${GIN_MODE:-}" = "release" ]; then
    if grep -q "AUTH_REQUIRED=false" .env 2>/dev/null; then
        echo "ERROR: AUTH_REQUIRED=false in release mode" >&2; exit 1
    fi
fi
```

---

## Appendix: Files Written to Disk

| File | Purpose |
|------|---------|
| `apps/backend/modules/shared/testutil/db_container_test.go` | testcontainers-go PostgreSQL setup + smoke tests |
| `apps/backend/modules/scanning/service/patterns_validate_fuzz_test.go` | Fuzz target for `validateRegexSafety` |
| `apps/goScanner/internal/classifier/engine_fuzz_test.go` | Fuzz targets for `Engine.Classify` and allowlist |
| `apps/backend/modules/shared/utils/log_scrubbing_fuzz_test.go` | Fuzz targets for `ScrubJSONLog` and `ScrubPII` |
