# ARC-HAWK-DD Final Pre-Integration Audit

**Date:** 2026-04-24  
**Auditor:** Automated deep audit (main branch @ a5bb915)  
**Scope:** Go backend + goScanner + React frontend, all production-path code  

---

## Executive Summary

**Integration Readiness: 7.5 / 10**

ARC-HAWK-DD is production-capable for controlled deployment. The codebase compiles cleanly, all 57 database migrations are paired, security hardening is thorough (RLS, JWT, API keys, rate limiting, CORS, security headers, OTel tracing), and the scanner↔backend auth channel is properly implemented. Three issues block unguarded production deployment: the MSSQL connector still has a hardcoded `TOP 10000` (F-04 was not applied), PostgreSQL RLS uses session-level `SET` instead of transaction-level `SET LOCAL` enabling stale tenant context under connection pool reuse, and two `panic()` calls exist in production goroutines in the discovery service. Remediation connectors are explicitly gated behind `REMEDIATION_ENABLED=false` and Temporal is optional/gated. No SQL injection vectors were found outside of properly sanitized identifier-based queries.

---

## Section 1: Build Integrity

| Target | Command | Result |
|---|---|---|
| Backend | `go build ./...` | CLEAN |
| Backend | `go vet ./...` | CLEAN |
| goScanner | `go build ./...` | CLEAN |
| goScanner | `go vet ./...` | CLEAN |
| Frontend | `npm run build` | CLEAN — 23 routes |

**Frontend routes built:** `/`, `/analytics`, `/asset-inventory`, `/assets`, `/assets/[id]`, `/audit`, `/compliance`, `/connectors`, `/discovery`, `/findings`, `/history`, `/lineage`, `/login`, `/posture`, `/profile`, `/regex`, `/remediation`, `/reports`, `/scans`, `/scans/[id]`, `/settings`, `/settings/notifications`, `/users`

No dead imports, no unused symbols flagged by vet. The modules compile as a single binary (modular monolith pattern). goScanner is a separate binary at `apps/goScanner/cmd/scanner/main.go`.

---

## Section 2: API Contracts

### Backend (port 8080)

All `/api/v1/*` routes are wrapped in `authMW.Authenticate()` + `middleware.PolicyMiddleware(db)`.

**Public (no auth):**
- `GET /livez` — Kubernetes liveness probe
- `GET /readyz` — readiness probe (503 if DB or Neo4j down)
- `GET /health` — alias for /readyz
- `GET /metrics` — Prometheus metrics (manual exposition + runtime stats)
- `GET /swagger/*` — Swagger UI (admin-gated in `GIN_MODE=release`)

**Auth module:**
- `POST /api/v1/auth/login` — public
- `POST /api/v1/auth/register` — public
- `POST /api/v1/auth/refresh` — public
- `GET /api/v1/auth/profile` — JWT required
- `POST /api/v1/auth/change-password`
- `GET /api/v1/auth/users`
- `GET/PUT /api/v1/auth/settings`
- `GET/PUT /api/v1/auth/settings/notifications`

**Scanning module:**
- `POST /api/v1/scans/ingest-verified` — scanner callback (X-Scanner-Token)
- `POST /api/v1/scans/trigger` — rate-limited scan trigger
- `GET /api/v1/scans` — list scans
- `GET /api/v1/scans/:id` — get scan
- `GET /api/v1/scans/:id/status`
- `POST /api/v1/scans/:id/complete` — scanner callback
- `POST /api/v1/scans/:id/cancel`
- `DELETE /api/v1/scans/:id`
- `GET/POST/PUT/DELETE /api/v1/scans/patterns` — PII pattern CRUD
- `GET /api/v1/scans/findings/export`
- `POST /api/v1/scans/feedback`
- `GET /api/v1/scans/dashboard`
- `POST /api/v1/scans/agent/sync`
- `GET/POST/PUT/DELETE /api/v1/scans/schedules`

**Compliance module:**
- `GET /api/v1/compliance/overview`
- `GET /api/v1/compliance/violations`
- `GET/POST /api/v1/compliance/dpdpa`
- `GET/POST /api/v1/compliance/consent`
- `GET /api/v1/compliance/dpr`
- `GET/PUT /api/v1/compliance/gro`
- `POST /api/v1/compliance/evidence`
- `GET /api/v1/compliance/audit-trail`
- `GET/POST /api/v1/compliance/retention`
- `GET /api/v1/compliance/audit-logs`

**Connections module:**
- `GET /api/v1/connections/available-types`
- `GET/POST /api/v1/connections` — list/create
- `GET/PUT/DELETE /api/v1/connections/:id`
- `POST /api/v1/connections/:id/test`
- `POST /api/v1/connections/:id/sync`
- `POST /api/v1/connections/scans/scan-all`
- `GET /api/v1/connections/scans/:id/status`
- `GET /api/v1/connections/scans/jobs`

**Discovery module:**
- `GET /api/v1/discovery/overview`
- `GET /api/v1/discovery/inventory`
- `GET /api/v1/discovery/snapshots`
- `GET /api/v1/discovery/risk`
- `GET /api/v1/discovery/drift`
- `GET /api/v1/discovery/reports`
- `GET /api/v1/discovery/glossary`

**Assets module:**
- `GET /api/v1/assets`
- `GET /api/v1/findings`
- `GET /api/v1/dataset/golden`

**Masking module:**
- `POST /api/v1/masking/mask-asset`
- `GET /api/v1/masking/status/:assetId`

**Lineage module:**
- `GET /api/v1/lineage`
- `GET /api/v1/lineage/stats`
- `POST /api/v1/lineage/sync`
- `GET /api/v1/lineage/semantic`

**FP Learning module:**
- `GET/POST /api/v1/fp/false-positives`
- `GET /api/v1/fp/confirmed`
- `GET/POST/PUT/DELETE /api/v1/fp/learnings`
- `GET /api/v1/fp/stats`
- `POST /api/v1/fp/check`

**Analytics module:**
- `GET /api/v1/analytics/...` (heatmaps, trend data)

**Remediation module:**
- `GET/POST/PUT/DELETE /api/v1/remediation/...` (gated behind `REMEDIATION_ENABLED`)

**Memory module:**
- `GET /api/v1/memory/status`
- `POST /api/v1/memory/search`

**WebSocket:**
- `GET /ws` — real-time event stream

**Health component check:**
- `GET /api/v1/health/components` — per-service health (DB, Neo4j, Vault, Presidio)

### goScanner (port 8001)

- `GET /health` — public
- `GET /metrics` — Prometheus (promhttp.Handler)
- `POST /scan` — requires `X-Scanner-Token` header

---

## Section 3: Database Layer

**Migration count:** 57 `.up.sql` files (000001–000057)  
**Pairing:** All 57 have matching `.down.sql` — zero unpaired migrations  
**Auto-migration:** Runs at server startup via `golang-migrate`; server exits on migration failure  
**Latest migration:** `000057_notification_settings`

**Row-Level Security:**
- Enabled on: `findings`, `scan_runs`, `assets` (migration 000047)
- Extended to 4 additional tables (migration 000053)
- Policy uses `current_setting('app.current_tenant_id', true)::uuid`

**Known RLS limitation (documented in `auth_middleware.go:416`):**  
`setRLSTenantContext` issues a session-level `SET`, not `SET LOCAL`. Under connection pool reuse, a connection may carry a stale `tenant_id` to the next request. The code contains a comment acknowledging this. True isolation requires wrapping each handler in an explicit transaction. This is a **P0 issue for multi-tenant production deployments**.

**SQL injection review:**
- Remediation connectors use `sanitizePgIdentifier()` on all table/field names before interpolation — safe
- Dashboard handler builds dynamic WHERE clauses with `fmt.Sprintf` but all values are parameterized — safe
- Scanner connectors use `fmt.Sprintf` for table names from connector metadata (not user input) — acceptable risk, but MSSQL uses `[schema].[table]` bracketed quoting which provides some injection resistance
- No raw user input directly interpolated into SQL found

---

## Section 4: Security

**Authentication:**
- JWT Bearer tokens: validated via `JWTService.ValidateToken`; expired tokens return specific error
- API keys: SHA256-hashed in DB (`api_keys` table, migration 000028); expiry + revocation checked
- Scanner service token: constant-time compare (`crypto/subtle`) in `ServiceTokenAuth`
- `AUTH_REQUIRED=false` permitted only when `GIN_MODE != release`; production refuses to start

**Authorization:**
- `PolicyMiddleware(db)` on all `/api/v1/*` routes
- `RequireRole` / `RequirePermission` / `RequireAnyPermission` available per-route
- Swagger UI admin-gated in release

**Transport:**
- `DB_SSLMODE=disable` blocked in release mode
- `NEO4J_URI` with `bolt://` blocked in release (requires `bolt+ssc://` or `neo4j+s://`)
- `VAULT_ADDR` must be HTTPS in release when Vault is enabled
- `TEMPORAL_TLS_ENABLED=true` required in release

**Rate limiting:** 100 req/min per IP (token bucket, cleans up on shutdown)  
**CORS:** Configurable via `ALLOWED_ORIGINS` env; defaults to `http://localhost:3000`; `AllowCredentials: true`  
**Security headers:** `middleware.SecurityHeaders()` — HSTS, CSP, X-Frame-Options  
**Request size limit:** 10MB multipart  
**Graceful shutdown:** 5s context timeout; all modules, Temporal worker, rate limiter, OTel tracer shut down in order  
**Secret validation:** `validateRequiredEnvVars()` checks placeholder values for ENCRYPTION_KEY, JWT_SECRET, POSTGRES_PASSWORD, SCANNER_SERVICE_TOKEN, VAULT_TOKEN at startup  

**Issues found:**
1. **`panic()` in production goroutines** — `apps/backend/modules/discovery/service/snapshot_service.go:76` and `drift_detection_service.go:71`. These are in `recover()`-wrapped goroutines based on the pattern, but should be `return` with error propagation instead of re-panic.
2. **Default `.env.example` has `AUTH_REQUIRED=false`** — safe in dev but operators must explicitly change before production. The startup check enforces this; document clearly.
3. **`/metrics` endpoint unauthenticated** — intentional per inline comment (Prometheus scrape compatibility) but exposes goroutine count and memory stats. Consider IP allowlist in production.

---

## Section 5: Integration Surface

### External dependencies

| Service | Connection | Required | Notes |
|---|---|---|---|
| PostgreSQL | `DB_HOST:DB_PORT` | YES | Auto-migrates at startup |
| Neo4j | `NEO4J_URI` (bolt) | YES | `NEO4J_PASSWORD` required — no default |
| goScanner | `SCANNER_URL` (HTTP) | YES | Shared `SCANNER_SERVICE_TOKEN` |
| Presidio | `PRESIDIO_ADDR` (HTTP) | Optional | Falls back gracefully if unreachable |
| Vault | `VAULT_ADDR` | Optional | `VAULT_ENABLED=false` by default |
| Temporal | `TEMPORAL_HOST_PORT` (gRPC) | Optional | `TEMPORAL_ENABLED=false` by default |
| Supermemory | `SUPERMEMORY_API_URL` | Optional | `SUPERMEMORY_ENABLED=false` by default |
| Redis | `REDIS_ADDR` | Optional | Used by rate limiter |

### Internal message flows

1. **Scan trigger:** Frontend → `POST /api/v1/scans/trigger` → Backend enqueues → `POST /scan` to goScanner with `X-Scanner-Token`
2. **Scan result ingestion:** goScanner → `POST /api/v1/scans/ingest-verified` with `X-Scanner-Token` → Backend stores findings
3. **Neo4j sync:** Backend writes to `neo4j_sync_queue` (outbox) → `Neo4jSyncWorker` drains queue on 30s tick → Neo4j graph update
4. **Temporal (optional):** Scan completion triggers `ScanWorkflow` → durable retry orchestration
5. **WebSocket:** Real-time scan progress pushed to frontend via `/ws`

---

## Section 6: Docker / Deployment

**docker-compose.yml services:**

| Service | Image | Port | Notes |
|---|---|---|---|
| postgres | postgres:15-alpine | 127.0.0.1:5432 | loopback-only |
| neo4j | neo4j:5.15-community | 127.0.0.1:7474, 127.0.0.1:7687 | loopback-only; APOC plugin |
| presidio-analyzer | mcr.microsoft.com/presidio-analyzer | expose 3000 only | no host mapping — Docker-internal only |
| temporal | temporalio/auto-setup:1.22.0 | 127.0.0.1:7233 | loopback-only |
| temporal-ui | temporalio/ui:2.21.0 | 127.0.0.1:8088 | no auth — loopback-only required |
| vault | hashicorp/vault:1.15 | 127.0.0.1:8200 | dev mode; `VAULT_DEV_ROOT_TOKEN` required |
| redis | redis:7-alpine | 127.0.0.1:6379 | loopback-only; AOF enabled |

All services use a private bridge network (`172.31.240.0/24`). All host ports are bound to `127.0.0.1`. Presidio has no host port at all. Resource limits defined for every service.

Backend and goScanner containers are not in docker-compose — built and run separately (or via a deployment-specific compose override).

**Startup order requirement:** postgres must be healthy before temporal. Backend requires postgres + neo4j healthy before modules initialize. The compose file has `condition: service_healthy` on the postgres dependency for temporal.

---

## Section 7: Known Partial Integrations

| Area | Status | Notes |
|---|---|---|
| Remediation connectors | STUB | `REMEDIATION_ENABLED=false` default; `validateRequiredEnvVars` warns in release; connector write paths exist in code but untested against real DBs |
| MSSQL connector sample size | INCOMPLETE | `apps/goScanner/internal/connectors/databases/mssql.go:73` still has `TOP 10000` hardcoded — F-04 was not applied to MSSQL |
| Temporal workflows | OPTIONAL | Disabled by default; scan retries and remediation rollbacks unavailable without it; production boot warns |
| Vault AppRole | PARTIAL | Token auth implemented; AppRole env vars documented in .env.example but client init path only uses token |
| Celery manifests | LEGACY | 17 Kubernetes celery manifests exist with LEGACY comments; no Go Celery implementation — these are dead deployment artifacts |
| RLS transaction isolation | WEAK | Session-level SET (not SET LOCAL) — stale tenant context possible under pool reuse |
| Presidio | OPTIONAL | Falls back to rule-based classification if unreachable; no hard failure |
| Supermemory | OPTIONAL | `NoOpMemoryRecorder` when disabled |

---

## Section 8: Fix Priority

### P0 — Block production deployment

| ID | Location | Issue | Fix |
|---|---|---|---|
| P0-1 | `apps/backend/modules/auth/middleware/auth_middleware.go:416` | RLS uses session-level `SET`, not `SET LOCAL` — stale tenant context under connection pool reuse | Wrap each DB operation in an explicit transaction and use `SET LOCAL app.current_tenant_id` |
| P0-2 | `apps/goScanner/internal/connectors/databases/mssql.go:73` | `TOP 10000` hardcoded — F-04 not applied to MSSQL connector | Add `sampleSize int` field, `cfgInt(config, "sample_size", 1000, 50000)` in Connect(), change query to `TOP %d` with `c.sampleSize` |

### P1 — Fix before multi-tenant load

| ID | Location | Issue | Fix |
|---|---|---|---|
| P1-1 | `apps/backend/modules/discovery/service/snapshot_service.go:76` | `panic(p)` in production goroutine | Replace with error channel or structured error return |
| P1-2 | `apps/backend/modules/discovery/service/drift_detection_service.go:71` | Same as P1-1 | Same fix |

### P2 — Operational improvements

| ID | Location | Issue | Fix |
|---|---|---|---|
| P2-1 | `.env.example` | `AUTH_REQUIRED=false` as default | Change to `AUTH_REQUIRED=true` with comment for dev override |
| P2-2 | `.env.example` + `main.go:577` | `TEMPORAL_TLS_ENABLED=false` default but release mode requires `true` | Change default to `true` or add stronger dev/prod split documentation |
| P2-3 | `apps/backend/cmd/server/main.go:365` | `/metrics` unauthenticated | Add IP allowlist or internal-network-only binding in production |

### P3 — Deferred / acceptable risk

| ID | Location | Issue | Fix |
|---|---|---|---|
| P3-1 | Multiple connectors | Table names interpolated in SQL via `fmt.Sprintf` | Already mitigated by `sanitizePgIdentifier` (remediation) and connector-metadata-sourced names (scanner); verify no user input reaches these paths |
| P3-2 | `apps/goScanner/internal/connectors/saas/salesforce.go:105` | `LIMIT 10000` hardcoded in Salesforce SOQL | Apply same F-04 sample_size pattern |
