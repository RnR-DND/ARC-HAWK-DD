# GSD (Get Shit Done) Production Report
## ARC-HAWK-DD — Enterprise PII Discovery Platform
**Audit Date:** 2026-04-23 | **Branch:** ralph/memory-and-tenant-fixes | **Auditor:** Multi-agent parallel analysis

---

## SEVERITY LEGEND
| Level | Meaning |
|-------|---------|
| **CRITICAL** | Blocks production deploy. Data loss, security breach, or compliance violation at rest. |
| **HIGH** | Must fix before merge. Functional failure or security risk under normal load. |
| **MEDIUM** | Fix before next release. Degrades reliability or observability. |
| **LOW** | Tech debt / hardening. |

---

## SECTION 1: TOOLING & CI/CD BLOCKERS

### [C1] Supermemory API key `sm_XXDSFvG3...` was previously committed — rotation unconfirmed
**CRITICAL**
Key prefix visible in comments at `.env:68` and `apps/backend/.env:3`. The full key was exposed. There is no evidence it has been rotated in any external service.
```bash
# Verify key is dead by testing it — if it returns 200, rotate immediately
curl -H "Authorization: Bearer sm_XXDSFvG3..." https://api.supermemory.ai/v1/memories
```

---

### [H1] `golang:1.25-alpine` base image does not exist — all backend Docker builds fail
**HIGH**
`apps/backend/Dockerfile:1` and `apps/backend/go.mod:3` declare `go 1.25`, which is not a released Go version. Every `docker build` on the backend image will fail to pull the base image on any current Docker host.
```dockerfile
# Fix: apps/backend/Dockerfile line 1
FROM golang:1.23-alpine AS builder

# Fix: apps/backend/go.mod line 3
go 1.23
```

---

### [H2] `build.yml` has no `pull_request` trigger — cannot block PRs from merging
**HIGH**
`.github/workflows/build.yml:3-8` only triggers on `push` to `main`. PRs are never validated. Broken code merges freely.
```yaml
# Fix: apps/backend/build.yml — add PR trigger
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
```

---

### [H3] Backend CI jobs have no `working-directory` — `go test` and `go build` run from repo root
**HIGH**
`.github/workflows/build.yml:30-36`: The backend source is in `apps/backend/`, not the root. The job will fail on `go test ./...` because there is no Go module at the repo root.
```yaml
# Fix: add working-directory to every backend job step
jobs:
  backend:
    steps:
      - name: Run tests
        working-directory: apps/backend
        run: go test ./...
```

---

### [H4] Scanner Docker image pushed to Hub with no `needs:` gate — broken image can ship
**HIGH**
`.github/workflows/build.yml:75-89`: The scanner image build-and-push job declares no `needs:` dependency on the test job. A broken scanner ships to production Docker Hub on every `main` push.
```yaml
# Fix: add needs dependency
jobs:
  build-scanner:
    needs: [test-scanner]
```

---

### [H5] Neo4j password `password123` hardcoded in CI YAML
**HIGH**
`.github/workflows/ci-cd.yml:182,187,207,222`: Neo4j service container and test env vars use `password123` as plaintext committed credentials.
```yaml
# Fix: use GitHub Actions secrets
NEO4J_PASSWORD: ${{ secrets.CI_NEO4J_PASSWORD }}
```

---

### [H6] Hardcoded Postgres DSN committed in Go source
**HIGH**
`apps/backend/cmd/debug/main.go:16`: `postgres://postgres:postgres@localhost:5432/arc_platform?sslmode=disable` is a hardcoded fallback.
```go
// Fix: remove fallback; fail fast if env missing
dsn := os.Getenv("DATABASE_URL")
if dsn == "" {
    log.Fatal("DATABASE_URL required")
}
```

---

### [H7] `AUTH_REQUIRED=false` + `VAULT_DEV_ROOT_TOKEN` persisted in `.env`
**HIGH**
`.env:27,75`: Running with auth disabled exposes all API endpoints. Vault root dev token must never leave localhost. Both are present in the on-disk `.env`.
```bash
# Minimum fix: add to .env.example and document that .env must NEVER be shared
# Vault: switch to AppRole auth before staging deploy
```

---

### [M1] No SAST, `gosec`, `golangci-lint`, or container scanning in any CI pipeline
**MEDIUM**
Zero static security analysis across all three workflow files.
```yaml
# Add to ci-cd.yml:
- name: Security scan
  uses: securecodewarrior/github-action-gosec@v2
- name: Container scan
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: ${{ env.IMAGE_TAG }}
```

---

### [M2] Regression quality gate is a stub `echo` — F1 score never validated
**MEDIUM**
`.github/workflows/regression.yml:45-48`: The step prints a string but never reads the actual F1 score from the binary output and does not fail the job.
```yaml
# Fix: capture and assert F1 score
- name: Check Quality Gate
  run: |
    SCORE=$(go run cmd/regression/main.go | grep "F1:" | awk '{print $2}')
    awk "BEGIN {if ($SCORE < 0.85) exit 1}"
```

---

### [M3] `alpine:latest` unpinned in backend final stage
**MEDIUM**
`apps/backend/Dockerfile:18`: Unpinned `latest` tag breaks reproducibility and silently upgrades the base image.
```dockerfile
FROM alpine:3.20 AS runtime
```

---

## SECTION 2: ARCHITECTURE & STATE FLAWS

### [C2] `time.Now()` called inside Temporal workflow — non-determinism violation
**CRITICAL**
`apps/backend/modules/scanning/workflows/scan_workflows.go:101`:
```go
// BROKEN — non-deterministic on replay:
workflow.ExecuteActivity(ctx, "CloseExposureWindow", findingID, time.Now())

// FIX — use workflow-safe time:
workflow.ExecuteActivity(ctx, "CloseExposureWindow", findingID, workflow.Now(ctx))
```
On replay, `time.Now()` returns a different value than original execution, causing history mismatch and workflow panic.

---

### [C3] Map iteration in `flattenCheckpoints()` — non-deterministic `ContinueAsNew` input
**CRITICAL**
`apps/backend/modules/scanning/workflows/streaming_supervisor_workflow.go:209`: Go map iteration is random. The resulting slice passed to `ContinueAsNew` differs on every replay, breaking Temporal determinism.
```go
// Fix: sort the slice after iteration
func flattenCheckpoints(m map[string]StreamingCheckpointState) []StreamingCheckpointState {
    out := make([]StreamingCheckpointState, 0, len(m))
    for _, v := range m {
        out = append(out, v)
    }
    sort.Slice(out, func(i, j int) bool {
        return out[i].SourceID < out[j].SourceID
    })
    return out
}
```

---

### [C4] Celery K8s manifests reference a Python app that does not exist
**CRITICAL**
All 7 manifests in `k8s/celery/` and `helm/arc-hawk-dd/templates/celery-workers.yaml` reference `app.tasks.celery_app`. No Python Celery code exists in this repository. Deploying these manifests causes CrashLoopBackOff. `helm/arc-hawk-dd/values.yaml` also has no `celeryWorkers:` stanza, so the Helm template render will fail.
```bash
# Fix option A: delete the manifests entirely (Go streaming via Redis Streams is the real broker)
rm k8s/celery/*.yaml
rm helm/arc-hawk-dd/templates/celery-workers.yaml

# Fix option B: if Python workers are planned, stub values.yaml:
# celeryWorkers:
#   enabled: false
```

---

### [H8] `CloseExposureWindow` called with wrong argument types — runtime deserialization failure
**HIGH**
`scan_workflows.go:101` passes `(findingID, time.Now())` but the activity signature at `scan_activities.go:161` is `(ctx, assetID string, piiType string, closedAt time.Time)`. `findingID` is passed as `assetID`, `piiType` is omitted entirely. This will either cause a Temporal deserialization panic or silently update the wrong Neo4j node at runtime.
```go
// Fix — resolve assetID and piiType from the finding before the workflow call:
workflow.ExecuteActivity(ctx, a.CloseExposureWindow, assetID, piiType, workflow.Now(ctx))
```

---

### [H9] XACK before processing — message loss on Temporal activity retry
**HIGH**
`apps/backend/modules/scanning/activities/scan_activities.go:477-483`: Messages are XACKed immediately after XREADGROUP, before processing. If Temporal retries the activity (network error, timeout), the messages are already acknowledged and cannot be re-delivered from the PEL.
```go
// Fix: XACK only after successful processing
for _, msg := range msgs {
    if err := processMessage(msg); err != nil {
        return err // Temporal will retry; message stays in PEL
    }
    rdb.XAck(ctx, streamKey, groupName, msg.ID)
}
```

---

### [H10] KEDA ScaledObjects monitor Redis List — actual queue is Redis Streams — autoscaling never fires
**HIGH**
All `k8s/celery/keda-scaledobject-*.yaml` use `type: redis` with `listName:`. The actual implementation uses Redis Streams (XADD/XREADGROUP). KEDA sees a permanently empty list. Workers sit at `minReplicaCount: 2` regardless of backlog.
```yaml
# Fix: switch to redis-streams trigger
triggers:
- type: redis-streams
  metadata:
    address: redis:6379
    stream: arc-hawk:stream:scan
    consumerGroup: arc-hawk-workers
    pendingEntriesCount: "50"
```

---

### [H11] Neo4j sync worker has no `SELECT FOR UPDATE SKIP LOCKED` — race on concurrent pods
**HIGH**
`neo4j_sync_worker.go:51-57`: Plain `SELECT` without row locking means multiple backend replicas will process the same batch concurrently, producing duplicate Neo4j writes.
```sql
-- Fix: apps/backend/modules/shared/infrastructure/persistence/neo4j_sync_worker.go
SELECT id, operation, payload 
FROM neo4j_sync_queue 
WHERE status IN ('pending','failed') AND attempts < 5 
ORDER BY created_at 
LIMIT $1
FOR UPDATE SKIP LOCKED
```

---

### [H12] Redis client in ScanActivities created without AUTH password
**HIGH**
`apps/backend/modules/scanning/activities/scan_activities.go:41`: `goredis.NewClient(&goredis.Options{Addr: redisAddr})` — no `Password` field. In any Redis deployment with ACL enforcement, every XREADGROUP/XACK call fails with `NOAUTH`.
```go
// Fix:
rdb: goredis.NewClient(&goredis.Options{
    Addr:     redisAddr,
    Password: os.Getenv("REDIS_PASSWORD"),
}),
```

---

### [H13] `http.DefaultClient` (no timeout) in Firebase connector — goroutine blocks indefinitely
**HIGH**
`apps/goScanner/internal/connectors/databases/firebase.go:56,82`: `http.DefaultClient.Do(req)` has no timeout. Slow Firebase endpoints block the goroutine permanently.
```go
// Fix: use context-aware client
client := &http.Client{Timeout: 30 * time.Second}
resp, err := client.Do(req.WithContext(ctx))
```

---

### [H14] YAML syntax error in Helm values.yaml — `helm install` fails
**HIGH**
`helm/arc-hawk-dd/values.yaml:207`: `initialDelaySeconds` and `periodSeconds` are incorrectly indented under `secretName`, producing a YAML mapping error.
```yaml
# Fix: correct indentation
supermemory:
  secretName: "supermemory"
livenessProbe:
  initialDelaySeconds: 10
  periodSeconds: 10
```

---

### [M4] `dataRows` not deferred-closed on `Scan()` error in DB connectors — connection pool leak
**MEDIUM**
`postgres.go:111`, `mysql.go:82`, `mssql.go:80`, and warehouse connectors: `dataRows.Close()` at end-of-loop is unreachable on scan error. Use `defer` immediately after open.
```go
// Fix pattern for all DB connectors:
dataRows, err := c.db.QueryContext(ctx, query, args...)
if err != nil { return err }
defer dataRows.Close() // not at end of loop
```

---

### [M5] `rows.Err()` never checked in any DB connector — partial scan results treated as complete
**MEDIUM**
All connectors in `apps/goScanner/internal/connectors/databases/` and `warehouses/` omit `rows.Err()` after iteration. Network interruptions mid-scan are silently ignored.
```go
// Fix: add after every rows iteration
if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("rows iteration error: %w", err)
}
```

---

### [M6] MySQL sampling via `ORDER BY RAND()` — full-table O(N) sort
**MEDIUM**
`apps/goScanner/internal/connectors/databases/mysql.go:70`: On multi-GB tables, this causes extreme latency and MySQL memory pressure.
```sql
-- Fix: use inline sampling
SELECT * FROM `%s` WHERE RAND() < %f LIMIT %d
```

---

### [M7] No cross-source deduplication at orchestrator level
**MEDIUM**
Dedup is intra-source only (`dedup.go`). Same PII value appearing in multiple data sources produces duplicate findings with identical `(pii_type, source_path, value)` in the output.

---

### [M8] No HPA for any service component
**MEDIUM**
`helm/arc-hawk-dd/values.yaml`: No `HorizontalPodAutoscaler` exists anywhere. Backend, scanner, and Presidio pods have fixed `replicaCount`. Scan burst traffic is not handled.

---

### [M9] Redis persistence disabled by default — streaming checkpoints lost on restart
**MEDIUM**
`helm/arc-hawk-dd/values.yaml:140`: `persistence.enabled: false`. Non-prod deployments lose all pending stream messages and `arc-hawk:stream:checkpoint:*` keys on Redis restart.

---

### [M10] Temporal Deployment missing liveness/readiness probes
**MEDIUM**
`helm/arc-hawk-dd/templates/platform.yaml:19-41`: If Temporal hangs during DB migration, K8s will not restart it. Workflow workers fail silently.

---

## SECTION 3: DATA INTEGRITY & COMPLIANCE RISKS

### [C5] `GET /findings` emits no audit event — DPDPA Section 8(2) accountability gap
**CRITICAL**
`apps/backend/modules/assets/api/findings_handler.go:30-95` and `findings_service.go:62-90`: Every bulk PII read is untracked. No `AuditLogger` is injected into `FindingsHandler`. This is the highest-frequency read path for PII data and produces zero audit trail. DPDPA Section 8(2) requires accountability for every access to personal data.
```go
// Fix: inject and call audit logger in FindingsHandler.GetFindings
h.auditLogger.Record(ctx, entity.AuditEvent{
    Action:     "FINDINGS_ACCESSED",
    ResourceID: tenantID,
    Details:    map[string]any{"count": len(findings), "filters": req.Filters},
})
```

---

### [C6] `CloseExposureWindow` argument mismatch causes wrong Neo4j node update
**CRITICAL** — (also listed in Section 2 as H8)
Beyond the Temporal panic risk: if deserialization silently coerces `findingID` → `assetID`, the wrong asset's exposure window is marked closed in Neo4j. DPDPA lineage graph shows compliant state for the wrong asset.

---

### [H15] `LedgerLogger` (DPDPA compliance events) wired only to compliance module — not scanning or remediation
**HIGH**
`EventPIIDiscovered`, `EventRemediationApplied`, `EventConsentRevoked` are defined as constants but scanning and remediation modules use the hash-chained `PostgresAuditLogger` instead of `LedgerLogger`. These events never reach `audit_ledger`, breaking the DPDPA evidence chain for the most compliance-critical operations.
```go
// Fix: inject LedgerLogger into IngestionService and RemediationService
// Call on every PII discovery:
s.ledger.Record(ctx, audit.LedgerEvent{Type: audit.EventPIIDiscovered, ...})
// Call on every remediation:
s.ledger.Record(ctx, audit.LedgerEvent{Type: audit.EventRemediationApplied, ...})
```

---

### [H16] `audit_ledger` table has no DB-level write protection — "immutable" only by comment
**HIGH**
`000048_audit_ledger.up.sql`: `COMMENT ON TABLE audit_ledger IS 'Never UPDATE or DELETE'` is advisory only. Any app or DBA with `UPDATE`/`DELETE` privilege can tamper with compliance evidence.
```sql
-- Fix: add enforcement rules
CREATE RULE no_update_audit_ledger AS ON UPDATE TO audit_ledger DO INSTEAD NOTHING;
CREATE RULE no_delete_audit_ledger AS ON DELETE TO audit_ledger DO INSTEAD NOTHING;
-- Or use a trigger to raise exception on UPDATE/DELETE
```

---

### [H17] IP address never populated in `PostgresAuditLogger.Record` — all audit entries have blank IP
**HIGH**
`apps/backend/modules/shared/infrastructure/audit/postgres_logger.go:58-100`: `IPAddress` is never extracted from context or request. All 10+ audit event types (`SCAN_COMPLETED`, `REMEDIATION_EXECUTED`, etc.) produce non-attributable records, defeating DPDPA accountability traceability.
```go
// Fix: extract IP in middleware, store in context
// In audit logger:
ipAddress, _ := ctx.Value(contextkeys.IPAddress).(string)
```

---

### [H18] Row-Level Security applied to only 3 tables — audit_logs, classifications, consent_records excluded
**HIGH**
`000047_row_level_security.up.sql`: RLS is enabled on `findings`, `scan_runs`, `assets` only. `audit_logs`, `classifications`, `consent_records`, `masking_audit_log` have no tenant isolation at the DB layer. Cross-tenant data leakage is possible if the app layer misconfigures tenant context.
```sql
-- Fix: add RLS to remaining tables
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_logs
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
-- Repeat for classifications, consent_records, masking_audit_log
```

---

### [H19] `neo4j_sync_queue` has no `tenant_id` column — cross-tenant payload leakage risk
**HIGH**
`000050_neo4j_sync_queue.up.sql`: The table schema has no `tenant_id`. The JSONB payload contains `asset_id` + `pii_type_counts` without tenant scoping. If Neo4j repository doesn't enforce tenant isolation during sync, one tenant's scan data can pollute another's lineage graph.
```sql
-- Fix: add tenant_id to migration
ALTER TABLE neo4j_sync_queue ADD COLUMN tenant_id UUID NOT NULL;
CREATE INDEX ON neo4j_sync_queue(tenant_id, status);
```

---

### [H20] No dead-letter handling after 5 neo4j sync failures — permanent silent divergence
**HIGH**
`neo4j_sync_worker.go:54`: Rows with `attempts >= 5` are silently excluded from processing forever. No status transition to `dead_letter`, no alert, no operator notification. Postgres and Neo4j permanently diverge with no visibility.
```go
// Fix: transition to dead_letter status + emit alert metric
if attempts >= 5 {
    db.ExecContext(ctx, "UPDATE neo4j_sync_queue SET status='dead_letter' WHERE id=$1", id)
    neo4jSyncDeadLetterTotal.Inc() // Prometheus metric
}
```

---

### [H21] ScanLifecycleWorkflow swallows Neo4j sync failure — scan marked COMPLETED with stale lineage
**HIGH**
`scan_workflows.go:46-48`: `SyncToNeo4j` failure is logged but workflow continues to COMPLETED state. Lineage is permanently incomplete while compliance posture shows clean.
```go
// Fix: fail the workflow or transition to DEGRADED state on sync failure
if err := syncFuture.Get(ctx, nil); err != nil {
    return fmt.Errorf("neo4j sync failed — scan cannot be marked complete: %w", err)
}
```

---

### [H22] Remediation: source mutation succeeds but Neo4j lineage sync failure not rolled back
**HIGH**
`remediation_service.go:148-158`: Step 8 mutates source (mask/delete), step 9 marks `COMPLETED` in Postgres, step 10 syncs Neo4j. If step 10 fails — logged, not surfaced. PII is altered but lineage graph shows old exposure state.

---

### [H23] `RollbackRemediation` does not undo Neo4j exposure edge
**HIGH**
`scan_activities.go:241-271`, `remediation_service.go:173-235`: Rollback restores source value + updates Postgres but makes no Neo4j call. After rollback, Neo4j may show the exposure window as closed when the original PII data is restored.

---

### [H24] Batch remediation has no saga — partial completion is silent and unrecoverable
**HIGH**
`remediation_service.go:339-364`: `ExecuteRemediationRequest` iterates all `FindingIDs` with no compensating rollback. Failures increment a counter but the final `Status` is always `"COMPLETED"` regardless of how many findings failed.

---

### [H25] Migration 000051 down.sql drops wrong table name — rollback silently fails
**HIGH**
`000051_fp_learning.up.sql` creates `fp_learning`. `000051_fp_learning.down.sql` runs `DROP TABLE IF EXISTS fp_learnings` (plural). The rollback silently succeeds but the table remains. Re-running the up migration fails on duplicate table.
```sql
-- Fix: 000051_fp_learning.down.sql
DROP TABLE IF EXISTS fp_learning;
```

---

### [H26] `remediation_actions_failed_total` metric referenced in alert rules but never registered
**HIGH**
`infra/monitoring/prometheus/rules/arc-hawk-rules.yml:54`: `RemediationFailureRate` alert fires on `remediation_actions_failed_total`. No Go file in the repo registers this metric. The alert will never fire in production — remediation failures are invisible.
```go
// Fix: register in remediation_service.go
var remediationActionsFailed = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "remediation_actions_failed_total",
    Help: "Total failed remediation actions",
}, []string{"tenant_id", "action_type"})
```

---

### [H27] No Temporal workflow failure rate metric exists anywhere
**HIGH**
No Go code registers metrics for `WorkflowExecutionFailed` or activity error rates. The Temporal SDK emits internal metrics but these are not bridged to the project's Prometheus registry. `ScanLifecycleWorkflow` and `RemediationWorkflow` can fail silently with zero telemetry.
```go
// Fix: register in scan_workflows.go or a dedicated telemetry init
var workflowFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "temporal_workflow_failures_total",
}, []string{"workflow_type"})
```

---

### [M11] OTel traces exported over insecure gRPC — trace metadata in plaintext
**MEDIUM**
`apps/backend/modules/shared/telemetry/otel.go:33`: `otlptracegrpc.WithInsecure()` hardcoded unconditionally. In production, trace metadata traverses the network unencrypted.
```go
// Fix: use TLS or make it conditional
if os.Getenv("OTEL_INSECURE") != "true" {
    opts = append(opts, otlptracegrpc.WithTLSClientConfig(tlsConfig))
}
```

---

### [M12] No ServiceMonitor CRD for backend or scanner — Prometheus Operator won't scrape them
**MEDIUM**
`k8s/` has no `ServiceMonitor` for `backend:8080` or `go-scanner:8001`. In a Prometheus Operator deployment, static `prometheus.yml` scrape config is ignored. Metrics exist but are never collected.
```yaml
# Add: k8s/monitoring/servicemonitor-backend.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: arc-hawk-backend
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: backend
  endpoints:
  - port: http
    path: /metrics
```

---

### [M13] Celery ServiceMonitor CRD references non-existent pods — Prometheus scrape errors
**MEDIUM**
`infra/k8s/monitoring/servicemonitor-celery.yaml` has an explicit comment: "LEGACY: references Python Celery workers that have no active implementation." Deploying produces endless scrape errors in Prometheus Operator.
```bash
rm infra/k8s/monitoring/servicemonitor-celery.yaml
```

---

### [M14] Migration 000039 backfill is non-idempotent — re-run overwrites legitimate 1.0-confidence scores
**MEDIUM**
`000039_backfill_confidence_scores.up.sql:37-42`: `WHERE confidence_score IN (0.5, 0.66, 1.0)` includes `1.0`. Every re-run overwrites legitimately high-confidence classifications.
```sql
-- Fix: add a migration flag or narrow the WHERE clause
WHERE confidence_score = 0.5 AND manually_reviewed = false
```

---

## SECTION 4: ACTIONABLE REMEDIATION COMMANDS

### Immediate (before next deploy)

```bash
# 1. Fix Go version (breaks ALL Docker builds)
sed -i '' 's/golang:1.25-alpine/golang:1.23-alpine/g' apps/backend/Dockerfile
sed -i '' 's/^go 1.25.0/go 1.23.0/' apps/backend/go.mod

# 2. Fix Temporal non-determinism (CRITICAL — corrupts workflow history)
# scan_workflows.go:101 — replace time.Now() with workflow.Now(ctx)
# streaming_supervisor_workflow.go:209 — add sort.Slice after map iteration

# 3. Fix CloseExposureWindow argument mismatch
# scan_workflows.go:101 — pass assetID, piiType instead of findingID

# 4. Fix 000051 down migration table name
echo "DROP TABLE IF EXISTS fp_learning;" > apps/backend/migrations_versioned/000051_fp_learning.down.sql

# 5. Rotate supermemory API key — verify old key is dead
curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer sm_XXDSFvG3..." https://api.supermemory.ai/v1/memories
# If 200: rotate immediately in supermemory dashboard

# 6. Add SKIP LOCKED to neo4j sync worker
# neo4j_sync_worker.go:51 — append "FOR UPDATE SKIP LOCKED" to SELECT

# 7. Fix KEDA ScaledObjects to use redis-streams trigger type
# k8s/celery/keda-scaledobject-*.yaml — change type: redis → type: redis-streams

# 8. Fix Helm values.yaml YAML syntax error (breaks helm install)
# helm/arc-hawk-dd/values.yaml:207 — fix indentation of initialDelaySeconds/periodSeconds

# 9. Add pull_request trigger to build.yml CI
# .github/workflows/build.yml:3 — add pull_request: branches: [main]

# 10. Add working-directory to backend CI steps
# .github/workflows/build.yml — add working-directory: apps/backend
```

### Before next release

```bash
# 11. Add RLS to excluded tables
psql $DATABASE_URL << 'EOF'
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_logs USING (tenant_id = current_setting('app.tenant_id')::uuid);
ALTER TABLE classifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON classifications USING (tenant_id = current_setting('app.tenant_id')::uuid);
ALTER TABLE consent_records ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON consent_records USING (tenant_id = current_setting('app.tenant_id')::uuid);
EOF

# 12. Add tenant_id to neo4j_sync_queue
psql $DATABASE_URL << 'EOF'
ALTER TABLE neo4j_sync_queue ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_neo4j_sync_tenant ON neo4j_sync_queue(tenant_id, status);
EOF

# 13. Add audit_ledger write-protection rules
psql $DATABASE_URL << 'EOF'
CREATE OR REPLACE RULE no_update_audit_ledger AS ON UPDATE TO audit_ledger DO INSTEAD NOTHING;
CREATE OR REPLACE RULE no_delete_audit_ledger AS ON DELETE TO audit_ledger DO INSTEAD NOTHING;
EOF

# 14. Register missing Prometheus metric
# apps/backend/modules/remediation/service/remediation_service.go — add promauto.NewCounterVec for remediation_actions_failed_total

# 15. Add ServiceMonitors for backend and scanner
kubectl apply -f k8s/monitoring/servicemonitor-backend.yaml
kubectl apply -f k8s/monitoring/servicemonitor-scanner.yaml

# 16. Remove legacy Celery manifests
git rm k8s/celery/*.yaml
git rm helm/arc-hawk-dd/templates/celery-workers.yaml
git rm infra/k8s/monitoring/servicemonitor-celery.yaml

# 17. Fix Redis AUTH in ScanActivities
# scan_activities.go:41 — add Password: os.Getenv("REDIS_PASSWORD") to goredis.Options

# 18. Add REDIS_PASSWORD to Kubernetes secrets
kubectl create secret generic scanner-secrets \
  --from-literal=REDIS_PASSWORD="$(openssl rand -base64 32)" \
  --dry-run=client -o yaml | kubectl apply -f -

# 19. Fix Firebase / CouchDB / Salesforce / HubSpot / Teams connectors — add http timeouts
# Each file: replace &http.Client{} with &http.Client{Timeout: 30 * time.Second}

# 20. Fix dataRows defer-close in all DB connectors
# Pattern: move dataRows.Close() to defer immediately after open

# 21. Add rows.Err() checks to all DB connectors
# Pattern: after for rows.Next() loop, check if err := rows.Err(); err != nil { return err }
```

---

## FINDINGS SUMMARY

| ID | Category | Severity | File(s) |
|----|----------|----------|---------|
| C1 | Secrets | CRITICAL | `.env:68`, `apps/backend/.env:3` |
| C2 | Temporal | CRITICAL | `scan_workflows.go:101` |
| C3 | Temporal | CRITICAL | `streaming_supervisor_workflow.go:209` |
| C4 | K8s Deploy | CRITICAL | `k8s/celery/*.yaml`, `helm/.../celery-workers.yaml` |
| C5 | DPDPA | CRITICAL | `findings_handler.go:30-95` |
| C6 | DB Integrity | CRITICAL | `scan_workflows.go:101` + `scan_activities.go:161` |
| H1 | Build | HIGH | `apps/backend/Dockerfile:1`, `go.mod:3` |
| H2 | CI/CD | HIGH | `build.yml:3-8` |
| H3 | CI/CD | HIGH | `build.yml:30-36` |
| H4 | CI/CD | HIGH | `build.yml:75-89` |
| H5 | Secrets | HIGH | `ci-cd.yml:182,187` |
| H6 | Secrets | HIGH | `cmd/debug/main.go:16` |
| H7 | Secrets | HIGH | `.env:27,75` |
| H8 | Temporal | HIGH | `scan_workflows.go:101` |
| H9 | Streaming | HIGH | `scan_activities.go:477-483` |
| H10 | K8s/KEDA | HIGH | `keda-scaledobject-*.yaml:23` |
| H11 | DB Integrity | HIGH | `neo4j_sync_worker.go:51-57` |
| H12 | Secrets | HIGH | `scan_activities.go:41` |
| H13 | Connectors | HIGH | `firebase.go:56,82` |
| H14 | K8s Deploy | HIGH | `values.yaml:207` |
| H15 | DPDPA | HIGH | `compliance/module.go:65` |
| H16 | DPDPA | HIGH | `000048_audit_ledger.up.sql` |
| H17 | DPDPA | HIGH | `postgres_logger.go:58-100` |
| H18 | DPDPA | HIGH | `000047_row_level_security.up.sql` |
| H19 | DB Integrity | HIGH | `000050_neo4j_sync_queue.up.sql` |
| H20 | DB Integrity | HIGH | `neo4j_sync_worker.go:54` |
| H21 | DB Integrity | HIGH | `scan_workflows.go:46-48` |
| H22 | DB Integrity | HIGH | `remediation_service.go:148-158` |
| H23 | DB Integrity | HIGH | `scan_activities.go:241-271` |
| H24 | DB Integrity | HIGH | `remediation_service.go:339-364` |
| H25 | Migrations | HIGH | `000051_fp_learning.down.sql` |
| H26 | Observability | HIGH | `arc-hawk-rules.yml:54` |
| H27 | Observability | HIGH | `scan_workflows.go` (missing) |
| M1 | CI/CD | MEDIUM | `.github/workflows/*.yml` |
| M2 | CI/CD | MEDIUM | `regression.yml:45-48` |
| M3 | Build | MEDIUM | `apps/backend/Dockerfile:18` |
| M4 | Connectors | MEDIUM | `postgres.go:111`, `mysql.go:82`, etc. |
| M5 | Connectors | MEDIUM | All DB connectors |
| M6 | Connectors | MEDIUM | `mysql.go:70` |
| M7 | Scanner | MEDIUM | `orchestrator.go` |
| M8 | K8s | MEDIUM | `values.yaml` |
| M9 | K8s | MEDIUM | `values.yaml:140` |
| M10 | K8s | MEDIUM | `platform.yaml:19-41` |
| M11 | Observability | MEDIUM | `otel.go:33` |
| M12 | Observability | MEDIUM | `k8s/` (missing ServiceMonitors) |
| M13 | Observability | MEDIUM | `servicemonitor-celery.yaml` |
| M14 | Migrations | MEDIUM | `000039_backfill_confidence_scores.up.sql:37` |

**Total: 6 CRITICAL, 21 HIGH, 14 MEDIUM**

---

*Generated by parallel multi-agent audit. All findings verified against actual source files.*
