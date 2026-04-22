# Integration Ready — ARC-HAWK-DD v3.0.0

Consumer checklist for embedding ARC-HAWK-DD as a microservice in a main platform.

---

## Architecture Summary

ARC-HAWK-DD exposes a single REST API (`/api/v1/*`) that a main platform consumes via its API gateway. Long-running work (scans, remediation) is initiated via REST and delivers results via HMAC-signed webhooks to a callback URL you provide.

```
Main Platform
  └── API Gateway ──► ARC-HAWK-DD :8080 /api/v1/*  (JWT or API-key)
                          └── Go Scanner :8001 (internal only, SCANNER_SERVICE_TOKEN)
                          └── Temporal (internal workflows)
                          └── Neo4j / PostgreSQL (internal data)
                          └── Presidio (internal NLP)
Main Platform
  └── Webhook receiver ◄── ARC-HAWK-DD (HMAC-signed POST, X-ARC-Signature)
```

---

## Pre-Integration Checklist

### Network
- [ ] ARC-HAWK-DD backend reachable at `https://<your-domain>/api/v1` from main platform
- [ ] Main platform's webhook receiver URL reachable from ARC-HAWK-DD (for async scan events)
- [ ] Scanner port 8001 is **NOT** exposed externally (internal only)
- [ ] Ports 9090 (Prometheus), 3002 (Grafana), 8088 (Temporal UI) restricted to internal network

### Authentication
- [ ] Create a tenant-scoped API key via `POST /api/v1/auth/login` → use returned JWT, **or**
- [ ] Pre-provision an API key in `api_keys` table with appropriate scopes
- [ ] Store API key in your platform's secrets manager (never in source code)

### Required environment variables (ARC-HAWK-DD backend)
| Variable | Purpose |
|---|---|
| `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection |
| `NEO4J_URI`, `NEO4J_PASSWORD` | Neo4j for semantic lineage |
| `JWT_SECRET` | ≥32 chars, random |
| `ENCRYPTION_KEY` | ≥32 chars, random — encrypts PII samples at rest |
| `SCANNER_SERVICE_TOKEN` | ≥32 chars, shared with Go scanner |
| `DB_SSLMODE=require` | Required in production |
| `NEO4J_URI=bolt+ssc://...` | TLS required in release mode |
| `TEMPORAL_TLS_ENABLED=true` | Required in release mode |

### Pending user actions before production
- [ ] Rotate Supermemory API key (see `TODO.md`)
- [ ] Generate fresh `SCANNER_SERVICE_TOKEN` (`openssl rand -hex 32`)
- [ ] Provision Neo4j TLS certificate
- [ ] Provision Temporal TLS certificate

---

## Core Integration Flows

### 1. Connect a data source
```
POST /api/v1/connections
Authorization: Bearer <jwt>
{
  "name": "Production MySQL",
  "type": "mysql",
  "credentials": { "host": "...", "port": 3306, "database": "prod", "username": "...", "password": "..." }
}
→ 201 { "id": "<connection_id>", "status": "connected" }
```

### 2. Trigger a scan
```
POST /api/v1/scans/trigger
{ "connection_ids": ["<connection_id>"], "scan_type": "full" }
→ 202 { "scan_id": "<scan_id>", "status": "pending" }
```

### 3. Poll scan status (or receive webhook)
```
GET /api/v1/scans/<scan_id>/status
→ { "status": "completed|partial|failed|running", "progress_pct": 100 }
```

### 4. Retrieve findings
```
GET /api/v1/findings?scan_id=<scan_id>&limit=100&offset=0
→ { "data": [{ "id": "...", "pii_type": "AADHAAR", "risk_score": 0.87, "asset_id": "..." }] }
```

---

## Webhook Events (async)

Register your callback URL in the connection config or via env `WEBHOOK_CALLBACK_URL`.

ARC-HAWK-DD will POST to your URL with header `X-ARC-Signature: sha256=<hmac>`:

| Event | When |
|---|---|
| `scan.started` | Scan begins |
| `scan.progress` | Each ingest chunk completes |
| `scan.completed` | Scan finishes (status: completed/partial/failed) |
| `finding.created` | High-risk finding detected (risk_score > 0.8) |
| `remediation.applied` | Remediation action executed |

Verify the signature: `HMAC-SHA256(payload, WEBHOOK_SECRET)`.

---

## Rate Limits

| Limit | Value |
|---|---|
| Global | 100 req/min per IP |
| Per-tenant | Configurable via semaphore (`MAX_CONCURRENT_SCANS_PER_TENANT`) |
| Scan concurrency | 1 concurrent scan per tenant by default |

---

## API Reference

- **Interactive (Swagger UI):** `GET /swagger/index.html` (admin JWT required in release mode)
- **OpenAPI spec:** `docs/openapi/openapi.yaml` (committed to repo)
- **Human reference:** `docs/INTEGRATION_GUIDE.md`

---

## Health Endpoints

| Endpoint | Auth | Purpose |
|---|---|---|
| `GET /livez` | Public | Process alive (always 200) — Kubernetes liveness |
| `GET /readyz` | Public | Deps healthy (200/503) — Kubernetes readiness |
| `GET /health` | Public | Back-compat alias for /readyz |
| `GET /api/v1/health/components` | JWT | Detailed component status |

---

## Known Limitations (P2 items — tracked in `TODO.md`)

- No per-API-key rate quotas (global IP rate limit only)
- No OAuth / SSO — JWT/API-key only
- Prometheus, Grafana, Temporal UI have no auth gate (restrict at network level)
- Per-tenant Neo4j isolation not yet implemented (shared graph, tenant_id in properties)
- JWT stored in localStorage (not httpOnly cookie) — planned for P2-13
- Oracle DB connector is a stub — implement or remove
- Audit log pagination hardcoded at limit 200
