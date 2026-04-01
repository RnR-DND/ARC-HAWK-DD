# Production Audit Report — ARC-Hawk

**Date:** 2026-03-31  
**Auditor:** Claude Code (Senior Full-Stack Production Engineer)  
**Branch:** dev

---

## Summary

| Category | Found | Fixed | Remaining |
|----------|-------|-------|-----------|
| 🔴 CRITICAL | 9 | 9 | 0 |
| 🟡 WARNING | 5 | 5 | 0 |
| 🟢 SUGGESTION | 2 | 2 | 0 |
| **Total** | **16** | **16** | **0** |

---

## Changes Made

### Security (CRITICAL)

**1. Hardcoded credentials in docker-compose.yml**
- **Before:** Postgres env vars commented out with hardcoded `postgres`/`postgres`/`arc_platform` literals
- **After:** `${POSTGRES_USER}`, `${POSTGRES_PASSWORD}`, `${POSTGRES_DB}` from env file
- **Before:** Neo4j healthcheck had `password123` hardcoded in plain text
- **After:** Uses `${NEO4J_USERNAME}` / `${NEO4J_PASSWORD}` from env
- **Before:** Grafana `GF_SECURITY_ADMIN_PASSWORD=admin` hardcoded
- **After:** `${GRAFANA_ADMIN_PASSWORD}` from env file

**2. Backend port publicly exposed**
- **Before:** `"8080:8080"` — bound to `0.0.0.0`, accessible from any interface
- **After:** `"127.0.0.1:8080:8080"` — localhost only (consistent with Postgres/Neo4j)

**3. Credential leak via debug log — connection_service.go:31**
- **Before:** `log.Printf("DEBUG CONFIG INPUT: %+v", config)` logged plaintext DB credentials
- **After:** Line removed; unused `log` import also removed

**4. Hardcoded Neo4j fallback password — main.go**
- **Before:** `getEnv("NEO4J_PASSWORD", "password123")` — backdoor default if env var unset
- **After:** `os.Getenv("NEO4J_PASSWORD")` with `log.Fatal` if empty — fails fast

**5. Fake finding injection — scanner_api.py:246-269**
- **Before:** When a scan found zero results, a synthetic `IN_PAN` finding with `"test_injection"` validator was silently injected into production data, corrupting findings integrity
- **After:** Empty scans log an informational message and return — no data is fabricated

**6. Auth bypass enabled by default — main.go**
- **Before:** `getEnv("AUTH_REQUIRED", "false")` — all API routes open without authentication by default
- **After:** `getEnv("AUTH_REQUIRED", "true")` — authentication required by default; `false` is an explicit opt-out for dev

**7. Missing JWT_SECRET and GRAFANA_ADMIN_PASSWORD in .env**
- **Before:** `.env` had no `JWT_SECRET` (JWT service fell back to ephemeral random key) and no `GRAFANA_ADMIN_PASSWORD`
- **After:** Both keys added with `CHANGE_ME_*` placeholder values to make the requirement explicit

---

**8. ClearScanData endpoint — no authorization check**
- **Before:** `DELETE /api/v1/scans/clear` deleted all scan data with no role check — any authenticated user could wipe the database
- **After:** Admin role required (`user_role == "admin"`); 403 returned otherwise

**9. Dockerfile ran as root + no health check**
- **Before:** Final image used `WORKDIR /root/` with implicit root user; no `HEALTHCHECK` directive
- **After:** Dedicated `appuser`/`appgroup` (non-root); `HEALTHCHECK` via `wget` on `/health`

---

### Code Quality (WARNING)

**8. X-RateLimit-Limit header encoding bug — rate_limiter.go:85**
- **Before:** `string(rune(rl.requestsRate))` — converts int to Unicode rune, producing a garbage character (e.g., `d` for 100) instead of the string `"100"`
- **After:** `strconv.Itoa(rl.requestsRate)` — correct integer-to-string conversion

**9. IP spoofing via X-Forwarded-For — rate_limiter.go**
- **Before:** Trusted client-supplied `X-Forwarded-For` / `X-Real-IP` headers, allowing bypass by spoofing any IP
- **After:** Uses `c.ClientIP()` (Gin's built-in, respects trusted proxy config); removed `strings` import

**10. Untyped context key — main.go**
- **Before:** String literal `"tenant_id"` as context key — can collide with other packages using the same string
- **After:** Private `contextKey` type (`type contextKey string`) with typed constant `contextKeyTenantID`

**11. print() instead of logging — scanner_api.py**
- **Before:** `print("[Scanner] ...")` throughout — no log levels, no timestamps, no structured output
- **After:** `import logging` with `logger = logging.getLogger('arc-hawk-scanner')`, using `logger.info/warning/error` with `exc_info=True` on errors

---

### Configuration / Documentation (SUGGESTION)

**12. Hardcoded WebSocket URL — frontend/app/page.tsx**
- **Before:** `'ws://localhost:8000/ws'` — wrong port (8000 vs 8080) and not configurable
- **After:** `process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws'`

**12b. Internal error details exposed in ingestion_handler.go**
- **Before:** `"details": err.Error()` in HTTP 500 responses leaked internal stack/DB details to clients
- **After:** Generic `"Failed to ingest scan"` message only; details remain server-side

**13. .env.example missing required variables**
- **Before:** Missing `JWT_SECRET`, `GRAFANA_ADMIN_PASSWORD`; used `DATABASE_USER` (mismatch with `DB_USER` in main.go); no `AUTH_REQUIRED`
- **After:** Complete, aligned with actual env var names used in code, with generation instructions for secrets

---

### New Files Created

| File | Purpose |
|------|---------|
| `ARCHITECTURE.md` | System diagram, module graph, data flow, tech stack |
| `AUDIT_REPORT.md` | This file |

---

## Architecture Overview

See `ARCHITECTURE.md` for the full system diagram.

**Stack:** Go 1.24 backend (modular monolith) · Next.js frontend · Python scanner · PostgreSQL · Neo4j · Docker Compose

---

## How to Run

```bash
# 1. Copy and fill in secrets
cp apps/backend/.env.example .env
# Edit .env — set JWT_SECRET, ENCRYPTION_KEY, all passwords

# 2. Start all services
docker-compose up --build

# 3. Verify health
curl http://localhost:8080/health
```

---

## Deployment Checklist

- [x] All credentials loaded from environment variables (no hardcoded values)
- [x] Backend port localhost-restricted in docker-compose
- [x] Auth required by default (`AUTH_REQUIRED=true`)
- [x] Security headers enabled (HSTS, CSP, X-Frame-Options)
- [x] Rate limiting enabled (100 req/min per IP)
- [x] Database migrations run on startup
- [x] Graceful shutdown handles in-flight requests
- [x] Health check endpoint at `/health`
- [ ] `JWT_SECRET` — must be set to a strong random value before deploy
- [ ] `ENCRYPTION_KEY` — must be set to a random 32-char value before deploy
- [ ] `NEO4J_PASSWORD` — must be changed from default before deploy
- [ ] `POSTGRES_PASSWORD` — must be changed from default before deploy
- [ ] `AUTH_REQUIRED=false` — must NOT be set in production
- [ ] HTTPS/TLS termination — must be configured at load balancer / reverse proxy level
- [ ] Token blacklist — `InvalidateToken` in jwt_service.go is a stub; implement Redis blacklist before production logout is critical

---

## Known Limitations / Technical Debt

1. **Token invalidation not implemented** — `jwt_service.go:InvalidateToken` logs the token but does not blacklist it. Logout does not actually invalidate sessions. Requires a Redis or DB-backed blacklist.

2. **In-memory scan state** — `active_scans` dict in `scanner_api.py` is lost on restart. Use a persistent store (Postgres) for scan status in production.

3. **Masking tokenization uses SHA-256** — for reversible tokenization, HMAC-SHA256 with a secret key should be used instead of plain SHA-256.

4. **Findings limit hardcoded at 10,000** — `ListFindingsByAsset(ctx, assetID, 10000, 0)` in masking_service.go; add proper pagination.

5. **No test coverage for handlers** — only `ml_patterns_test.go` and `scan_service_test.go` exist. Critical API handlers have no tests.
