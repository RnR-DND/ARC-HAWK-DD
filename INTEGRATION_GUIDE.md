# ARC-HAWK-DD Integration Guide

**Version:** 3.0.1  
**Last updated:** 2026-04-24  
**API base:** `http://<host>:8080`  

---

## Architecture Overview

ARC-HAWK-DD is a modular monolith with a separate scanner microservice.

```
                    ┌─────────────────────────────────┐
                    │         React Frontend           │
                    │         (Next.js, :3000)         │
                    └────────────┬────────────────────┘
                                 │ JWT Bearer / API Key
                    ┌────────────▼────────────────────┐
                    │       Go Backend (:8080)         │
                    │  auth · scanning · compliance    │
                    │  assets · lineage · discovery    │
                    │  connections · masking · fp      │
                    │  analytics · remediation         │
                    └──┬──────────┬──────────┬────────┘
                       │          │          │
          X-Scanner-Token  bolt+ssc  HTTP/gRPC
                       │          │          │
              ┌────────▼──┐ ┌────▼──┐ ┌────▼──────┐
              │ goScanner │ │ Neo4j │ │ Presidio  │
              │  (:8001)  │ │(:7687)│ │  (:3000)  │
              └───────────┘ └───────┘ └───────────┘
```

**Backend** is the integration entry point. All clients talk to the backend only. The scanner and Presidio are internal services — never exposed directly.

---

## Quickstart

### Prerequisites

- Docker + Docker Compose
- Go 1.21+ (for backend build)
- Node.js 18+ (for frontend build)

### 1. Configure environment

```bash
cp .env.example .env
# Required: set all CHANGE_ME values
# Minimum required for dev:
#   DB_HOST, DB_USER, DB_PASSWORD, DB_NAME, DB_PORT
#   NEO4J_PASSWORD
#   JWT_SECRET (min 32 chars)
#   ENCRYPTION_KEY (exactly 32 chars)
#   SCANNER_SERVICE_TOKEN (min 32 chars, shared with goScanner)
```

### 2. Start infrastructure

```bash
docker-compose up -d postgres neo4j presidio-analyzer redis
# Wait for postgres and neo4j to be healthy before starting the backend
```

### 3. Build and run backend

```bash
cd apps/backend
go build -o bin/server ./cmd/server
./bin/server
# Migrations run automatically at startup
# Ready when you see: "Server starting on port 8080"
```

### 4. Build and run goScanner

```bash
cd apps/goScanner
go build -o bin/scanner ./cmd/scanner
SCANNER_SERVICE_TOKEN=<same-token-as-backend> ./bin/scanner
# Ready when you see: "Go scanner starting on :8001"
```

### 5. Build and run frontend

```bash
cd apps/frontend
npm install
npm run dev   # development
# or
npm run build && npm start   # production
```

### 6. Verify

```bash
curl http://localhost:8080/readyz
# {"status":"ready","service":"arc-platform-backend","db_healthy":true,"neo4j_healthy":true}

curl http://localhost:8001/health
# {"status":"ok"}
```

---

## Authentication

### JWT (user sessions)

```http
POST /api/v1/auth/login
Content-Type: application/json

{"email": "user@example.com", "password": "password"}
```

Response:
```json
{
  "token": "eyJ...",
  "refresh_token": "eyJ...",
  "user": {"id": "...", "email": "...", "role": "admin"}
}
```

Use the token on all subsequent requests:
```http
Authorization: Bearer eyJ...
```

**Token refresh:**
```http
POST /api/v1/auth/refresh
Authorization: Bearer <refresh_token>
```

### API Keys (service-to-service)

API keys are created through the settings UI or directly in the `api_keys` table. They are SHA256-hashed at rest.

```http
X-API-Key: <raw-api-key>
```

API key context sets `user_role` to `"service"` and `user_id` to `uuid.Nil`. Scope enforcement is via `api_key_scopes` array on the key record.

### Scanner Token (internal only)

The goScanner uses a shared secret for callbacks to the backend. This header is for machine-to-machine use only — never expose to end users.

```http
X-Scanner-Token: <SCANNER_SERVICE_TOKEN>
```

---

## Endpoints Reference

### Auth

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/auth/login` | Public | Login, returns JWT |
| POST | `/api/v1/auth/register` | Public | Register new user |
| POST | `/api/v1/auth/refresh` | Public | Refresh JWT |
| GET | `/api/v1/auth/profile` | JWT | Current user profile |
| POST | `/api/v1/auth/change-password` | JWT | Change password |
| GET | `/api/v1/auth/users` | JWT + admin | List users |
| GET | `/api/v1/auth/settings` | JWT | User settings |
| PUT | `/api/v1/auth/settings` | JWT | Update settings |
| GET | `/api/v1/auth/settings/notifications` | JWT | Notification prefs |
| PUT | `/api/v1/auth/settings/notifications` | JWT | Update notification prefs |

### Scans

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/scans/trigger` | JWT | Trigger a new scan (rate-limited) |
| GET | `/api/v1/scans` | JWT | List scans |
| GET | `/api/v1/scans/:id` | JWT | Get scan details |
| GET | `/api/v1/scans/:id/status` | JWT | Scan status |
| DELETE | `/api/v1/scans/:id` | JWT | Delete scan |
| POST | `/api/v1/scans/:id/cancel` | JWT | Cancel scan |
| GET | `/api/v1/scans/patterns` | JWT | List PII patterns |
| POST | `/api/v1/scans/patterns` | JWT | Create pattern |
| PUT | `/api/v1/scans/patterns/:id` | JWT | Update pattern |
| DELETE | `/api/v1/scans/patterns/:id` | JWT | Delete pattern |
| GET | `/api/v1/scans/findings/export` | JWT | Export findings (CSV/JSON) |
| POST | `/api/v1/scans/feedback` | JWT | Submit scan feedback |
| GET | `/api/v1/scans/dashboard` | JWT | Dashboard metrics |
| POST | `/api/v1/scans/agent/sync` | JWT | Sync agent state |
| GET | `/api/v1/scans/schedules` | JWT | List schedules |
| POST | `/api/v1/scans/schedules` | JWT | Create schedule |
| PUT | `/api/v1/scans/schedules/:id` | JWT | Update schedule |
| DELETE | `/api/v1/scans/schedules/:id` | JWT | Delete schedule |
| POST | `/api/v1/scans/ingest-verified` | Scanner Token | Scanner callback: ingest findings |
| POST | `/api/v1/scans/:id/complete` | Scanner Token | Scanner callback: mark complete |

### Connections

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/connections/available-types` | JWT | List supported connector types |
| GET | `/api/v1/connections` | JWT | List connections |
| POST | `/api/v1/connections` | JWT | Create connection |
| GET | `/api/v1/connections/:id` | JWT | Get connection |
| PUT | `/api/v1/connections/:id` | JWT | Update connection |
| DELETE | `/api/v1/connections/:id` | JWT | Delete connection |
| POST | `/api/v1/connections/:id/test` | JWT | Test connection |
| POST | `/api/v1/connections/:id/sync` | JWT | Sync connection metadata |
| POST | `/api/v1/connections/scans/scan-all` | JWT | Trigger scan on all connections |
| GET | `/api/v1/connections/scans/:id/status` | JWT | Connection scan status |
| GET | `/api/v1/connections/scans/jobs` | JWT | List all scan jobs |

### Compliance

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/compliance/overview` | JWT | Compliance posture summary |
| GET | `/api/v1/compliance/violations` | JWT | List violations |
| GET | `/api/v1/compliance/dpdpa` | JWT | DPDPA status |
| POST | `/api/v1/compliance/dpdpa` | JWT | Submit DPDPA assessment |
| GET | `/api/v1/compliance/consent` | JWT | List consent records |
| POST | `/api/v1/compliance/consent` | JWT | Record consent |
| GET | `/api/v1/compliance/dpr` | JWT | DPR requests |
| GET | `/api/v1/compliance/gro` | JWT | GRO settings |
| PUT | `/api/v1/compliance/gro` | JWT | Update GRO settings |
| POST | `/api/v1/compliance/evidence` | JWT | Generate evidence package |
| GET | `/api/v1/compliance/audit-trail` | JWT | Compliance audit trail |
| GET | `/api/v1/compliance/retention` | JWT | Retention policies |
| POST | `/api/v1/compliance/retention` | JWT | Create retention policy |
| GET | `/api/v1/compliance/audit-logs` | JWT | Audit logs |

### Assets & Findings

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/assets` | JWT | List data assets |
| GET | `/api/v1/findings` | JWT | List PII findings |
| GET | `/api/v1/dataset/golden` | JWT | Golden dataset |

### Lineage

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/lineage` | JWT | Data lineage graph |
| GET | `/api/v1/lineage/stats` | JWT | Lineage statistics |
| POST | `/api/v1/lineage/sync` | JWT | Trigger lineage sync to Neo4j |
| GET | `/api/v1/lineage/semantic` | JWT | Semantic lineage graph |

### Discovery

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/discovery/overview` | JWT | Discovery overview |
| GET | `/api/v1/discovery/inventory` | JWT | Data inventory |
| GET | `/api/v1/discovery/snapshots` | JWT | Historical snapshots |
| GET | `/api/v1/discovery/risk` | JWT | Risk profile |
| GET | `/api/v1/discovery/drift` | JWT | Data drift report |
| GET | `/api/v1/discovery/reports` | JWT | Downloadable reports |
| GET | `/api/v1/discovery/glossary` | JWT | Data glossary |

### Masking

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/masking/mask-asset` | JWT | Mask PII in findings table (internal DB only) |
| GET | `/api/v1/masking/status/:assetId` | JWT | Masking status for asset |

**Note:** Masking only affects the internal findings table. It does not mutate source-system data. Use Remediation connectors for source mutations.

### False-Positive Learning

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/fp/false-positives` | JWT | List FP reports |
| POST | `/api/v1/fp/false-positives` | JWT | Submit FP report |
| GET | `/api/v1/fp/confirmed` | JWT | Confirmed FPs |
| GET | `/api/v1/fp/learnings` | JWT | FP learning rules |
| POST | `/api/v1/fp/learnings` | JWT | Create learning rule |
| PUT | `/api/v1/fp/learnings/:id` | JWT | Update rule |
| DELETE | `/api/v1/fp/learnings/:id` | JWT | Delete rule |
| GET | `/api/v1/fp/stats` | JWT | FP statistics |
| POST | `/api/v1/fp/check` | JWT | Check if field is FP |

### Memory

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/memory/status` | JWT | Supermemory integration status |
| POST | `/api/v1/memory/search` | JWT | Hybrid semantic search over scan history |

### Health & Observability

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/livez` | Public | Liveness probe (always 200 while up) |
| GET | `/readyz` | Public | Readiness probe (503 if DB or Neo4j down) |
| GET | `/health` | Public | Alias for /readyz |
| GET | `/metrics` | Public | Prometheus metrics |
| GET | `/api/v1/health/components` | JWT | Per-component health (DB, Neo4j, Vault, Presidio) |
| GET | `/ws` | JWT | WebSocket — real-time scan progress |
| GET | `/swagger/*` | Public (dev) / Admin (release) | Swagger UI |

### goScanner API

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | Public | Scanner health |
| GET | `/metrics` | Public | Prometheus metrics |
| POST | `/scan` | X-Scanner-Token | Trigger scan job |

---

## Event Streams

### WebSocket (`GET /ws`)

Connect with a valid JWT. The server pushes JSON events:

```json
{"type": "scan_progress", "scan_id": "...", "percent": 42, "message": "Scanning table users"}
{"type": "scan_complete", "scan_id": "...", "findings_count": 1234}
{"type": "scan_failed", "scan_id": "...", "error": "connection timeout"}
```

### Prometheus Metrics

Backend exposes at `/metrics`:
- `go_goroutines` — goroutine count
- `go_memstats_alloc_bytes` — heap in use
- `go_memstats_sys_bytes` — system memory
- `arc_hawk_modules_count` — initialized module count
- `arc_hawk_scan_findings_total{pii_type, source_type}` — findings counter
- `arc_hawk_presidio_latency_seconds` — Presidio classification histogram
- `arc_hawk_active_scans` — concurrent scans in progress
- `arc_hawk_neo4j_sync_queue_pending` — outbox queue depth

goScanner exposes standard Go runtime metrics via `promhttp.Handler()` at `/metrics`.

---

## Environment Variables Reference

### Required

| Variable | Description | Example |
|---|---|---|
| `DB_HOST` | PostgreSQL host | `postgres` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `strong-password` |
| `DB_NAME` | PostgreSQL database | `arc_platform` |
| `NEO4J_PASSWORD` | Neo4j password (no default) | `strong-password` |
| `JWT_SECRET` | JWT signing key (min 32 chars) | `openssl rand -base64 48` |
| `ENCRYPTION_KEY` | AES-256 key (exactly 32 chars) | `openssl rand -hex 16` |
| `SCANNER_SERVICE_TOKEN` | Shared secret backend↔scanner (min 32 chars) | `openssl rand -hex 32` |

### Optional with defaults

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Backend listen port |
| `GIN_MODE` | `debug` | Set `release` for production |
| `AUTH_REQUIRED` | `false` | Must be `true` in production |
| `NEO4J_URI` | `bolt://127.0.0.1:7687` | Neo4j connection URI |
| `NEO4J_USERNAME` | `neo4j` | Neo4j username |
| `ALLOWED_ORIGINS` | `http://localhost:3000` | CORS allowed origins (comma-separated) |
| `DB_SSLMODE` | `disable` | Set `require` or `verify-full` in production |
| `PRESIDIO_ENABLED` | `true` | Enable ML classification |
| `PRESIDIO_ADDR` | `http://presidio-analyzer:3000` | Presidio service URL |
| `TEMPORAL_ENABLED` | `false` | Enable Temporal workflow engine |
| `TEMPORAL_HOST_PORT` | `temporal:7233` | Temporal gRPC address |
| `TEMPORAL_TLS_ENABLED` | `false` | Required `true` in release mode |
| `VAULT_ENABLED` | `false` | Enable HashiCorp Vault |
| `VAULT_ADDR` | `http://vault:8200` | Vault address |
| `VAULT_TOKEN` | — | Vault token (dev) or use AppRole |
| `SUPERMEMORY_ENABLED` | `false` | Enable cross-session memory |
| `SUPERMEMORY_API_KEY` | — | Supermemory API key |
| `REMEDIATION_ENABLED` | `false` | Enable remediation connectors |
| `PII_STORE_MODE` | `full` | `full` \| `mask` \| `none` |
| `SCAN_TIMEOUT_SECONDS` | `1800` | Max wall-clock time per scan |
| `TENANT_MAX_CONCURRENT_SCANS` | `5` | Per-tenant scan concurrency limit |
| `SCANNER_AUTH_REQUIRED` | `true` | Set `false` only for scanner dev mode |
| `REDIS_ADDR` | `localhost:6379` | Redis for rate limiter |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTel collector endpoint |

### Production-required overrides

When `GIN_MODE=release`, the server will refuse to start unless:
- `AUTH_REQUIRED=true`
- `ENCRYPTION_KEY` is ≥ 32 chars and not the placeholder
- `JWT_SECRET` is ≥ 32 chars and not the placeholder
- `DB_SSLMODE` is not `disable`
- `SCANNER_SERVICE_TOKEN` is ≥ 32 chars and not `dev-scanner-token-change-me`
- `NEO4J_URI` does not start with `bolt://` (use `bolt+ssc://` or `neo4j+s://`)
- `TEMPORAL_TLS_ENABLED=true`
- If `VAULT_ENABLED=true`: `VAULT_ADDR` must be HTTPS, `VAULT_TOKEN` ≥ 32 chars

---

## Integration Checklist

### Connecting ARC-HAWK-DD to a parent system

- [ ] Parent system creates a user via `POST /api/v1/auth/register` or provisions one directly in the DB
- [ ] Parent system authenticates via `POST /api/v1/auth/login` and stores the JWT + refresh token
- [ ] Parent system refreshes JWT before expiry via `POST /api/v1/auth/refresh`
- [ ] Alternatively: create an API key for service-to-service calls (use `X-API-Key` header)
- [ ] Set `ALLOWED_ORIGINS` to include the parent system's domain
- [ ] Set `TENANT_MAX_CONCURRENT_SCANS` appropriate for your load

### Triggering scans

```bash
# 1. Create a connection
curl -X POST http://localhost:8080/api/v1/connections \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"name": "prod-db", "type": "postgres", "config": {"host": "...", "port": 5432, ...}}'

# 2. Trigger scan
curl -X POST http://localhost:8080/api/v1/scans/trigger \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"connection_id": "<id>", "scan_type": "full"}'

# 3. Poll status
curl http://localhost:8080/api/v1/scans/<scan_id>/status \
  -H "Authorization: Bearer $JWT"

# 4. Get findings
curl "http://localhost:8080/api/v1/findings?scan_id=<scan_id>" \
  -H "Authorization: Bearer $JWT"
```

### Monitoring integration

```yaml
# Prometheus scrape config
scrape_configs:
  - job_name: arc-hawk-backend
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics

  - job_name: arc-hawk-scanner
    static_configs:
      - targets: ['localhost:8001']
    metrics_path: /metrics
```

Key alerts to configure:
- `arc_hawk_neo4j_sync_queue_pending > 100` — outbox falling behind
- `arc_hawk_active_scans > TENANT_MAX_CONCURRENT_SCANS` — scan queue pressure
- Backend `/readyz` returning 503 — dependency down

### Security hardening before production

- [ ] Generate all secrets: `openssl rand -base64 48` for JWT_SECRET, `openssl rand -hex 16` for ENCRYPTION_KEY, `openssl rand -hex 32` for SCANNER_SERVICE_TOKEN
- [ ] Set `GIN_MODE=release`
- [ ] Set `AUTH_REQUIRED=true`
- [ ] Set `DB_SSLMODE=require` (or `verify-full`)
- [ ] Set `NEO4J_URI=bolt+ssc://...` or `neo4j+s://...`
- [ ] Set `TEMPORAL_TLS_ENABLED=true` (or disable Temporal)
- [ ] Restrict `/metrics` to Prometheus scraper IP only (reverse proxy or firewall)
- [ ] Keep all docker-compose ports bound to `127.0.0.1` — do not change to `0.0.0.0`
- [ ] Do not expose Temporal UI to the public internet (no auth)
- [ ] Set `REMEDIATION_ENABLED=false` until write-path connectors are verified

### Known limitations to communicate to parent system

1. **Masking is internal-DB-only** — `POST /api/v1/masking/mask-asset` marks findings as masked in ARC-HAWK-DD's findings table. Source system data is unchanged. Use Remediation connectors for source mutations.
2. **MSSQL sample size is hardcoded at 10,000** — configurable sample_size does not apply to MSSQL connector until P0-2 fix is applied.
3. **RLS uses session-level isolation** — under high concurrency, a connection pool connection may briefly carry a stale tenant ID. Low risk for single-tenant deployments; fix is in progress.
4. **Remediation connectors are stubs** — `REMEDIATION_ENABLED` must remain `false` until connector write-paths are verified against target databases.
