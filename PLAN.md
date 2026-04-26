# ARC-HAWK-DD Production Readiness Plan

## Goal

Make ARC-HAWK-DD production-ready. The platform handles sensitive PII classification for enterprise data discovery. Zero tolerance for security gaps, data races, or broken contracts.

## Context

5 parallel skill sessions completed:

1. **Code Quality/Security**: Found swallowed errors, missing input validation, SQL injection risks, wrong HTTP status codes, missing rate limiting on scan trigger + evidence package endpoints, HTTP client timeouts missing on all external calls, goroutine lifecycle issues, hardcoded credential scan.

2. **UI/UX Overhaul**: Implemented empty states (Scans, Findings, Connectors, DPR, Consent), loading skeletons on all async tables, error toast notifications, accessibility fixes (aria-labels, form labels, color-only indicators), confirmation dialogs for destructive actions, tooltips on DPDP section IDs + PII codes.

3. **Feature Gaps**: Built scan scheduler UI + backend, findings CSV export, connector health widget, risk score trend chart (Recharts time-series), notification settings page + backend CRUD.

4. **DevEx/Ship Readiness**: Fixed docker-compose healthchecks + resource limits, SIGTERM graceful shutdown, missing .down.sql stubs, .env.example completeness. Created Makefile, .golangci.yml, VS Code launch.json + extensions.json, CHANGELOG.md, updated README.

5. **Tests/Benchmarks**: Table-driven Go unit tests for FeedbackService, ObligationRegressionDetector, ScanWatchdog, all HTTP handlers. goScanner tests for circuit breaker + Indian PII patterns (Aadhaar, PAN, VoterID, Passport, IFSC). Playwright E2E tests. Go benchmark suite. BENCHMARKS.md.

## Critical Vulnerabilities Found (Audit)

### CRITICAL

- **C-1**: Supermemory API key `sm_XXDSFvG3GsPdsgJQv4AQ7t_*` exposed in .env — active key, needs rotation NOW
- **C-2**: ~~101 endpoints return `err.Error()` directly to HTTP clients~~ **FIXED** — commit 274f1b4, 31 handler files migrated to `interfaces.NewErrorResponse`
- **C-3**: `VAULT_DEV_ROOT_TOKEN=dev-root-token-arc-hawk` committed to .env
- **C-4**: `SCANNER_SERVICE_TOKEN=dev-scanner-token-change-me` — placeholder token allows scanner impersonation
- **C-5**: ~~`AUTH_REQUIRED=false` in default .env~~ **FIXED** — main.go fatals in release mode if AUTH_REQUIRED != "true"

### HIGH

- **H-1**: ~~19+ unmanaged goroutines~~ **FIXED** — context propagation + bounded goroutines in commit 274f1b4
- **H-2**: ~~Auth middleware `go func()` for api_keys.last_used_at — unbounded goroutines~~ **FIXED** — commit 274f1b4
- **H-3**: ~~Neo4j sync worker race condition~~ **FIXED** — commit 274f1b4, status updates moved inside transaction
- **H-4**: ~~Unencrypted PII fallback path~~ **FIXED** — commit 6dee383, default to `[ENCRYPTION_UNAVAILABLE]`
- **H-5**: PII metadata sent to Supermemory.ai — tenant IDs and source types still in tags/metadata
- **H-6**: ~~No rate limiting on /auth/login, /auth/register, /auth/refresh~~ **FIXED** — StrictRateLimiter on public group in auth/module.go
- **H-7**: ~~HTTP ingest requests in goScanner use `context.Background()` with no timeout~~ **FIXED** — commit 274f1b4
- **H-8**: Silent `_, _ = db.ExecContext(...)` — 3 non-trivial instances remain: `risk_engine.go:212`, `feedback_service.go:131`, `obligation_regression.go:56`

### DATA INTEGRITY

- **D-1**: ~~Neo4j sync worker: 6 ignored errors~~ **FIXED** — commit 274f1b4, stale row recovery + error handling hardened
- **D-2**: ~~No rollback on Neo4j queue commit failure~~ **FIXED** — transactional outbox in ingestion_service.go
- **D-3**: ~~No distributed transaction for Postgres + Neo4j dual-writes~~ **RESOLVED** — transactional outbox pattern + saga in remediation_service.go cover the main flows
- **D-4**: ~~Missing context timeout on neo4j_sync_worker.go processBatch~~ **FIXED** — commit 274f1b4

### DEAD CODE / PHANTOM CONTRACTS

- **Dead endpoints**: `/agent/*` not registered. `/masking/*` and `/fplearning/*` have 23 frontend references — not dead, active modules.
- **E2E coverage**: Playwright tests cover scans, findings, connectors, dashboard UI interactions. Missing: full auth login→dashboard flow, form submission with real data persistence.
- **Testcontainers**: Infrastructure exists (`testutil/db_container_test.go`) but used only for smoke test, not actual service tests.
- **testid mismatch**: E2E `export-csv-btn` testid but frontend uses `export-findings-btn` — test will fail.

## Remaining Production Gaps

### P0 — Ship Blockers

1. **[OPEN]** Rotate exposed Supermemory API key (C-1) + VAULT_DEV_ROOT_TOKEN (C-3) + SCANNER_SERVICE_TOKEN (C-4) — operational task, scrub from git history
2. ~~Enforce AUTH_REQUIRED=true in all deployments~~ **DONE** — main.go:106-109 fatals in release mode
3. ~~Sanitize ALL error responses~~ **DONE** — commit 274f1b4
4. ~~Fix Neo4j sync worker race condition~~ **DONE** — commit 274f1b4
5. ~~Bound the api_keys.last_used_at goroutine~~ **DONE** — commit 274f1b4
6. ~~Add context timeouts to all HTTP ingest requests in goScanner~~ **DONE** — commit 274f1b4
7. ~~Handle PII encryption fallback — fail hard, don't store plaintext~~ **DONE** — commit 6dee383

### P1 — Quality Gates

8. ~~Rate limit /auth endpoints~~ **DONE** — StrictRateLimiter on login/register/refresh
9. ~~Fix remaining silent `_, _ =`~~ **DONE** — commit 3573743: risk_engine, feedback_service, obligation_regression all log errors now
10. ~~Add distributed transaction or saga for Postgres + Neo4j dual-writes~~ **DONE** — outbox + saga pattern in place
11. ~~Remove dead endpoints (/masking/*, /fplearning/*, /agent/*)~~ **NOT REQUIRED** — masking+fplearning have 23 frontend refs; /agent/* not registered
12. ~~Add data-testid attributes to all critical UI elements~~ **DONE** — 168 testids across app pages
13. ~~Fix E2E testid mismatch: `export-csv-btn` vs `export-findings-btn`~~ **DONE** — commit 3573743
14. ~~Add E2E login → dashboard auth flow~~ **DONE** — commit a5bb915: login page at /login with full data-testid coverage; auth.setup.ts Playwright fixture; 5 login.spec.ts tests with API route mocking

### P2 — Hardening

15. ~~Add fuzzing tests for Indian PII validators~~ **DONE** — commit 3573743: FuzzValidateAadhaar, FuzzValidatePAN, FuzzValidateVoterID, FuzzValidateIFSC, FuzzValidatePassport with structural invariant assertions
16. ~~Convert mock DB tests to testcontainers integration tests~~ **DONE** — commit a5bb915: neo4jSyncer interface extracted; 5 integration tests (success, failure, dead-letter, stale row recovery, empty queue) using testcontainers Postgres; NewTestDB moved to importable non-test file
17. ~~Add `-race` flag to CI test runs~~ **DONE** — commit 3573743: `make test-race` runs both backend and scanner with -race
18. ~~Add context timeout to neo4j_sync_worker.go QueryContext~~ **DONE** — commit 274f1b4
19. ~~PII metadata redaction before Supermemory.ai~~ **DONE** — commit 3573743: tenant_id replaced with sha256[:12] hash in all tags/metadata; source types count only in content

## Architecture

- **Backend**: Go + Gin, PostgreSQL, Neo4j, Redis (rate limiting)
- **goScanner**: 36 connectors (S3, GCS, Azure, databases, SaaS), Presidio NER integration
- **Frontend**: React + TypeScript, Recharts for analytics
- **Auth**: JWT, API keys, scanner service token
- **Infra**: Docker Compose, K8s (Helm), Prometheus/Grafana

## Success Criteria

- [x] Zero unhandled goroutines at shutdown
- [x] Zero err.Error() in HTTP responses
- [x] AUTH_REQUIRED enforced — no anonymous access in release mode
- [ ] All secrets rotated and removed from git history
- [x] Neo4j sync worker passes race condition integration test
- [ ] E2E tests cover: login, connector creation, scan trigger, findings view, CSV export
- [ ] CI runs with -race flag, no data races reported
- [x] All critical endpoints have rate limiting
