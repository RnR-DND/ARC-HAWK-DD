# TODO — ARC-HAWK-DD

Active backlog. Ordered by priority within each tier.

---

## User-Action Required (cannot be automated)

- [ ] **Rotate Supermemory API key** at https://app.supermemory.ai — old key `sm_XXDSFvG3…` was exposed in chat logs
- [ ] **Generate `SCANNER_SERVICE_TOKEN`**: `openssl rand -hex 32` — set in both backend `.env` and scanner env
- [ ] **Neo4j TLS cert** — set `NEO4J_URI=bolt+ssc://neo4j:7687` in docker-compose and provision cert (P1-5)
- [ ] **Temporal TLS** — set `TEMPORAL_TLS_ENABLED=true` and provision cert (P1-6)
- [ ] **Merge PR #30**: `gh pr merge 30 --merge`

---

## In-Flight (next session)

- [ ] **WS3: `fplearning` module** — migration 000045, FP corrections CRUD (`POST/GET /fplearning/corrections`, `GET /fplearning/patterns/:id/stats`, `DELETE /fplearning/corrections/:id`), service + repo layer, tenant isolation tests
- [ ] **WS4: OpenAPI via swaggo** — `go get github.com/swaggo/swag/cmd/swag`, `swag init`, annotate ~60 handlers in `apps/backend/modules/*/api/*.go`, wire `/swagger/*any`, add `make openapi` target
- [ ] **WS5 docs (remaining)** — `docs/INTEGRATION_GUIDE.md`, `docs/WEBHOOKS.md`, `docs/RUNBOOK_E2E.md`, `docs/SCANNER_REFERENCE.md`, `docs/releases/v3.0.0.md`, `tests/e2e/full-scan.sh`
- [ ] **WS6 doc sweep (remaining)** — `readme.md`, `docs/architecture/INTEGRATION.md`, `docs/architecture/ARCHITECTURE.md`, `docs/architecture/overview.md`, `DEPLOYMENT_RUNBOOK.md`, `docs/SEAMLESS_SCANNING.md`, `docs/phase1_deployment.md`, `docs/INDEX.md`
- [ ] **WS7: patterns handler test stubs** — implement 6 TODO stubs in `apps/backend/modules/scanning/api/patterns_handler_test.go`
- [ ] **WS7: scan_activities policy stubs** — implement/defer 4 TODOs in `apps/backend/modules/scanning/activities/scan_activities.go`
- [ ] **CHANGELOG.md** — append ~30 commits since 2026-04-09 (through current branch)

---

## P2 — Deferred to Post-Integration Milestones

### Observability
- [ ] Prometheus metrics exporter: goroutines, semaphore depth, vault errors, ingest failures (P2-2, ~1 day)
- [ ] Audit chain integrity cron + `GET /audit/verify` tamper-evident endpoint (P2-3, ~1 day)

### Reliability
- [ ] Testcontainers integration tests for Postgres, MySQL, MongoDB, S3 connectors (P2-4, ~2d/connector)
- [ ] Retry + circuit breaker for every connector `Connect()` / `StreamFields()` (P2-5, ~1 day)
- [ ] Adaptive ingest chunk delay — backpressure from backend response times (P2-6, ~2h)

### Connectors
- [ ] Oracle DB connector — implement fully or remove stub (P2-7, ~30min–1day)

### Frontend
- [ ] Audit page pagination — hardcoded `limit:200` at `app/audit/page.tsx:53` (P2-8, ~3h)
- [ ] Error toasts on mutations — ~50% currently silent `console.error` (P2-9, ~4h)

### Security / Auth
- [ ] Auth gate on Prometheus (9090), Grafana (3002), Temporal UI (8088) (P2-11, ~3h)
- [ ] Per-tenant Neo4j `tenant_id` on every Cypher node + query filter (P2-12, ~2 days)
- [ ] JWT → httpOnly + Secure + SameSite cookies; CSRF tokens; 15min TTL + refresh (P2-13, ~2 days)

### Features
- [ ] `fplearning` ML-based pattern learning (beyond minimal CRUD from WS3) — real frequency learner, pattern auto-tuning
- [ ] Agent sync Redis LPUSH — wire real Redis in `agent_sync_handler.go:225` for async classify pipeline

---

## Won't Fix / Out of Scope

- Root `node_modules/` — deleted; was untracked cruft from wrong `npm` working dir
- `apps/scanner/` Python scanner — retired; Go scanner at `apps/goScanner/` is canonical
- Celery workers — removed from Helm; no Celery code in Go stack
- `infra/k8s/` and `k8s/` flat manifests — consolidated into `helm/arc-hawk-dd/templates/`
