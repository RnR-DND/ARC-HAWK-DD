# ARC-HAWK-DD Tech Stack Deep Report
*Generated: 2026-04-24*
*Auditor: Claude Sonnet 4.6 — all claims grounded in actual code reads*

---

## 1. Languages & Runtimes

| Language | Version | Location |
|----------|---------|----------|
| Go (backend) | 1.25.0 | `apps/backend/go.mod` |
| Go (goScanner) | 1.22 | `apps/goScanner/go.mod` |
| TypeScript / React | React 19.2.4, Next.js 16.1.6 | `apps/frontend/package.json` |
| Node.js (build/lint) | 20 (CI pinned) | `.github/workflows/ci-cd.yml` |
| SQL (PostgreSQL) | 15 (Docker), 16 (CI service) | `docker-compose.yml`, `ci-cd.yml` |

---

## 2. Core Infrastructure

### PostgreSQL
- **Status:** ✅ FULLY INTEGRATED
- **Version:** 15-alpine (Docker), driver `github.com/lib/pq v1.10.9`
- **Where initialized:** `apps/backend/cmd/server/main.go` — `database.Connect(dbConfig)` called unconditionally at startup; migrations run via `golang-migrate/migrate/v4` immediately after
- **Env var required:** `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT` (all required — validated at startup with `validateRequiredEnvVars()`)
- **What works:** Connection pool, automated schema migrations (versioned), all module repositories, RLS session variable support, integration tests via testcontainers
- **What's missing:** Nothing blocking — `DB_SSLMODE=disable` is blocked in release mode

---

### Neo4j
- **Status:** ✅ FULLY INTEGRATED
- **Version:** 5.15-community (Docker), driver `github.com/neo4j/neo4j-go-driver/v5 v5.28.4`
- **Where initialized:** `apps/backend/cmd/server/main.go` — `persistence.NewNeo4jRepository(...)` called unconditionally; `NEO4J_PASSWORD` is a hard-required env var (process fatals if missing)
- **Env var required:** `NEO4J_URI`, `NEO4J_USERNAME`, `NEO4J_PASSWORD` (all required)
- **What works:** Driver connection established at startup, `/readyz` probe checks `VerifyConnectivity`, `SyncFindingsToPIICategories` writes Asset→PII_Category edges, Neo4jSyncWorker drains `neo4j_sync_queue` outbox every 5s
- **What's missing:** `bolt://` URI is blocked in release mode (must use `bolt+ssc://` or `neo4j+s://`); APOC plugin auto-installed

---

### Redis
- **Status:** 🔧 PARTIALLY INTEGRATED
- **Version:** `github.com/redis/go-redis/v9 v9.18.0` (backend), `v9.5.1` (goScanner)
- **Where initialized:** NOT in `main.go` for normal operation. Redis is wired ONLY inside `ScanActivities` (the Temporal activity worker at `modules/scanning/activities/scan_activities.go`), which is only instantiated when `TEMPORAL_ENABLED=true`. The backend main.go has zero Redis initialization calls.
- **Env var required:** `REDIS_ADDR` (defaults to `localhost:6379`), `REDIS_PASSWORD` — only consumed if Temporal is enabled
- **What works:** XREADGROUP/XACK streaming fully implemented in `RunStreamingWindowActivity`; checkpoint persistence in `PersistStreamingCheckpoints`; Redis connector in goScanner for scanning Redis key-space; `testRedis` in connection test service
- **What's missing:** Redis is NOT initialized in the main server path. It only activates when both `TEMPORAL_ENABLED=true` AND the `StreamingSupervisorWorkflow` is triggered. No standalone Redis service in `docker-compose.yml` — the streaming pipeline has no container to connect to by default.

---

### Temporal
- **Status:** ⚡ INTEGRATED, ENV-GATED
- **Version:** `go.temporal.io/sdk v1.25.0`; server image `temporalio/auto-setup:1.22.0`
- **Where initialized:** `apps/backend/cmd/server/main.go` lines 296–319 — gated by `TEMPORAL_ENABLED=true` (default `false` per `.env.example`)
- **Env var required:** `TEMPORAL_ENABLED=true`, `TEMPORAL_HOST_PORT` (default `localhost:7233`), `TEMPORAL_TLS_ENABLED=true` in release
- **What works:** `TemporalWorker` fully implemented (`modules/scanning/worker/temporal_worker.go`); three workflows defined (`ScanLifecycleWorkflow`, `RemediationWorkflow`, `PolicyEvaluationWorkflow`) plus `StreamingSupervisorWorkflow` for long-running streaming scans; graceful shutdown hooked; worker stopped on process exit
- **What's missing:** Disabled by default. In the default docker-compose setup Temporal container IS running, but the backend worker does not connect to it. Must explicitly set `TEMPORAL_ENABLED=true`.

---

### HashiCorp Vault
- **Status:** ⚡ INTEGRATED, ENV-GATED
- **Version:** `github.com/hashicorp/vault/api v1.23.0`; server image `hashicorp/vault:1.15`
- **Where initialized:** `apps/backend/cmd/server/main.go` lines 238–248 — `vault.NewClient()` always called; internally returns a no-op client when `VAULT_ENABLED != "true"` (docker-compose default: `VAULT_ENABLED=${VAULT_ENABLED:-false}`)
- **Env var required:** `VAULT_ENABLED=true`, `VAULT_ADDR`, `VAULT_TOKEN` (or AppRole: `VAULT_ROLE_ID` + `VAULT_SECRET_ID`), `VAULT_SECRET_MOUNT`
- **What works:** Full KV v2 implementation — `WriteConnectionSecret`, `ReadConnectionSecret`, `DeleteConnectionSecret`, `HealthCheck`; AppRole auth supported; passed as `baseDeps.VaultClient` to all modules; health check surfaced via `/api/v1/health/components`; release-mode enforces HTTPS addr + non-dev token
- **What's missing:** Disabled by default. Falls back to PostgreSQL AES-256 encryption when disabled.

---

### Microsoft Presidio
- **Status:** ✅ FULLY INTEGRATED
- **Version:** `mcr.microsoft.com/presidio-analyzer:latest` (Docker); no Go module — HTTP client only
- **Where initialized:** `apps/goScanner/api/handler.go` (ScanHandler reads `PRESIDIO_URL` env and constructs a `presidio.Client`); Presidio service defined in `docker-compose.yml` at `172.31.240.12:3000`
- **Env var required:** `PRESIDIO_URL` (default `http://presidio-analyzer:3000`)
- **What works:** Full NER integration in orchestrator (`orchestrator.go`); circuit breaker wrapping every call (`presidio_client.go` — trips after 3 consecutive failures, 30s timeout); three classification modes: `regex`, `ner`, `contextual`; Indian PII ad-hoc recognizers (IN_PAN, IN_AADHAAR etc.); custom pattern delegation to Presidio; Presidio latency histogram in Prometheus (`arc_hawk_presidio_latency_seconds`)
- **What's missing:** Presidio runs when `ClassificationMode` is `ner` or `contextual` — if mode is `regex` (which may be the default for some scan profiles), Presidio is skipped. Backend docker-compose passes `PRESIDIO_ADDR` to backend but backend itself does not call Presidio directly; it's the goScanner that calls it.

---

### OpenTelemetry (Tracing)
- **Status:** ⚡ INTEGRATED, ENV-GATED
- **Version:** `go.opentelemetry.io/otel v1.41.0` (backend), `v1.24.0` (goScanner)
- **Where initialized:** `apps/backend/cmd/server/main.go` line 91 — `telemetry.InitTracer(...)` called unconditionally; `apps/goScanner/cmd/scanner/main.go` line 26 — same pattern. Both fall back to no-op when `OTEL_EXPORTER_OTLP_ENDPOINT` is unset.
- **Env var required:** `OTEL_EXPORTER_OTLP_ENDPOINT` (unset = no-op); optional: `OTEL_INSECURE=true`, `OTEL_TRACE_SAMPLE_RATE` (default 0.1)
- **What works:** OTLP/gRPC exporter; `otelgin.Middleware` attached to both Gin routers (W3C traceparent propagation); Orchestrator creates spans per scan and per Presidio call; default sample rate 10%
- **What's missing:** No OTEL collector defined in docker-compose. Traces are silently dropped unless operator configures `OTEL_EXPORTER_OTLP_ENDPOINT`.

---

### Prometheus + Grafana
- **Status:** ✅ FULLY INTEGRATED
- **Version:** `github.com/prometheus/client_golang v1.23.2` (backend), `v1.20.0` (goScanner); Prometheus `prom/prometheus:latest`; Grafana `grafana/grafana:latest`
- **Where initialized:** Backend: `/metrics` endpoint in `main.go` lines 365–383 (manual Go runtime metrics — goroutines, alloc, sys, module count). GoScanner: `promhttp.Handler()` on `/metrics` route. Both services defined in docker-compose. SLO recording rules in `infra/prometheus/slo_rules.yml`.
- **Env var required:** `GRAFANA_ADMIN_PASSWORD`
- **What works:** Metrics exposed on both services; SLO alert rules for scan success rate and Presidio latency; `arc_hawk_scan_findings_total` counter (by pii_type + source_type), `arc_hawk_active_scans` gauge, `arc_hawk_presidio_latency_seconds` histogram; `neo4j_sync_dead_letter_total` counter; Prometheus scrapes both `/metrics` endpoints
- **What's missing:** Backend `/metrics` endpoint uses manual `fmt.Fprintf` rather than `promhttp.Handler()` — only 3 runtime metrics exposed, not the full prometheus/client_golang registry (scan and Presidio metrics are exported by the goScanner, not the backend). No Grafana dashboards committed to the repo.

---

### Scan Watchdog
- **Status:** ✅ FULLY INTEGRATED
- **Version:** Internal implementation (`modules/scanning/service/scan_watchdog.go`)
- **Where initialized:** `apps/backend/cmd/server/main.go` line 184 — `scanningservice.NewScanWatchdog(db, 5*time.Minute)` called unconditionally; `Start(serverCtx)` called immediately
- **Env var required:** None
- **What works:** Background goroutine ticking every 5 minutes; marks `scan_runs` rows stuck in `running` status for >2 hours as `failed`; logs count of reaped scans; stopped via context cancellation on shutdown. The 5-minute interval passed from main.go overrides the internal `stalledScanThreshold` (2 hours) which is the DB cutoff — both are effectively active.
- **What's missing:** Nothing critical. Note that `ScanningModule.Initialize()` also starts its own timeout checker goroutine every 5 minutes — both the watchdog and the module-level ticker run simultaneously (minor duplication).

---

### Neo4j Sync Worker (Transactional Outbox)
- **Status:** ✅ FULLY INTEGRATED
- **Version:** Internal (`modules/shared/infrastructure/persistence/neo4j_sync_worker.go`)
- **Where initialized:** `apps/backend/cmd/server/main.go` line 179 — `persistence.NewNeo4jSyncWorker(db, neo4jRepo)` called unconditionally; `Start(serverCtx)` called immediately; `Stop()` called on graceful shutdown
- **Env var required:** None
- **What works:** Polls `neo4j_sync_queue` table every 5s; processes up to 50 rows per batch; uses `FOR UPDATE SKIP LOCKED` for safe multi-instance operation; marks rows `processing` before releasing lock (no double-processing); exponential retry up to 5 attempts; rows exceeding 5 attempts move to `dead_letter` status; `neo4j_sync_dead_letter_total` Prometheus counter
- **What's missing:** Only one operation type handled: `sync_findings`. Other operations log "unknown operation" and return nil (no-error swallow).

---

### Transactional Outbox (General)
- **Status:** 🔧 PARTIALLY INTEGRATED
- **What exists:** The Neo4j sync queue is a purpose-built outbox table (`neo4j_sync_queue`) — fully working. `modules/scanning/service/ingestion_service.go` also references outbox patterns.
- **What's missing:** No general-purpose outbox for other event types (audit, remediation, etc.). Audit events are written synchronously to `audit_ledger`. No Redis-based outbox in the main path.

---

### Redis XREADGROUP/XACK Streaming
- **Status:** 🔲 STUB / PLACEHOLDER (production path only via Temporal + ENV gate)
- **Where implemented:** `modules/scanning/activities/scan_activities.go` — `RunStreamingWindowActivity` uses `XREADGROUP` with at-least-once delivery and `XACK` after successful processing
- **Env var required:** `TEMPORAL_ENABLED=true` plus a running Temporal worker AND Redis service (not in docker-compose)
- **What works:** Code is complete and correct — XREADGROUP consumer group creation, batch read, XACK per message, checkpoint persistence via HSet
- **What's missing:** No Redis service in docker-compose; only reachable via the Temporal activity path which requires `TEMPORAL_ENABLED=true` (default false). Not exercised in any default configuration.

---

### Circuit Breaker (gobreaker)
- **Status:** ✅ FULLY INTEGRATED (goScanner only)
- **Version:** `github.com/sony/gobreaker v1.0.0` (goScanner go.mod indirect dependency, directly used)
- **Where initialized:** `apps/goScanner/internal/orchestrator/presidio_client.go` — package-level `var presidioBreaker` initialized at import time
- **Env var required:** None
- **What works:** Presidio circuit breaker trips after 3 consecutive failures; stays open 30 seconds; resets interval 60 seconds; health probe used to distinguish "no PII" from "Presidio unreachable"; graceful degradation to regex-only on open circuit
- **What's missing:** No circuit breaker in backend (not imported in `apps/backend/go.mod`). Breaker only protects Presidio calls, not database or Neo4j calls.

---

### PostgreSQL RLS
- **Status:** 🔧 PARTIALLY INTEGRATED
- **Version:** Implementation at `modules/shared/infrastructure/database/rls.go`
- **Where initialized:** `SetTenantContext` function defined; migration 000053 sets up RLS policies
- **Env var required:** None
- **What works:** `SetTenantContext(ctx, tx, tenantID)` sets `app.current_tenant_id` via `SET LOCAL` — designed for RLS enforcement on `scan_runs`, `findings`, `assets`, `connections`, `audit_logs`, `fp_learning`. RLS policies created in migration 000053.
- **What's missing:** `SetTenantContext` is ONLY defined in `rls.go` — grep confirms it is not called anywhere else in the backend codebase. RLS policies exist in the DB schema but the application never calls `SetTenantContext` to set the session variable. This means RLS is enforced at the DB layer only if the policies work with the default empty role, but the per-request tenant context injection is NOT wired into any request handler or middleware. Application-layer `WHERE tenant_id = $n` filtering is the active isolation mechanism.

---

### Feedback / Bayesian Loop
- **Status:** ✅ FULLY INTEGRATED
- **Version:** Internal (`modules/scanning/service/feedback_service.go`)
- **Where initialized:** `ScanningModule.Initialize()` line 132 — `service.NewFeedbackService(deps.DB)` wired; `FeedbackHandler` registered; routes exposed: `POST /findings/:id/feedback`, `GET /patterns/precision`
- **Env var required:** None
- **What works:** `RecordCorrection` writes to `feedback_corrections` table; async `maybeAdjustPatternThreshold` goroutine fires after each correction; Bayesian delta computed as `(precision - 0.7) * 100`; threshold adjusted once ≥10 corrections exist (30–90% clamp); upserts into `pattern_confidence_overrides`; `GetPatternPrecisionStats` returns per-pattern stats with confidence delta
- **What's missing:** Nothing critical. Threshold adjustment runs in a fire-and-forget goroutine (error silently ignored).

---

### Audit Ledger
- **Status:** ✅ FULLY INTEGRATED
- **Version:** Two implementations: `audit.LedgerLogger` (compliance events → `audit_ledger` table) and `audit.PostgresAuditLogger` (general activity → `audit_logs` table)
- **Where initialized:** In `main.go` line 196 — `auditLogger := audit.NewPostgresAuditLogger(auditRepo)` wired into `baseDeps.AuditLogger`. `LedgerLogger` initialized in `ComplianceModule.Initialize()` and in `ScanningModule.Initialize()` (via `m.ingestionService.SetLedger(...)`)
- **Env var required:** None
- **What works:** Immutable append-only `audit_ledger` table; 11 event types (scan completed, PII discovered, consent granted/revoked, DPR submitted/resolved, remediation applied, policy evaluated, cross-border transfer, GRO escalation, evidence package generated); scan completion hook wired in compliance module triggers audit ledger write + obligation regression detection; `GET /compliance/audit-trail` endpoint
- **What's missing:** `audit_logs` (general activity) and `audit_ledger` (compliance events) are separate tables — both active. IP address capture requires `middleware.IPContextMiddleware()` to populate context (wired in main.go).

---

### Evidence Package ZIP
- **Status:** ✅ FULLY INTEGRATED
- **Version:** Internal (`modules/compliance/service/evidence_package.go`)
- **Where initialized:** `ComplianceModule.Initialize()` line 67 — `compsvc.NewEvidencePackageService(deps.DB, m.ledgerLogger)`; handler `evidenceHandler` registered; route: `POST /api/v1/compliance/evidence-package`
- **Env var required:** None
- **What works:** Generates 9-section ZIP archive (README + 8 JSON files): obligation scorecard, DPR log, GRO details, consent records, scan history, remediation actions, audit trail (last 90 days), PII categories; generation event logged to audit_ledger; filename: `dpdp_evidence_{tenantID_prefix}_{date}.zip`
- **What's missing:** Nothing. Fully wired end-to-end from HTTP handler to ZIP response.

---

### Obligation Regression Detector
- **Status:** ✅ FULLY INTEGRATED
- **Version:** Internal (`modules/compliance/service/obligation_regression.go`)
- **Where initialized:** `ComplianceModule.Initialize()` line 66 and 86 — wired via scan completion hook into `ScanningModule`; fires automatically after every completed scan
- **Env var required:** None
- **What works:** Compares current scan PII categories against all prior completed scans for the tenant; emits `EventPIIDiscovered` audit event per new category; upserts into `obligation_regressions` table; `ON CONFLICT (tenant_id, pii_category) DO UPDATE` prevents duplicates
- **What's missing:** Nothing. Fully automatic — no manual invocation needed.

---

## 3. Data Source Connectors (goScanner)

All connectors use a self-registration pattern (`init()` in each package); `main.go` imports each package with blank identifiers.

### Always-registered (default build)

| Connector | Source Type Key | Status | Sampling | SSL/TLS |
|-----------|----------------|--------|----------|---------|
| PostgreSQL | `postgresql` | ✅ Real code | `TABLESAMPLE BERNOULLI` + `LIMIT` (configurable 50–50000, default 1000) | `sslmode=prefer` default; `DB_SSLMODE_DEFAULT` env override |
| MySQL | `mysql` | ✅ Real code | `LIMIT sample_size` | `tls=preferred` default |
| MongoDB | `mongodb` | ✅ Real code | `Sample` aggregation stage | TLS configurable via DSN |
| Redis | `redis` | ✅ Real code | SCAN cursor up to `scan_size` | Password auth only; no TLS config |
| SQLite | `sqlite` | ✅ Real code | `LIMIT` | N/A (file-based) |
| MSSQL | `mssql` | ✅ Real code | `TOP` clause | TLS via `encrypt` param |
| Firebase Firestore | `firebase` | ✅ Real code | `Limit(sample_size)` per collection | Google ADC (service account JSON) |
| CouchDB | `couchdb` | ✅ Real code | `_all_docs` with `limit` | HTTP/HTTPS via URL scheme |
| Amazon S3 | `s3` | ✅ Real code | Random object sampling with limit | AWS SDK (IAM/credentials) |
| Google Cloud Storage | `gcs` | ✅ Real code | Object listing + content scan | Google ADC |
| Filesystem | `filesystem` | ✅ Real code | Walk with size limits | N/A |
| Text/CSV/Excel | `text`, `csv_excel` | ✅ Real code | Row-limit per file | N/A |
| HTML files | `html_files` | ✅ Real code | File-based | N/A |
| PDF | `pdf` | ✅ Real code | Page limit | N/A |
| DOCX / PPTX | `docx`, `pptx` | ✅ Real code | Full document | N/A |
| Email files (.eml) | `email_files` | ✅ Real code | Full message | N/A |
| Avro | `avro` | ✅ Real code | Record limit | N/A |
| Parquet | `parquet` | ✅ Real code | Row group sampling | N/A |
| ORC | `orc` | ✅ Real code | Stripe-based | N/A |
| Kafka | `kafka` | ✅ Real code | Window-based consumer | SASL/TLS configurable |
| Kinesis | `kinesis` | ✅ Real code | GetRecords limit | AWS SDK |
| Slack | `slack` | ✅ Real code | Channel history + DMs | Slack Bot Token |
| Jira | `jira` | ✅ Real code | Issue JQL pagination | HTTPS (Jira API) |
| Salesforce | `salesforce` | ✅ Real code | SOQL `LIMIT` | OAuth2 |
| HubSpot | `hubspot` | ✅ Real code | Contact/deal pagination | API key |
| Microsoft Teams | `ms_teams` | ✅ Real code | Graph API pagination | OAuth2 |
| BigQuery | `bigquery` | ✅ Real code | `TABLESAMPLE SYSTEM` | Google ADC |
| Snowflake | `snowflake` | ✅ Real code | `SAMPLE (n ROWS)` | Snowflake DSN |
| Redshift | `redshift` | ✅ Real code | `TABLESAMPLE BERNOULLI` | sslmode from config |

### Stub-only (requires `connector_stub` build tag — NOT in default binary)

| Connector | Source Type Key | Status | Notes |
|-----------|----------------|--------|-------|
| Oracle | `oracle` | 🔲 STUB | Requires Oracle Instant Client + `godror` (CGO); excluded from default image |
| Scanned Images (OCR) | `scanned_images` | 🔲 STUB | Listed in `register_stubs.go` for files package |
| Google Drive (personal) | `gdrive` | 🔲 STUB | Cloud stub — needs OAuth scope |
| Google Drive Workspace | `gdrive_workspace` | 🔲 STUB | Cloud stub — needs domain-wide delegation |
| Azure Blob Storage | `azure_blob` | 🔲 STUB | Cloud stub — needs Azure SDK import |

---

## 4. Frontend Libraries

All from `apps/frontend/package.json`. Verified by checking usage patterns:

| Library | Version | Status |
|---------|---------|--------|
| Next.js | 16.1.6 | ✅ Core framework — `next dev/build/start` in scripts |
| React | 19.2.4 | ✅ Core framework |
| @radix-ui/* (avatar, dialog, dropdown, scroll-area, separator, slot) | ^1.x / ^2.x | ✅ Used in UI components — Radix primitives are the component foundation |
| Tailwind CSS | 3.4.17 | ✅ Styling system — autoprefixer and tailwindcss-animate also present |
| class-variance-authority + clsx + tailwind-merge | various | ✅ Standard shadcn/ui variant system |
| lucide-react | 0.562.0 | ✅ Icon set — used throughout UI |
| axios | 1.6.2 | ✅ HTTP client for API calls |
| recharts | 3.7.0 | ✅ Chart library — dashboards and analytics pages |
| reactflow | 11.10.3 | ✅ Node graph — lineage visualization |
| cytoscape + dagre | 3.33.1 / 0.8.5 | ✅ Graph layout — alternative to reactflow for some views |
| framer-motion | 12.28.1 | ✅ Animation — UI transitions |
| date-fns | 4.1.0 | ✅ Date formatting |
| @playwright/test | 1.57.0 | ✅ E2E tests (devDep) — `test:e2e` script defined |
| @testing-library/react + jest | 13.4.0 / 27.5.1 | ✅ Unit tests (devDep) — `test` script defined |

Note: `--legacy-peer-deps` required in npm install because `@testing-library/react@13` declares `react@^18` peer dep but project uses react@19. This is flagged explicitly in CI comments.

---

## 5. CI/CD Pipelines

### `build.yml` (trigger: push/PR to main)
- **Jobs:** `backend` (vet + test + build), `scanner` (vet + test + build), `frontend` (install + lint + build)
- **Tests run:** `go test ./... -short -skip "TestContainerSmokeTest|TestTruncateAll"` for backend; `go test ./... -short` for scanner; `npm run lint` for frontend
- **Notes:** Basic CI gate — ensures code compiles and unit tests pass

### `ci-cd.yml` (trigger: push to main/develop, PR to main, manual dispatch)
- **Jobs:** `lint-backend`, `lint-frontend`, `lint-go-scanner`, `build-backend`, `build-frontend`, `build-go-scanner`, `integration-tests`, `push-images` (main only), `deploy` (manual dispatch only)
- **Integration tests:** Runs with a live PostgreSQL 16 service container; skips testcontainers tests (`TestContainerSmokeTest|TestTruncateAll`); sets `NEO4J_ENABLED=false`
- **Image push:** Only on push to main — pushes to DockerHub using `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` secrets
- **Deploy:** Via SSH action to `/opt/arc-hawk` on a server — manual trigger only, with environment choice (staging/production)
- **Auth safety gate:** CI step fails if `AUTH_REQUIRED=false` + `GIN_MODE=release` found in .env files

### `security.yml` (trigger: push/PR to main, weekly Monday 08:00 UTC)
- **Jobs:** `gosec`, `govulncheck`, `container-scan` (Trivy), `dependency-review` (PR only)
- **gosec:** Runs on both backend and scanner; results uploaded as SARIF; **non-blocking** (`|| true` — advisory mode)
- **govulncheck:** Runs on both backend and scanner; **non-blocking** (`|| true` — advisory mode)
- **Trivy:** Scans backend Docker image; `exit-code: '0'` — **advisory mode**, does not block builds; SARIF uploaded to GitHub Security tab
- **dependency-review:** Fails on critical severity — this IS blocking for PRs

### `regression.yml` (trigger: push/PR to main/develop)
- **Jobs:** `regression` — classifier tests, connector tests (short), orchestrator tests, classifier quality gate
- **Quality gate:** Passes if classifier test pass rate >= 90%; computed inline with awk

---

## 6. Security Tools

| Tool | Configured | Blocking | Notes |
|------|-----------|----------|-------|
| gosec | ✅ Yes — `security.yml` | ❌ No (`\|\| true`) | SARIF uploaded to GitHub Security; advisory only |
| govulncheck | ✅ Yes — `security.yml` | ❌ No (`\|\| true`) | Advisory only |
| Trivy (container) | ✅ Yes — `security.yml` | ❌ No (`exit-code: '0'`) | Scans backend image for CRITICAL/HIGH, ignores unfixed; SARIF uploaded |
| dependency-review | ✅ Yes — `security.yml` | ✅ Yes | Fails PRs on critical severity new dependencies |
| Auth safety gate | ✅ Yes — `ci-cd.yml` | ✅ Yes | Fails if AUTH_REQUIRED=false + GIN_MODE=release in .env |
| Release env guards | ✅ Yes — `main.go` | ✅ Yes at runtime | Enforces TLS, key length, non-dev tokens, Temporal TLS in release mode |

**Key finding:** gosec, govulncheck, and Trivy all run in advisory mode with `|| true` / `exit-code: '0'`. Security findings do not block merges — they surface in GitHub Security tab only.

---

## 7. Summary Table

| Tool | Version | Status | Key Notes |
|------|---------|--------|-----------|
| PostgreSQL | 15 (Docker) | ✅ FULLY INTEGRATED | Required at startup; migrations auto-applied |
| Neo4j | 5.15-community | ✅ FULLY INTEGRATED | Required at startup; NEO4J_PASSWORD fatal if missing |
| Redis | go-redis v9 | 🔧 PARTIALLY INTEGRATED | Exists in Temporal activities + goScanner connector; NOT in main server path; no docker-compose service |
| Temporal | SDK v1.25.0 | ⚡ INTEGRATED, ENV-GATED | TEMPORAL_ENABLED=false by default; full workflow code written |
| HashiCorp Vault | API v1.23.0 | ⚡ INTEGRATED, ENV-GATED | VAULT_ENABLED=false by default; AppRole + Token auth; falls back to PG encryption |
| Microsoft Presidio | HTTP (no SDK) | ✅ FULLY INTEGRATED | Circuit-broken; 3 classification modes; Indian PII recognizers |
| OpenTelemetry | v1.41.0 (backend) | ⚡ INTEGRATED, ENV-GATED | No-op without OTEL_EXPORTER_OTLP_ENDPOINT; no collector in docker-compose |
| Prometheus | client_golang v1.23.2 | ✅ FULLY INTEGRATED | Both services expose /metrics; SLO alert rules committed |
| Grafana | latest | ✅ FULLY INTEGRATED | Running in docker-compose; no dashboards committed to repo |
| Scan Watchdog | Internal | ✅ FULLY INTEGRATED | Always-on background goroutine; 5-min tick; 2h stall threshold |
| Neo4j Sync Worker | Internal | ✅ FULLY INTEGRATED | Outbox pattern; at-least-once; dead-letter after 5 attempts |
| Transactional Outbox | Internal (neo4j_sync_queue) | 🔧 PARTIALLY INTEGRATED | Only Neo4j sync outbox; no general outbox |
| Redis XREADGROUP/XACK | go-redis v9 | 🔲 STUB / PLACEHOLDER | Code complete; only reachable via Temporal (disabled by default) + missing Redis service |
| Circuit Breaker (gobreaker) | v1.0.0 | ✅ FULLY INTEGRATED | goScanner only; protects Presidio calls; not in backend |
| PostgreSQL RLS | Internal | 🔧 PARTIALLY INTEGRATED | SetTenantContext defined but never called in request handlers; DB policies exist but app-side injection is missing |
| Feedback / Bayesian Loop | Internal | ✅ FULLY INTEGRATED | Routes registered; Bayesian threshold auto-adjustment after ≥10 corrections |
| Audit Ledger | Internal | ✅ FULLY INTEGRATED | Two tables: audit_logs + audit_ledger; compliance events + general activity |
| Evidence Package ZIP | Internal | ✅ FULLY INTEGRATED | 9-section DPDP Act evidence ZIP; HTTP endpoint live |
| Obligation Regression Detector | Internal | ✅ FULLY INTEGRATED | Auto-fires via scan completion hook; new PII category → audit event |
| gosec | latest | ✅ Configured, NON-BLOCKING | Advisory mode (`\|\| true`) |
| govulncheck | latest | ✅ Configured, NON-BLOCKING | Advisory mode (`\|\| true`) |
| Trivy | aquasecurity/trivy-action | ✅ Configured, NON-BLOCKING | exit-code 0; advisory |
| Go (backend) | 1.25.0 | ✅ | All CI steps use 1.25 |
| Go (goScanner) | 1.22 | ✅ | go.mod declares 1.22; CI also uses 1.25 |
| React + Next.js | 19.2.4 + 16.1.6 | ✅ | Full stack frontend |
