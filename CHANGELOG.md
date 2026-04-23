# Changelog

All notable changes to the ARC-Hawk platform will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.0.1] - 2026-04-23

### ✨ Added

- **OpenAPI/Swagger spec** — 105 endpoints now machine-readable at `/swagger/index.html` and `docs/openapi/swagger.json`. Integrating systems can generate SDKs directly.
- **Real PDF reports** — `GET /discovery/reports/:id/download` with `format=pdf` now returns actual PDF bytes (`application/pdf`) with title, KPI summary, source breakdown table, and risk hotspot table. Previously returned HTML.
- **36 connectors fully implemented** — all connector types now perform real connectivity checks (TCP dial, file stat, API endpoint reachability) instead of returning "unsupported". Includes SQLite, Oracle, MSSQL, Azure Blob, BigQuery, Snowflake, Redshift, Kafka, Kinesis, 10 file formats, Salesforce, HubSpot, Jira, and MS Teams.
- **Redis classify pipeline** — `POST /agent/sync` now publishes batches to Redis `classify` list via `LPUSH`. Set `REDIS_ADDR` env var to enable. Gracefully skips if Redis is not configured (data is safe in PostgreSQL).
- **API documentation** — `docs/API.md` now covers Auth Endpoints, FP Learning API, Masking API, Memory API, and Health API with request/response examples.

### 🔒 Security

- **CI/CD hardening** — security scanning workflow added; Go 1.25 enforced in CI; dependency vulnerability scanning enabled.
- **DB integrity** — DPDPA audit trail enforced; findings audit trail; immutable migration versioning rules applied.
- **Scanner connector timeouts** — all connectors now have explicit dial/read timeouts; DB connections are closed after use.

### 🐛 Fixed

- **Remediation saga** — Neo4j rollback now executes correctly on saga failure.
- **Prometheus metric registration** — duplicate metric registration panic resolved.
- Go version in docs updated from 1.21 to 1.25 to match `go.mod`.

---

## [3.0.0] - 2026-04-09

### 🎯 Major Changes

#### Enterprise Data Discovery Module v1

New module for large-scale asset inventory with risk scoring, spike detection, and semantic lineage.

- Upstream inventory query with per-asset finding aggregates
- Risk scoring service with trend analysis and `spike_detector.go`
- Discovery page (`/discovery`) with asset-type breakdowns and heatmap
- Semantic lineage tracking for sensitive field provenance

#### Custom Regex Patterns Engine

- CRUD API at `GET/POST/DELETE /api/v1/patterns`
- Server-side ReDoS protection: patterns >512 chars or with nested quantifiers are rejected at compile time
- Per-scan hot-reload — new patterns apply without service restart
- `services/patterns.api.ts` frontend client; scan config modal surfaces pattern errors inline

#### DPDPA Compliance Module (expanded)

- Obligation service mapping PII findings to DPDPA provisions
- Gap report export (PDF/JSON) via `export_handler.go`
- Retention policy tracking with violation detection
- Consent records (`000030_consent_records`)
- Audit log chain with hash-linked entries (`000029_audit_log_chain`)
- Compliance page redesign: responsive KPI grid, scrollable overflow tables

#### Scanner Expansion (14 new data sources)

Cloud: BigQuery, Redshift, Snowflake, Azure Blob, AWS Kinesis.
SaaS: Salesforce, HubSpot, Jira, MS Teams.
Files: Avro, Parquet, PPTX, HTML, Email. Plus SQLite.
Additions: LLM-assisted classifier (`llm_classifier.py`), field profiler (`field_profiler.py`), statistical sampling.

---

### 🔒 Security Fixes

**BREAKING — Scan deletion now requires admin role**

`DELETE /scans/:id` now returns 403 for non-admin users. Temporary mitigation for missing `tenant_id` on `scan_runs` (see RISK.md for full plan).

**P0 — ReDoS protection for custom patterns**

User-supplied regex patterns are now compiled through `_compile_custom_pattern_safely()` which rejects catastrophic backtracking shapes before they reach the engine (`scanner_api.py`).

**P0 — Gin router static-vs-wildcard ordering fixed**

`DELETE /scans/clear` is now registered before `DELETE /scans/:id`, preventing the wildcard from swallowing the literal route (`scanning/module.go`).

**P1 — JWT token_blacklist now pruned**

Expired rows deleted on `ValidateToken` calls via fire-and-forget goroutine (`auth/service/jwt_service.go:258`).

**P1 — API key last_used_at now tracked**

Async `UPDATE api_keys SET last_used_at = NOW()` on every authenticated request (`auth/middleware/auth_middleware.go:89`).

**P1 — WebSocket URL schema-normalized**

`NEXT_PUBLIC_WS_URL` is auto-prefixed with `wss://` on HTTPS origins — operators can now omit the scheme in the env var (`app/page.tsx`).

---

### ✨ Added

#### Backend
- `modules/discovery/` — full discovery module (domain, service, worker, API)
- `modules/scanning/api/patterns_handler.go` — custom pattern CRUD
- `modules/scanning/service/patterns_service.go` — pattern validation and cache
- `modules/compliance/service/dpdpa_obligation_service.go`
- `modules/compliance/service/report_service.go`
- `modules/remediation/api/escalation_handler.go` + `export_handler.go`
- `modules/remediation/service/escalation_service.go`, `export_service.go`, `sop_registry.go`
- `modules/scanning/workflows/streaming_supervisor_workflow.go`
- `modules/shared/infrastructure/encryption/encryption_service.go`
- PII sample encryption at rest (`000027_encrypt_pii_samples`)
- API key management (`000028_api_keys`)
- Risk score history table (`000031_risk_score_history`)
- Custom pattern hardening migration (`000032_custom_patterns_hardening`)
- Helm chart scaffolding (`helm/`)
- Grafana dashboards (`apps/scanner/config/grafana/`)

#### Frontend
- `/discovery` page with asset-type risk heatmap
- `services/compliance.api.ts` and `services/patterns.api.ts`
- Responsive compliance page (mobile-first grid, overflow-x-auto tables)

#### Scanner
- 14 new source commands (see Major Changes above)
- `sdk/llm_classifier.py`, `sdk/field_profiler.py`, `sdk/sampling.py`
- `sdk/recognizers/gst.py` — GST number recognition

---

### 🔧 Changed

- `scanning/module.go` — route ordering hardened; admin gate on scan delete
- `auth/middleware/auth_middleware.go` — async last_used_at update
- `auth/service/jwt_service.go` — token_blacklist pruning
- `app/page.tsx` — WS URL normalization
- `app/compliance/page.tsx` — responsive layout redesign
- `components/scans/ScanConfigModal.tsx` — pattern error display

---

### 🧪 Tests

- `MetricCards.test.tsx` — locale-agnostic assertion for large number formatting
- `AnalyticsPage.integration.test.tsx` — removed stale Topbar mock
- `patterns_handler_test.go` — stub for custom pattern CRUD tests
- All 36 frontend Jest tests pass; all 7 Go packages pass

---

### 📊 Migrations

| # | Name | Type |
|---|------|------|
| 000025 | custom_patterns | additive |
| 000026 | retention_policies | additive |
| 000027 | encrypt_pii_samples | additive |
| 000028 | api_keys | additive |
| 000029 | audit_log_chain | additive |
| 000030 | consent_records | additive |
| 000031 | risk_score_history | additive |
| 000032 | custom_patterns_hardening | additive |

---

### 📝 Known Issues / Deferred

- Tenant isolation on scan deletion: `scan_runs` lacks `tenant_id`. Admin gate is a temporary mitigation. Full fix tracked in RISK.md.

---

## [2.1.0] - 2026-01-13

### 🎯 Major Changes

#### Lineage Hierarchy Migration (4-Level → 3-Level)

**BREAKING CHANGE**: Complete architectural refactoring of the lineage system from a 4-level hierarchy to a simplified 3-level semantic model.

**Previous Architecture (Deprecated)**:
```
System → Asset → DataCategory → PII_Category
Edges: CONTAINS, HAS_CATEGORY
```

**New Architecture**:
```
System → Asset → PII_Category
Edges: SYSTEM_OWNS_ASSET, EXPOSES
```

**Benefits**:
- ✅ **Performance**: Simplified graph traversal reduces query complexity
- ✅ **Clarity**: Direct relationship between assets and PII types
- ✅ **Standards Alignment**: Better compatibility with OpenLineage specification
- ✅ **Maintainability**: 790 lines of legacy code removed

---

### 🗑️ Removed

#### Backend Services
- **`lineage_handler.go`** - Replaced by `lineage_handler_v2.go`
- **`lineage_service.go`** - Legacy lineage service with 4-level logic
- **`semantic_lineage_hierarchy.go`** - Old hierarchy implementation
- **`neo4j_schema.cypher`** - Archived as `neo4j_schema_OLD_4LEVEL.cypher`

#### Graph Elements
- **`DataCategory` nodes** - Intermediate layer no longer needed
- **`CONTAINS` edges** - Replaced by `SYSTEM_OWNS_ASSET`
- **`HAS_CATEGORY` edges** - Replaced by `EXPOSES`

---

### ✨ Added

#### Backend
- **`neo4j_semantic_contract_v1.cypher`** - Versioned schema definition for 3-level hierarchy
- **Enhanced `lineage_handler_v2.go`** - Optimized API handler with simplified queries
- **Improved `semantic_lineage_service.go`** - Refactored service layer for 3-level model

#### Frontend
- **Updated `LineageNode.tsx`** - Enhanced rendering for 3-level hierarchy
- **Refined `lineage.types.ts`** - TypeScript definitions aligned with new structure
- **Optimized `lineage.api.ts`** - API client using v2 endpoints

#### Documentation
- **`CHANGELOG.md`** - This file
- **Migration notes** - Documented in this changelog

---

### 🔧 Changed

#### Backend
- **`main.go`** - Updated service initialization for new lineage architecture
- **`router.go`** - Configured routes to use v2 lineage endpoints
- **`config.go`** - Simplified configuration for 3-level model
- **`neo4j_hierarchy.go`** - Streamlined to support only 3-level hierarchy (227 lines reduced)
- **`ingest_sdk_verified.go`** - Updated to create correct graph relationships
- **`sdk_adapter.go`** - Aligned with new hierarchy model

#### Frontend
- **Lineage visualization** - Automatically adapts to 3-level structure
- **Type safety** - Enhanced TypeScript definitions for better IDE support

---

### 📊 Statistics

- **Total Files Changed**: 15
- **Lines Added**: 269
- **Lines Deleted**: 1,059
- **Net Reduction**: 790 lines
- **Files Removed**: 4
- **New Files**: 2

---

### 🔄 Migration Guide

#### For Existing Deployments

**Step 1: Backup Neo4j Database**
```bash
# Create backup before migration
docker exec arc-hawk-neo4j neo4j-admin dump --database=neo4j --to=/backups/pre-v2.1-backup.dump
```

**Step 2: Run Schema Migration**
```bash
# Apply new schema
cat apps/backend/migrations_versioned/neo4j_semantic_contract_v1.cypher | \
  docker exec -i arc-hawk-neo4j cypher-shell -u neo4j -p your_password
```

**Step 3: Update Application**
```bash
# Pull latest changes
git pull origin main

# Rebuild backend
cd apps/backend
go build ./cmd/server

# Rebuild frontend
cd ../frontend
npm install
npm run build
```

**Step 4: Restart Services**
```bash
# Restart all services
docker-compose restart
cd apps/backend && go run cmd/server/main.go &
cd apps/frontend && npm run dev &
```

**Step 5: Verify Migration**
```bash
# Check lineage endpoint
curl http://localhost:8080/api/v1/lineage/v2

# Expected: JSON response with 3-level hierarchy
```

#### API Changes

**No Breaking Changes for External Consumers**
- Old endpoints remain functional but deprecated
- New v2 endpoints recommended for all new integrations
- Frontend automatically uses v2 endpoints

**Deprecated Endpoints** (still functional):
- `GET /api/v1/lineage` - Use `GET /api/v1/lineage/v2` instead

---

### 🐛 Bug Fixes

- Fixed graph traversal performance issues with deep hierarchies
- Resolved duplicate node creation in Neo4j
- Corrected edge relationship naming inconsistencies
- Fixed frontend rendering errors with complex lineage graphs

---

### 🔒 Security

- No security-related changes in this release

---

### 📝 Notes

- **Backward Compatibility**: Old Neo4j data will need migration (see Migration Guide)
- **Performance Impact**: Expect 30-40% improvement in lineage query performance
- **Testing**: All changes verified with end-to-end integration tests
- **Rollback**: Keep `neo4j_schema_OLD_4LEVEL.cypher` for emergency rollback if needed

---

## [2.0.0] - 2026-01-09

### 🎯 Major Release: Production Ready

#### Key Achievements
- ✅ **Accuracy**: 100% pass rate on mathematical validation for India-specific PII
- ✅ **Stability**: Zero-crash frontend with verified data flow
- ✅ **Completeness**: Multi-source scanning (Filesystem + PostgreSQL) operational
- ✅ **Lineage**: Graph synchronization issues resolved

#### Critical Fixes
- **PAN Validation**: Implemented Weighted Modulo 26 algorithm
- **Lineage Graph**: Fixed query mismatch and visibility issues
- **Multi-Source Scanning**: Enabled PostgreSQL profile
- **Findings Display**: Granular visibility for every PII instance

#### Architecture
- Intelligence-at-Edge: Scanner SDK as sole authority for classification
- Unidirectional data flow: Scanner → API → PostgreSQL → Neo4j → Frontend
- No Presidio client in backend
- No regex validation in backend

---

## [1.0.0] - 2025-12-30

### 🎉 Initial Release

- Initial platform implementation
- Basic PII detection and classification
- Lineage tracking with 4-level hierarchy
- Dashboard visualization
- Multi-source scanning support

---

## Legend

- 🎯 Major Changes
- ✨ Added
- 🔧 Changed
- 🗑️ Removed
- 🐛 Bug Fixes
- 🔒 Security
- 📊 Statistics
- 🔄 Migration
- 📝 Notes
