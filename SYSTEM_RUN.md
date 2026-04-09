# System Run Report — ARC-HAWK-DD v3.0
**Date:** 2026-04-09  
**Section:** S7 — Full System Run

---

## Stack Status

| Container | Image | Port(s) | Status |
|-----------|-------|---------|--------|
| arc-platform-backend | arc-hawk-dd-backend | 8080 | ✅ healthy |
| arc-platform-scanner | arc-hawk-dd-scanner | 5002 | ✅ healthy |
| arc-platform-frontend | arc-hawk-dd-frontend | 3000 | ✅ up |
| arc-platform-db | postgres:15-alpine | 5432 | ✅ healthy |
| arc-platform-neo4j | neo4j:5.15.0 | 7474/7687 | ✅ healthy |
| arc-platform-temporal | temporalio/auto-setup | 7233 | ✅ healthy |
| arc-platform-temporal-ui | temporalio/ui | 8088 | ✅ up |
| arc-platform-presidio | presidio-analyzer | 3001 | ✅ healthy |
| arc-platform-prometheus | prom/prometheus | 9090 | ✅ up |
| arc-platform-grafana | grafana/grafana | 3002 | ✅ up |

---

## 13-Test Results

| # | Test | Result | Detail |
|---|------|--------|--------|
| T1 | Backend health endpoint | ✅ PASS | `{"service":"arc-platform-backend","status":"healthy"}` |
| T2 | DB migrations applied | ✅ PASS | 33 tables in public schema (000001–000036) |
| T3 | Scanner health endpoint | ✅ PASS | `{"service":"arc-hawk-scanner","status":"healthy","version":"0.3.39"}` |
| T4 | Frontend HTTP 200 | ✅ PASS | Next.js serving on :3000 |
| T5 | Presidio health | ✅ PASS | `Presidio Analyzer service is up` |
| T6 | Neo4j HTTP 200 | ✅ PASS | Neo4j 5.15.0 bolt+HTTP ready |
| T7 | Temporal UI HTTP 200 | ✅ PASS | Temporal UI serving on :8088 |
| T8 | Prometheus targets up | ✅ PASS | 1/1 targets up |
| T9 | hawk_patterns.py in scanner | ✅ PASS | 34 patterns loaded |
| T10 | Aadhaar regex match | ✅ PASS | regex=True (validator correctly rejects synthetic test number — expected) |
| T11 | PAN regex + validator | ✅ PASS | regex=True, validator=True |
| T12 | hawk_patterns_test.py in container | ✅ PASS | 944 passed, 1 warning (permission on cache dir) |
| T13 | Patterns CRUD API `/api/v1/patterns` | ✅ PASS | Response: `{"data":[]}` (empty, correct for fresh DB) |

**Result: 13/13 PASS**

---

## Notes

- T10: `2345 6789 0126` is a synthetic test number that passes regex `[2-9]\d{3}[\s\-]?\d{4}[\s\-]?\d{4}` but fails Verhoeff. This is **correct** — in production, confidence would be `0.60` (regex only), not `0.98` (regex + validator). This is the system working as designed.
- T12 cache warning: `/app/.pytest_cache` has permission issues inside container. Does not affect test results. Fix: add `pytest_cache_dir = /tmp/.pytest_cache` to `pytest.ini`.

---

## Rebuild Note

The `scanner` container was rebuilt during this run (`docker compose build scanner`) to incorporate `hawk_patterns.py`, `hawk_validators.py`, and `hawk_patterns_test.py` created in S0/S2. The image should be pushed before deployment.
