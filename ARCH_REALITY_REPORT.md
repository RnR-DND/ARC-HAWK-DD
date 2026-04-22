# ARC-HAWK-DD Architecture Reality Report
**Audit Date:** 2026-04-23  
**Auditor:** Principal Systems Architect / Lead Reliability Engineer  
**Method:** Zero-assumption read of actual source files — no hallucination, every claim has a file:line citation.

---

## 1. Architecture Reality Check

### What Actually Exists vs. Audit Assumptions

| Component | Assumed | Reality |
|-----------|---------|---------|
| **Python** | Possible | **NOT PRESENT** — zero `.py` files in `apps/`. The `.agent/hive/` tool scripts are Python but are not part of the platform runtime. |
| **Celery** | Possible | **NOT PRESENT in runtime** — Celery YAML manifests exist in `k8s/celery/` and `helm/arc-hawk-dd/templates/celery-*.yaml` but there is no Python Celery worker code anywhere in `apps/`. These manifests appear to be legacy infrastructure definitions with no backing implementation. |
| **Temporal** | Possible | **PRESENT** — `go.temporal.io/sdk v1.25.0` in `apps/backend/go.mod`. Worker at `apps/backend/modules/scanning/worker/temporal_worker.go`. Workflows at `apps/backend/modules/scanning/workflows/scan_workflows.go`. Disabled by default (`TEMPORAL_ENABLED` env var). |
| **Presidio (Python service)** | Possible | **PRESENT as external container** — `mcr.microsoft.com/presidio-analyzer:latest` in `docker-compose.yml`. The Go scanner calls it at `PRESIDIO_URL` via HTTP. NOT embedded Python — it is a sidecar. |
| **Vault** | Possible | **PRESENT but OPTIONAL** — `vault_client.go` wraps HashiCorp Vault KV v2. Disabled by default (`VAULT_ENABLED=false`). When disabled, credentials fall back to AES-256 in PostgreSQL. |
| **Neo4j** | Possible | **PRESENT** — `neo4j:5.15-community` in `docker-compose.yml`. `neo4j-go-driver/v5 v5.28.4` in `go.mod`. Full repository at `apps/backend/modules/shared/infrastructure/persistence/neo4j_repository.go`. |
| **Redis** | Possible | **PRESENT** — `redis/go-redis/v9` in `go.mod`. Used for streaming window checkpoints in Temporal activities. |
| **Oracle Connector** | Possible | **BUILD-TAG STUB** — `apps/goScanner/internal/connectors/databases/oracle.go` is guarded by `//go:build connector_stub` and returns an error on all calls. |
| **Go Scanner** | Possible | **PRESENT** — `apps/goScanner/` is a fully independent Go service with 36 connectors, a classifier engine, Presidio bridge, and HTTP API. |
| **Agent** | Possible | **PRESENT** — `apps/agent/` is a separate edge scanner agent in Go with local queue buffering. |

### Runtime Service Topology (from docker-compose.yml)

```
Frontend (3000) → Backend (8080) → [PostgreSQL, Neo4j, Temporal, Vault, Go-Scanner, Presidio]
Go-Scanner (8001) → Presidio (3000/3001 internal)
Temporal (7233) → PostgreSQL
Prometheus (9090) → all services
```

---

## 2. Phase 1 — Ingestion, Connectors & Secret Management

### F-01 · CRITICAL — Scanner IPC Has No Authentication

**File:** `apps/backend/modules/scanning/api/scan_trigger_handler.go:257-263`  
**Also:** `apps/goScanner/cmd/scanner/main.go:28-31`

The backend dispatches scan jobs to the Go scanner over plain HTTP with no authentication token. The scanner registers no authentication middleware. Any process that can reach port 8001 can trigger an arbitrary scan job.

```go
// apps/backend/modules/scanning/api/scan_trigger_handler.go:257-264
reqHttp, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
// ...
reqHttp.Header.Set("Content-Type", "application/json")
// ← No Authorization header, no service token, no HMAC
resp, err := client.Do(reqHttp)
```

```go
// apps/goScanner/cmd/scanner/main.go:28-31
r := gin.Default()
r.POST("/scan", scannerapi.ScanHandler)   // ← no auth middleware
r.GET("/health", scannerapi.HealthHandler)
r.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

In `docker-compose.yml` the scanner binds `0.0.0.0:8001` (no `127.0.0.1:` prefix on the host port mapping), making port 8001 reachable externally in the default configuration.

**Fix:**
```go
// Scanner: add middleware in main.go
serviceToken := os.Getenv("SCANNER_SERVICE_TOKEN")
if serviceToken != "" {
    r.Use(func(c *gin.Context) {
        if c.GetHeader("X-Service-Token") != serviceToken {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }
        c.Next()
    })
}

// Backend: set header in scan_trigger_handler.go
reqHttp.Header.Set("X-Service-Token", os.Getenv("SCANNER_SERVICE_TOKEN"))
```

---

### F-02 · HIGH — Vault Is Disabled by Default; All Production Deployments Use Env-Var Credentials

**File:** `apps/backend/modules/shared/infrastructure/vault/vault_client.go:35-38`  
**Also:** `docker-compose.yml` (backend service: `VAULT_ENABLED=${VAULT_ENABLED:-false}`)

`VAULT_ENABLED` defaults to `false`. When disabled, all connection credentials are AES-256-GCM encrypted in PostgreSQL using `ENCRYPTION_KEY` from the environment. The Vault integration is correctly implemented (KV v2, write-only on creation, read-only on scan dispatch) but is not the default path.

```go
// vault_client.go:35-38
enabled := strings.EqualFold(os.Getenv("VAULT_ENABLED"), "true")
if !enabled {
    log.Println("Vault integration disabled (VAULT_ENABLED != true)")
    return &Client{enabled: false}, nil
}
```

The secondary risk: `VAULT_TOKEN` is passed via an environment variable (`VAULT_DEV_ROOT_TOKEN`). In dev mode this is a root token, not a narrowly-scoped AppRole. There is no token renewal or lease management.

**Fix:** Default `VAULT_ENABLED=true` for production Helm chart. Replace `VAULT_DEV_ROOT_TOKEN` path with AppRole auth and implement lease renewal via `go.temporal.io/sdk` background goroutine or a dedicated sidecar.

---

### F-03 · HIGH — Scanner Postgres Connector Forces `sslmode=disable`

**File:** `apps/goScanner/internal/connectors/databases/postgres.go:26`

When the Go scanner connects to a scanned PostgreSQL target, it hardcodes `sslmode=disable` regardless of configuration. This means all credential handshake and scanned PII data travels in plaintext over the wire.

```go
// postgres.go:26
dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
    host, user, pass, dbname, port)
```

**Fix:**
```go
sslMode := cfgString(config, "sslmode")
if sslMode == "" {
    sslMode = "require"
}
dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
    host, user, pass, dbname, port, sslMode)
```

---

### F-04 · MEDIUM — Connector Row Sampling Hardcoded at LIMIT 10000

**Files:**  
- `apps/goScanner/internal/connectors/databases/postgres.go:79`  
- `apps/goScanner/internal/connectors/databases/mysql.go:65`  
- `apps/goScanner/internal/connectors/databases/sqlite.go:60`  
- `apps/goScanner/internal/connectors/warehouses/redshift.go:70`  
- `apps/goScanner/internal/connectors/warehouses/bigquery.go:110`  
- `apps/goScanner/internal/connectors/warehouses/snowflake.go:82`  
- `apps/goScanner/internal/connectors/saas/salesforce.go:89`

Every DB connector issues `SELECT * FROM "schema"."table" LIMIT 10000` with no configurable limit, no cursor/pagination, and no column-level selection. On a table with 500 wide columns and 10,000 rows this buffers 5M cells in memory simultaneously. On a table with fewer than 10,000 rows PII in rows 10,001+ is silently skipped.

```go
// postgres.go:79
query := fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT 10000`, t.schema, t.name)
```

**Fix:** Expose `scan_sample_size` per-source in `ScanConfig`, default 1000, use keyset pagination for structured sources.

---

### F-05 · MEDIUM — DeleteConnection Has Partial-Failure Window (Vault-Then-Postgres)

**File:** `apps/backend/modules/connections/service/connection_service.go:116-127`

When Vault is enabled, `DeleteConnection` deletes from Vault first, then deletes from PostgreSQL. If the PostgreSQL delete fails after the Vault delete succeeds, the metadata row remains in PostgreSQL pointing to a Vault secret that no longer exists. Subsequent scan attempts will fail with a confusing "credentials not found in Vault" error.

```go
// connection_service.go:116-127
if s.vault != nil && s.vault.IsEnabled() {
    if conn, err := s.pgRepo.GetConnection(ctx, id); err == nil {
        if vErr := s.vault.DeleteConnectionSecret(conn.SourceType, conn.ProfileName); vErr != nil {
            log.Printf("WARN: Vault delete failed for %s/%s: %v", ...)
        }
    }
}
return s.pgRepo.DeleteConnection(ctx, id)  // ← if this fails, Vault is already empty
```

**Fix:** Reverse the order — delete from PostgreSQL first (inside a transaction), then delete from Vault only on successful commit. Log a WARN if the Vault delete fails but do not surface it to the caller.

---

## 3. Phase 2 — Classification Engine

### F-06 · HIGH — `IN_BANK_ACCOUNT` Pattern Matches Any 9-18 Digit String (Extreme False Positives)

**File:** `apps/goScanner/internal/classifier/patterns.go:52`

The Bank Account pattern fires on any 9–18 digit sequence. This matches timestamps, epoch seconds, ZIP codes padded to 9 digits, internal IDs, UAN numbers (already covered by `IN_EPF_UAN`), and ESIC numbers (already covered by `IN_ESIC`). In a large database scan this single pattern will dominate the finding count with false positives.

```go
// patterns.go:52
{"Bank Account", "IN_BANK_ACCOUNT", `\b[0-9]{9,18}\b`},
```

No Luhn or Verhoeff validator is applied to bank account numbers (unlike credit cards and Aadhaar). The validators exist in `apps/goScanner/internal/classifier/validators/` but are not wired to `IN_BANK_ACCOUNT`.

**Fix:** Apply IFSC-prefix structural check (Indian bank account must follow a schema tied to an IFSC code) or add a bank-name context keyword boost requirement (`score >= 70` threshold) before emitting a bank account finding.

---

### F-07 · HIGH — `PolicyEvaluationWorkflow` Activity Stubs Will Silently Succeed

**File:** `apps/backend/modules/scanning/activities/scan_activities.go:267-291`

Three activities called by `PolicyEvaluationWorkflow` are stub implementations that return nil/empty results:

```go
// scan_activities.go:268-291
func (a *ScanActivities) GetFinding(...) (map[string]interface{}, error) {
    var finding map[string]interface{}
    // TODO: Implement finding retrieval
    return finding, nil   // ← returns nil map, no error
}

func (a *ScanActivities) GetActivePolicies(...) ([]map[string]interface{}, error) {
    var policies []map[string]interface{}
    // TODO: Implement policy retrieval
    return policies, nil  // ← always empty, no error
}

func (a *ScanActivities) EvaluatePolicyConditions(...) (bool, error) {
    // TODO: Implement policy condition evaluation
    return false, nil     // ← always returns "no match"
}
```

The `PolicyEvaluationWorkflow` will always succeed silently — no finding fetched, no policies evaluated, no actions executed. No alarm is raised. This is not caught by any test.

**Fix:** Return a sentinel error `fmt.Errorf("not implemented: GetFinding — PolicyEvaluationWorkflow is not production-ready")` until the real implementation is wired, so the workflow fails explicitly rather than silently no-oping.

---

### F-08 · MEDIUM — `RemediationWorkflow` Activity Result Type Mismatch

**File:** `apps/backend/modules/scanning/workflows/scan_workflows.go:77-78`  
**Activity definition:** `apps/backend/modules/scanning/activities/scan_activities.go:192`

`ExecuteRemediation` returns `RemediationResult` (a struct with `ActionID`, `Status`, `Message` fields). The workflow deserializes the result into `var actionID string`. Temporal's SDK will deserialize `RemediationResult` into a string as JSON, producing the full JSON blob `{"action_id":"...","status":"...","message":"..."}` stored as the actionID string. The subsequent `RollbackRemediation` call passes this JSON blob as the `actionID`, and `UPDATE remediation_actions SET status = 'ROLLED_BACK' WHERE id = $1` will find zero rows.

```go
// scan_workflows.go:77-78
var actionID string  // ← wrong type
err := workflow.ExecuteActivity(ctx, "ExecuteRemediation", findingID, actionType, userID).Get(ctx, &actionID)
// actionID is now `{"action_id":"uuid","status":"pending","message":"remediation queued"}`
```

**Fix:**
```go
var result activities.RemediationResult
err := workflow.ExecuteActivity(ctx, "ExecuteRemediation", findingID, actionType, userID).Get(ctx, &result)
// ...
executedActions = append(executedActions, result.ActionID)
```

---

### F-09 · MEDIUM — `SyncToNeo4j` Is Fire-and-Forget in `ScanLifecycleWorkflow`

**File:** `apps/backend/modules/scanning/workflows/scan_workflows.go:44-45`

```go
// scan_workflows.go:44-45
// Async Neo4j sync (fire and forget - won't block completion)
workflow.ExecuteActivity(ctx, "SyncToNeo4j", scanID)
// ← .Get() is NOT called — result is discarded
```

This means: (a) Neo4j sync failures are never surfaced to the workflow — they appear as log warnings only; (b) if the worker crashes after `COMPLETED` is written to PostgreSQL but before Neo4j sync finishes, the lineage graph will be permanently out of sync with no retry; (c) there is no Temporal activity heartbeat covering the sync.

The comment says "won't block completion" which is correct, but the consequence is silent lineage divergence.

**Fix:** Either call `.Get(ctx, nil)` to make it synchronous (adds latency but guarantees consistency), or use a separate child workflow with its own retry policy dedicated to Neo4j sync.

---

### F-10 · LOW — `regexp.Compile` at Pattern Init Does Not Halt on Failure

**File:** `apps/goScanner/internal/classifier/patterns.go:69-73`

```go
// patterns.go:69-73
r, err := regexp.Compile(d.regex)
if err != nil {
    continue   // ← silently drops the pattern
}
```

A regex compilation failure silently drops the pattern. The built-in patterns use compile-time-safe literals and this will never fire in practice. However, the code path also runs for patterns loaded from the custom pattern table at scan time. A bad custom pattern is already pre-validated in `apps/backend/modules/scanning/service/patterns_service.go:103`, but this second compile (in `api/handler.go:215`) also silently skips the pattern without logging which one failed.

**Fix:** Log the skipped pattern name and regex at WARN level so operators can diagnose why custom patterns produce no findings.

---

## 4. Phase 3 — Orchestration

### F-11 · HIGH — Temporal Worker Is Optional; Scan State Machine Bypasses It by Default

**File:** `apps/backend/cmd/server/main.go:238-258`

Temporal is guarded by `TEMPORAL_ENABLED` which defaults to `false` (the env var is checked but has no docker-compose default). When disabled, the `ScanLifecycleWorkflow`, `RemediationWorkflow`, and streaming supervisor are all inactive. State transitions (`CREATED → RUNNING → COMPLETED`) happen ad-hoc through direct database calls from HTTP handlers, not through the workflow state machine.

```go
// main.go:238-258
if os.Getenv("TEMPORAL_ENABLED") == "true" {
    temporalAddress := getEnv("TEMPORAL_HOST_PORT", "localhost:7233")
    temporalWorker, err = worker.NewTemporalWorker(...)
    // ...
} else {
    log.Println("ℹ️  Temporal disabled (set TEMPORAL_ENABLED=true to enable)")
}
```

This means: for most deployments, there is no durable workflow, no retry policy, no compensation on failure, and no rollback for remediation. The production-quality orchestration described in the Temporal workflows is bypassed.

**Fix:** Make `TEMPORAL_ENABLED=true` the default in production Helm values. Add a startup warning when `TEMPORAL_ENABLED != true` in release mode.

---

### F-12 · MEDIUM — `RunStreamingWindowActivity` Creates a New Redis Client on Every Temporal Activity Invocation

**File:** `apps/backend/modules/scanning/activities/scan_activities.go:308-316`  
**Also:** `scan_activities.go:369-377`

```go
// scan_activities.go:308-316
func (a *ScanActivities) RunStreamingWindowActivity(...) ([]StreamMessage, error) {
    redisAddr := func() string { ... }()
    rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
    defer rdb.Close()
    // ...
}
```

`PersistStreamingCheckpoints` does the same. Temporal calls these activities on a schedule. Each invocation allocates a new TCP connection pool to Redis (go-redis maintains a pool per client), then closes it at activity end. Under sustained streaming load this creates connection churn: new TCP + TLS handshake per window tick, potentially exhausting Redis max connections.

**Fix:** Store the Redis client in `ScanActivities` struct, initialized once in `NewScanActivities`.

```go
type ScanActivities struct {
    db          *sql.DB
    neo4j       neo4j.DriverWithContext
    redis       *redis.Client    // ← add
    // ...
}
```

---

### F-13 · LOW — `IngestStreamingFindings` Is a Log-Only Stub

**File:** `apps/backend/modules/scanning/activities/scan_activities.go:349-366`

```go
// scan_activities.go:355-365
for _, msg := range msgs {
    scanID, ok := msg.Payload["scan_id"]
    // ...
    // Record acknowledgement — the actual ingest was already done by the scanner
    // posting to /ingest-verified. Here we count and log for Temporal tracking.
    log.Printf("INFO: streaming ack scan_id=%s msg_id=%s", scanID, msg.ID)
    ingested++
}
```

This activity only logs and counts — it does not perform any actual ingestion. It also does not XACK (acknowledge) the messages from the Redis stream, so the same messages will be re-delivered on the next window tick (XREAD with `"0"` reads from the beginning every time, not advancing the consumer group offset). The streaming pipeline will process every message on every tick.

**Fix:** Use Redis Consumer Groups with `XREADGROUP` and `XACK`. Advance the stream offset after successful processing.

---

## 5. Phase 4 — Data Lineage, Storage & Remediation

### F-14 · CRITICAL — PostgreSQL and Neo4j Have No Transactional Guarantee (Silent Divergence)

**File:** `apps/backend/modules/scanning/service/ingestion_service.go:552-573`

PostgreSQL findings are committed in a transaction. Neo4j sync runs **after** the commit, outside any transaction, without compensation:

```go
// ingestion_service.go:552-573
if err := tx.Commit(); err != nil {
    return nil, fmt.Errorf("commit failed: %w", err)
}
committed = true

// Sync to Neo4j AFTER commit — no saga, no outbox, no compensation
if s.neo4jRepo != nil {
    for assetID, piiTypes := range assetPIIMap {
        if err := s.neo4jRepo.SyncFindingsToPIICategories(ctx, assetID.String(), piiTypes); err != nil {
            log.Printf("WARN: neo4j PII category sync failed for asset %s: %v", assetID, err)
            // ← logged and swallowed; PostgreSQL already committed
        }
    }
}
```

A Neo4j unavailability, network partition, or Cypher error after PostgreSQL commit will leave findings in PostgreSQL with no corresponding graph nodes. The lineage graph silently diverges. There is no retry queue, no outbox table, no reconciliation job.

This is compounded by `ScanLifecycleWorkflow` calling `SyncToNeo4j` fire-and-forget (F-09).

**Fix:** Implement a transactional outbox: insert a `neo4j_sync_queue` row in the same PostgreSQL transaction as findings. A background worker (or Temporal activity with retry) dequeues rows and syncs to Neo4j, marking them complete. This provides at-least-once delivery with idempotent Cypher MERGE.

---

### F-15 · HIGH — Masking Has No Rollback Path for Source-System Changes

**File:** `apps/backend/modules/masking/service/masking_service.go:84-112`

The `MaskAsset` function updates finding display values in PostgreSQL (the platform's internal copy) and marks the asset as masked. It does **not** call any remediation connector to mask the actual data in the source system. There is no stored "original value" before masking and no `UnmaskAsset` method.

```go
// masking_service.go:84-112
// Update findings with masked values (in ARC-HAWK's own DB, not the source)
if err := s.findingRepo.UpdateMaskedValues(ctx, maskedData); err != nil { ... }
// Update asset masking status
if err := s.assetRepo.UpdateMaskingStatus(ctx, assetID, true, string(strategy)); err != nil { ... }
// Create audit log entry
// ← No call to remediation connector, no original value persisted for rollback
```

The platform records `[REDACTED]` in its own findings table but leaves the actual PII untouched in the external source. This creates a compliance false-positive: audit logs show "masked" but the source data is unchanged.

**Fix:** `MaskAsset` must dispatch to `RemediationService.ExecuteRemediation` per finding, OR clearly rename it `MaskFindingDisplayValues` so callers understand it only affects the platform's UI and does not touch source systems.

---

### F-16 · HIGH — S3 Remediation `RestoreValue` Deletes the Backup It Should Preserve

**File:** `apps/backend/modules/remediation/connectors/s3.go:213-232`

```go
// s3.go:213-232
func (c *S3Connector) RestoreValue(..., originalValue string) error {
    // Upload original content
    _, err := c.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
        Bucket: aws.String(c.bucket),
        Key:    aws.String(location),
        Body:   bytes.NewReader([]byte(originalValue)),
    })
    // ...
    // Remove backup   ← deletes the pre-remediation backup after restoring
    backupKey := location + ".backup"
    c.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(c.bucket),
        Key:    aws.String(backupKey),
    })
    return nil
}
```

Additionally, `Mask()` creates a backup at `location + ".backup"`, but `Delete()` creates a backup at `location + ".deleted.backup"`. `RestoreValue` always tries to delete `location + ".backup"`, so restoration after a `Delete` operation leaves a dangling `.deleted.backup` object and attempts to delete a non-existent `.backup` key (silently failing because the error is discarded with `_`).

**Fix:** Store the backup key alongside the remediation action record, pass it into `RestoreValue`.

---

### F-17 · MEDIUM — Remediation Connector `Filesystem.Encrypt` Is a No-Op

**File:** `apps/backend/modules/remediation/connectors/filesystem.go` (lines 114, 140, 173, 207)

All four filesystem remediation methods (`Mask`, `Delete`, `Encrypt`, `RestoreValue`) end with `return nil` — inspection shows only `Mask` and `Delete` have minimal stub logic. `Encrypt` writes nothing. `RestoreValue` writes nothing. The `ConnectorFactory` dispatches to these stubs for `source_type == "filesystem"`.

```go
// filesystem.go: RestoreValue stub pattern
func (c *FilesystemConnector) RestoreValue(...) error {
    // ...
    return nil   // ← no filesystem write, no error
}
```

**Fix:** Enumerate the filesystem connector as a stub in documentation and return `fmt.Errorf("filesystem RestoreValue: not implemented")` until real logic is written.

---

### F-18 · MEDIUM — MongoDB `GetOriginalValue` Returns Only the First Field Value; Encrypt Uses It

**File:** `apps/backend/modules/remediation/connectors/mongodb.go:148-176`

`GetOriginalValue` fetches the entire document and then tries to retrieve `fieldName` as a top-level string key. If the field is nested (e.g., `address.email`), or if the BSON value is not a string, the method returns an empty string with no error. `Encrypt` then encrypts the empty string and writes it, silently destroying the original value.

---

## 6. Phase 5 — Observability

### F-19 · HIGH — No Queue Depth Prometheus Metric; Temporal Task Queue Depth Is Invisible

**File:** `apps/backend/modules/scanning/api/scan_trigger_handler.go:36-65`

Prometheus metrics exist for scan trigger counts and HTTP latency:

```go
scanTriggerCounter      // counter
scanTriggerFailureCounter  // counter
scanTriggerDuration     // histogram
classificationConfidenceHist  // histogram
```

And ingestion totals in `ingestion_service.go`. However:

1. **No Temporal task queue depth metric** — there is no gauge for pending Temporal activities.
2. **No Redis stream lag metric** — the streaming supervisor has no Prometheus gauge for unprocessed message count.
3. **No Neo4j sync lag metric** — no metric for the number of assets pending neo4j sync.
4. **goScanner exposes `/metrics`** (`main.go:31`) but registers only the default Go runtime collectors — no scan-specific counters (findings per second, connector errors, Presidio latency).

**Fix:** Add at minimum:
- `arc_hawk_neo4j_sync_pending_total` gauge (count rows in `neo4j_sync_queue` outbox)
- `arc_hawk_scan_findings_per_second` counter in goScanner handler
- `arc_hawk_presidio_latency_seconds` histogram in `orchestrator/orchestrator.go`

---

### F-20 · MEDIUM — `AUTH_REQUIRED=false` Grants Admin Role to All Anonymous Requests

**File:** `apps/backend/modules/auth/middleware/auth_middleware.go:126-144`

```go
// auth_middleware.go:126-144
if os.Getenv("AUTH_REQUIRED") == "false" {
    // ...
    c.Set("user_role", "admin")    // ← every unauthenticated caller is admin
    c.Set("authenticated", false)
    c.Next()
    return
}
```

When `AUTH_REQUIRED=false` (the docker-compose default does not set `AUTH_REQUIRED`, leaving it unset which evaluates to `!= "false"`, so this only fires if explicitly set), every anonymous HTTP request receives `user_role = "admin"`. If any deployment sets `AUTH_REQUIRED=false` in a shared environment (e.g., internal staging), all API endpoints including remediation, masking, and user management are fully open.

**Fix:** Remove the `"admin"` role grant from the anonymous path. Assign `"viewer"` at most, and require explicit opt-in to elevated roles even in dev mode.

---

### F-21 · LOW — Prometheus `/metrics` Endpoint Is Behind Auth Middleware on Backend

**File:** `apps/backend/cmd/server/main.go:297`

```go
// main.go:297
// Prometheus Metrics endpoint (behind auth middleware)
```

The comment confirms `/metrics` is behind the auth middleware. This means Prometheus cannot scrape the backend without a valid JWT or API key. The `infra/k8s/monitoring/servicemonitor-backend.yaml` would need to be configured with credentials or the metrics endpoint must be moved to a separate non-authenticated port.

---

### F-22 · LOW — `log.Printf` Is Used Exclusively (374 call sites); No Structured Logging

**Count:** 374 `log.Printf`/`log.Println` calls across `apps/`

The entire codebase uses Go's stdlib `log` package — no `zerolog`, `zap`, or `slog`. Log lines are unstructured free-text with no trace IDs, no request IDs, no scan IDs as structured fields. Correlating a scan failure across backend → goScanner → Presidio requires string-grepping log files.

**Fix:** Adopt `log/slog` (stdlib, Go 1.21+, already available) with a structured JSON handler. Add `slog.With("scan_id", scanID)` at handler entry points.

---

## 7. Additional Findings

### F-23 · MEDIUM — Celery K8s Manifests Have No Backing Implementation

**Files:** `k8s/celery/*.yaml`, `helm/arc-hawk-dd/templates/celery-workers.yaml`

Eight Celery deployment manifests exist (`deployment-escalation.yaml`, `deployment-ingest-streaming.yaml`, `deployment-classify.yaml`, etc.) plus KEDA ScaledObjects. There is no Python Celery worker code anywhere in `apps/`. These manifests reference a Docker image but no image is built by any Dockerfile in the repo. Deploying these manifests would result in image-pull failures.

---

### F-24 · LOW — Oracle Connector Is a Build-Tag Stub Registered in the Dispatcher

**File:** `apps/goScanner/internal/connectors/databases/oracle.go:1`

```go
//go:build connector_stub
```

The `connector_stub` build tag is not a standard build tag. The connector is never compiled into the default binary. However, if a user configures a connection with `source_type = "oracle"`, the dispatcher `connectors.Dispatch("oracle")` will return "unsupported source type" rather than a meaningful "Oracle requires Instant Client" message, because the file is excluded from the build.

---

### F-25 · LOW — `sslmode=require` in Remediation PostgreSQL Connector Is Not Enforced

**File:** `apps/backend/modules/remediation/connectors/postgresql.go:39-44`

The remediation connector defaults `ssl_mode` to `"require"`, which is good. However, the `ssl_mode` value comes from the connection config map which originates from user-supplied JSON. A caller can pass `"ssl_mode": "disable"` and bypass TLS. There is no server-side enforcement.

---

## 8. Production Readiness Scorecard

| Dimension | Score | Key Issues |
|-----------|-------|------------|
| **Secret Management** | 4/10 | Vault disabled by default; ENCRYPTION_KEY/JWT_SECRET from env; no Vault token renewal |
| **Concurrency Safety** | 7/10 | errgroup + semaphore in orchestrator is correct; `sync.Mutex` on findings aggregation is correct; Redis client-per-activity is wasteful but not unsafe |
| **IPC Security** | 2/10 | Scanner endpoint has zero authentication; no mTLS between services; scanner port 8001 not restricted to 127.0.0.1 in docker-compose |
| **Classification Reliability** | 6/10 | Luhn/Verhoeff validators present and wired for CC/Aadhaar/PAN; `IN_BANK_ACCOUNT` has extreme false positives; `LIMIT 10000` causes silent miss on large tables |
| **Orchestration** | 5/10 | Temporal is present but disabled by default; three policy activities are TODO stubs; RemediationWorkflow has type mismatch; streaming XACK not implemented |
| **Dual-DB Consistency** | 2/10 | No outbox/saga pattern; Neo4j sync after PostgreSQL commit with no compensation; fire-and-forget in workflow; silent divergence on Neo4j unavailability |
| **Remediation Safety** | 3/10 | MaskAsset does not touch source systems; S3 restore has backup key mismatch; filesystem encrypt/restore are no-ops; MongoDB field path resolution is brittle |
| **Observability** | 4/10 | Prometheus integrated for scan triggers; no queue depth/stream lag/neo4j lag metrics; goScanner has no custom metrics; no structured logging; /metrics behind auth |
| **Overall** | **4/10** | Platform has solid foundations (connector architecture, Temporal scaffold, Vault plumbing, classifier with validators) but critical gaps in IPC security, dual-DB consistency, and remediation correctness block production readiness |

---

## 9. Priority Fix List

| Priority | Finding | Effort |
|----------|---------|--------|
| P0 | F-01: Scanner IPC authentication | Small (add middleware + token header) |
| P0 | F-14: Dual-DB consistency outbox | Medium (new table + background worker) |
| P0 | F-07: Policy workflow stubs must fail explicitly | Trivial (return error instead of nil) |
| P1 | F-08: RemediationWorkflow type mismatch | Trivial (change `var actionID string` → `var result RemediationResult`) |
| P1 | F-09: SyncToNeo4j fire-and-forget | Small (add `.Get(ctx, nil)` or child workflow) |
| P1 | F-11: Temporal disabled by default | Small (Helm default + startup warning) |
| P1 | F-15: Masking does not touch source systems | Medium (wire MaskAsset to remediation connectors) |
| P2 | F-02: Vault disabled by default | Medium (AppRole auth + Helm default) |
| P2 | F-03: Scanner sslmode=disable | Trivial (one-line fix) |
| P2 | F-06: IN_BANK_ACCOUNT false positives | Small (add context keyword threshold) |
| P2 | F-16: S3 restore backup key mismatch | Small (pass backup key through action record) |
| P2 | F-12: Redis client-per-activity | Small (move client to struct) |
| P3 | F-19: Missing Prometheus metrics | Medium (add gauges for queue depth, neo4j lag) |
| P3 | F-20: AUTH_REQUIRED=false grants admin | Trivial (change "admin" to "viewer") |
| P3 | F-22: No structured logging | Large (slog migration) |
| P3 | F-23: Celery manifests with no code | Small (delete or mark as legacy) |
