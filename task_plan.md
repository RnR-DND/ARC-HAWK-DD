# 📋 task_plan.md - ARC-Hawk Project Blueprint

**Status:** Phase 2: Agentic Integration (In Progress)
**Created:** 2026-01-22
**Last Updated:** 2026-02-10
**Version:** 3.0.0

---

## 🎯 Project Overview

**Mission:** Build and maintain an enterprise-grade PII discovery, classification, and lineage tracking platform using "Intelligence-at-Edge" architecture with 100% validation accuracy.

**Scope:** Multi-component platform with scanner (Python), backend (Go), frontend (Next.js), and infrastructure (Docker)

---

## 🏗️ Phase 1: Blueprint (Complete - CORRECTED)

### Goals Completed (CORRECTED)

1. **North Star Defined**
   - Enterprise PII discovery and lineage tracking
   - 100% validation accuracy target
   - Zero false positives guarantee

2. **Discovery Questions Answered (CORRECTED)**
   - Integrations mapped (PostgreSQL, Neo4j, Temporal, Presidio)
   - **CRITICAL:** Each data source requires UNIQUE connection parameters (NOT identical)
   - Source of Truth identified (PostgreSQL + Scanner SDK)
   - Delivery mechanisms specified (REST API + Next.js Dashboard)
   - Behavioral rules documented (Intelligence-at-Edge, Unidirectional Flow, Severity Rules Engine)

3. **Architecture Defined**
   - 3-layer architecture (Tools, Navigation, Architecture SOPs)
   - Data flow: Scanner → Backend → PostgreSQL → Neo4j → Frontend
   - 12 backend modules (130+ files): scanning, connections, assets, lineage, compliance, remediation, auth, analytics, masking, fplearning, websocket, shared

4. **Data Schemas Finalized (CORRECTED)**
   - Trigger schema (input)
   - Ingestion schema (output payload)
   - **CRITICAL:** Each data source has UNIQUE connection parameters - see connection.yml.sample
   - Finding schema (database)
   - Classification summary schema

### Deliverables

- [x] `gemini.md` - Project Constitution (Complete - CORRECTED)
- [x] `task_plan.md` - This file (Complete - CORRECTED)
- [x] Architecture diagram
- [x] Integration matrix (with CORRECT connection schemas)
- [x] Performance specifications

---

## 🤖 Phase 2: Agentic Integration (In Progress)

### Objectives

Integrate and configure the new Agentic Operation System to boost productivity.

### Tasks

#### 2.1 Tool Configuration
- [x] **2.1.1** Install MCP Tools
  - Tools: Supermemory, Get Shit Done, Ralph, Antigravity Awesome Skills, Hive, Superpowers
  - Location: `~/.agent`
  - Status: ✅ Complete

- [x] **2.1.2** Configure MCP Server
  - File: `mcp_config_snippet.json`
  - Server: `filesystem` pointing to `agent_tools`
  - Status: ✅ Complete

- [x] **2.1.3** Deep Codebase Analysis
  - Backend: 12 Go modules, 130+ files
  - Scanner: 14 connectors, 11 recognizers, 15 validators, Flask API
  - Frontend: 12 routes, 26+ components, 11 service APIs
  - Infrastructure: 7 Docker containers, Prometheus + Grafana
  - Status: ✅ Complete

- [x] **2.1.4** Map Agent Tools to B.L.A.S.T. Phases
  - GSD → Blueprint/Architect phases (planning, research)
  - Ralph → Link/Stylize phases (autonomous execution)
  - Hive → Trigger phase (multi-agent deploy)
  - Skills: 90+ relevant mapped (golang-pro, python-pro, nextjs, docker, security, etc.)
  - Status: ✅ Complete

- [ ] **2.1.5** Initialize GSD Project
  - Command: `npx get-shit-done-cc`
  - Output: `PROJECT.md`, `REQUIREMENTS.md`
  - Action: Map existing `gemini.md` content to GSD format

- [ ] **2.1.6** Configure Ralph
  - Action: Create `scripts/ralph` directory
  - Action: Copy `ralph.sh` and prompts

#### 2.2 Workflow Adoption
- [ ] **2.2.1** Update System Prompt
  - Action: Add skill-trigger rules from `gemini.md`
  - Load `@brainstorming` BEFORE all creative work
  - Load language-specific skills per component
- [ ] **2.2.2** Create First PRD with GSD
  - Target: Phase 3 (Link) verification scripts

---

## ⚡ Phase 3: Link (Connectivity Verification) — Tools: `@docker-expert`, `@systematic-debugging`

### Objectives

Verify all infrastructure connections and API endpoints are operational before proceeding to full logic implementation.

### Tasks

#### 2.1 Infrastructure Connectivity

**Goal:** Verify Docker services are running and healthy

- [ ] **2.1.1** Start PostgreSQL container
  - Command: `docker-compose up -d postgres`
  - Verify: `pg_isready -U postgres`
  - Expected: Connection on port 5432
  - Location: `docker-compose.yml:18-37`

- [ ] **2.1.2** Start Neo4j container
  - Command: `docker-compose up -d neo4j`
  - Verify: `cypher-shell -u neo4j -p password123 'RETURN 1'`
  - Expected: Connection on ports 7474, 7687
  - Location: `docker-compose.yml:39-60`

- [ ] **2.1.3** Start Temporal workflow engine
  - Command: `docker-compose up -d temporal temporal-ui`
  - Verify: `tctl --address temporal:7233 cluster health`
  - Expected: Health check passes
  - Location: `docker-compose.yml:82-118`

- [ ] **2.1.4** Start Presidio ML analyzer (optional)
  - Command: `docker-compose up -d presidio-analyzer`
  - Verify: `wget --quiet --tries=1 --spider http://localhost:5001/health`
  - Expected: HTTP 200 response
  - Location: `docker-compose.yml:62-80`

#### 2.2 Backend Connectivity

**Goal:** Verify Go backend can connect to infrastructure

- [ ] **2.2.1** Test PostgreSQL connection
  - File: `apps/backend/modules/shared/infrastructure/persistence/postgres.go`
  - Method: `NewPostgresRepository()`
  - Test: Run `go run cmd/server/main.go`
  - Verify: No connection errors in logs

- [ ] **2.2.2** Test Neo4j connection
  - File: `apps/backend/modules/shared/infrastructure/neo4j.go`
  - Method: `NewNeo4jConnection()`
  - Test: Run backend and check logs
  - Verify: "Connected to Neo4j" message

- [ ] **2.2.3** Test Temporal connection
  - File: `apps/backend/modules/shared/infrastructure/temporal.go`
  - Method: `NewTemporalClient()`
  - Test: Run backend and check logs
  - Verify: Temporal client initialized

#### 2.3 Scanner Connectivity

**Goal:** Verify Python scanner can reach backend API

- [ ] **2.3.1** Test backend API availability
  - Command: `curl http://localhost:8080/api/v1/health`
  - Expected: JSON health response
  - Location: `apps/backend/modules/shared/api/health.go`

- [ ] **2.3.2** Test scan ingestion endpoint
  - Command: `curl -X POST http://localhost:8080/api/v1/scans/ingest-verified -H "Content-Type: application/json" -d '{"test": true}'`
  - Expected: Success response or validation error (not 404)
  - Location: `apps/backend/modules/scanning/api/sdk_ingest_handler.go`

- [ ] **2.3.3** Verify scanner configuration
  - **CRITICAL:** Reference `apps/scanner/config/connection.yml.sample` (NOT connection.yml)
  - Check: Each source type has UNIQUE connection parameters
  - Check: PostgreSQL requires host, port, user, password, database, tables
  - Check: S3 requires access_key, secret_key, bucket_name
  - Check: Slack requires token, channel_types, channel_ids

#### 2.4 Frontend Connectivity

**Goal:** Verify Next.js dashboard can reach backend API

- [ ] **2.4.1** Test API URL configuration
  - File: `apps/frontend/next.config.js`
  - Variable: `NEXT_PUBLIC_API_URL`
  - Expected: `http://localhost:8080/api/v1`

- [ ] **2.4.2** Test API client
  - File: `apps/frontend/utils/api-client.ts`
  - Test: Run `npm run dev` and check browser console
  - Verify: No CORS errors

#### 2.5 Build Handshake Scripts

**Goal:** Create minimal scripts to verify connectivity

- [ ] **2.5.1** Create `tools/verify-postgres.sh`
  - Purpose: Test PostgreSQL connection
  - Output: Connection status, latency

- [ ] **2.5.2** Create `tools/verify-neo4j.sh`
  - Purpose: Test Neo4j connection
  - Output: Connection status, version

- [ ] **2.5.3** Create `tools/verify-backend.sh`
  - Purpose: Test backend API health
  - Output: API status, response time

- [ ] **2.5.4** Create `tools/verify-scanner.sh`
  - Purpose: Test scanner-backend integration
  - Output: Ingestion success/failure

### Expected Outputs

- Connectivity verification report
- Latency metrics for each service
- Error logs for failed connections
- Updated `.env` with working credentials

### Dependencies

- Docker Compose v3.8+
- Go 1.24+
- Python 3.9+
- Node.js 18+

---

## ⚙️ Phase 4: Architect (3-Layer Build) — Tools: `@golang-pro`, `@python-pro`, `@api-design-principles`

### Objectives

Implement deterministic automation logic following the 3-layer architecture.

### Layer 1: Architecture SOPs

**Location:** `architecture/*.md`

- [ ] **3.1.1** Create `architecture/scanning-sop.md`
  - Purpose: Define scan execution workflow
  - Include: Input validation, error handling, retry logic
  - **NOTE:** Each source type has unique connection handling

- [ ] **3.1.2** Create `architecture/ingestion-sop.md`
  - Purpose: Define data ingestion flow
  - Include: Schema validation, deduplication, enrichment

- [ ] **3.1.3** Create `architecture/lineage-sop.md`
  - Purpose: Define graph building workflow
  - Include: Node creation, edge linking, traversal

- [ ] **3.1.4** Create `architecture/compliance-sop.md`
  - Purpose: Define compliance mapping workflow
  - Include: DPDPA 2023 rules, consent tracking

### Layer 2: Navigation (Decision Making)

**Location:** `tools/navigation/*.py`

- [ ] **3.2.1** Create `route_scans.py`
  - Purpose: Route scan jobs to appropriate handlers
  - Logic: Source detection → Handler selection (based on connection.yml.sample schemas)

- [ ] **3.2.2** Create `route_findings.py`
  - Purpose: Route findings to classification
  - Logic: Pattern matching → Validation type selection

- [ ] **3.2.3** Create `route_compliance.py`
  - Purpose: Route findings to compliance rules
  - Logic: PII type → DPDPA category mapping

### Layer 3: Tools (Deterministic Scripts)

**Location:** `tools/*.py`

- [ ] **3.3.1** Scanner Tools (CORRECTED - each source has unique connection handling)
  - `scan_filesystem.py` - Filesystem scanning (path, exclude_patterns)
  - `scan_postgresql.py` - PostgreSQL scanning (host, port, user, password, database, tables)
  - `scan_mysql.py` - MySQL scanning (host, port, user, password, database, tables, exclude_columns)
  - `scan_mongodb.py` - MongoDB scanning (uri OR host, port, username, password, database, collections)
  - `scan_s3.py` - S3 scanning (access_key, secret_key, bucket_name, exclude_patterns)
  - `scan_gcs.py` - GCS scanning (credentials_file, bucket_name, exclude_patterns)
  - `scan_slack.py` - Slack scanning (token, channel_types, channel_ids, limit_mins)
  - `scan_redis.py` - Redis scanning (host, password)
  - `validate_pii.py` - Mathematical validation

- [ ] **3.3.2** Ingestion Tools
  - `ingest_findings.py` - Batch ingestion
  - `enrich_findings.py` - Metadata enrichment
  - `deduplicate.py` - Finding deduplication

- [ ] **3.3.3** Lineage Tools
  - `build_graph.py` - Neo4j graph building
  - `link_lineage.py` - Asset-PII relationship linking
  - `traverse_lineage.py` - Lineage traversal

- [ ] **3.3.4** Compliance Tools
  - `map_dpdpa.py` - DPDPA 2023 mapping
  - `track_consent.py` - Consent tracking
  - `generate_report.py` - Compliance report generation

---

## ✨ Phase 5: Stylize (Refinement) — Tools: `@nextjs-best-practices`, `@react-patterns`

### Objectives

Format outputs professionally and apply UI/UX improvements.

### Tasks

#### 4.1 API Response Formatting

- [ ] **4.1.1** Standardize JSON responses
  - Structure: `{ success: bool, data: any, error: string|null }`
  - Location: `apps/backend/pkg/response/`

- [ ] **4.1.2** Add pagination to list endpoints
  - Fields: `page`, `page_size`, `total`, `total_pages`
  - Location: `apps/backend/modules/*/api/`

- [ ] **4.1.3** Add error codes
  - Format: `ERR_XXX: Description`
  - Location: `apps/backend/pkg/errors/`

#### 4.2 Dashboard Styling

- [ ] **4.2.1** Apply consistent color scheme
  - PII Types: Color-coded by severity
  - High: Red (#EF4444)
  - Medium: Yellow (#F59E0B)
  - Low: Green (#10B981)

- [ ] **4.2.2** Improve data visualizations
  - Risk distribution: Pie chart
  - Lineage graph: Cytoscape.js
  - Findings list: Data table with sorting

- [ ] **4.2.3** Add responsive design
  - Mobile: Hamburger menu, stacked layouts
  - Tablet: Adaptive grid
  - Desktop: Full dashboard view

#### 4.3 Slack/Jira Notification Formatting

- [ ] **4.3.1** Format Slack alerts
  - Blocks: Section, Divider, Actions
  - Include: Asset name, PII type, severity, matches count
  - Reference: `connection.yml.sample` notify.slack configuration

- [ ] **4.3.2** Format Jira issues
  - Fields: summary, description, labels, assignee
  - Reference: `connection.yml.sample` notify.jira configuration

---

## 🛰️ Phase 6: Trigger (Deployment) — Tools: `@docker-expert`, `@security-auditor`

### Objectives

Deploy to production and set up automation.

### Tasks

#### 5.1 Cloud Deployment

- [ ] **5.1.1** Build Docker images
  - Backend: `apps/backend/Dockerfile`
  - Frontend: `apps/frontend/Dockerfile`
  - Scanner: `apps/scanner/Dockerfile`

- [ ] **5.1.2** Push to container registry
  - Registry: GitHub Container Registry or Docker Hub
  - Tags: `latest`, `2.1.0`, `sha-{commit}`

- [ ] **5.1.3** Deploy to Kubernetes
  - Files: `infra/kubernetes/*.yaml`
  - Services: Backend, Frontend, Scanner workers

#### 5.2 Automation Setup

- [ ] **5.2.1** Configure cron jobs
  - Scan schedule: Daily at 2 AM
  - Report generation: Weekly on Monday
  - Cleanup: Monthly

- [ ] **5.2.2** Set up webhooks
  - GitHub: CI/CD triggers
  - Slack: Alert notifications (reference connection.yml.sample)
  - Jira: Issue creation (reference connection.yml.sample)
  - Custom: API triggers

- [ ] **5.2.3** Configure monitoring
  - Metrics: Prometheus
  - Visualization: Grafana
  - Alerts: PagerDuty

#### 5.3 Documentation

- [ ] **5.3.1** Update `gemini.md` with deployment details
- [ ] **5.3.2** Create runbooks for common operations
- [ ] **5.3.3** Document rollback procedures

---

## 📊 Milestone Tracking

### Phase Completion Criteria

| Phase | Criteria | Status |
|-------|----------|--------|
| **Blueprint** | All 5 discovery questions answered (CORRECTED) | ✅ Complete |
| **Agentic Integration** | All tools mapped, GSD/Ralph configured | 🔄 In Progress |
| **Link** | All services responding to health checks | ⏳ Pending |
| **Architect** | All tools implemented and tested | ⏳ Pending |
| **Stylize** | Dashboard receives valid data | ⏳ Pending |
| **Trigger** | Production deployment complete | ⏳ Pending |

### Key Metrics

- **Scan Throughput:** 200-350 files/second
- **Validation Accuracy:** 100% (zero false positives)
- **API Latency:** <100ms (p95)
- **Dashboard Load Time:** <3 seconds

### Dependencies Between Phases

```
Phase 1 (Blueprint) → Phase 2 (Agentic) → Phase 3 (Link) → Phase 4 (Architect) → Phase 5 (Stylize) → Phase 6 (Trigger)
       ↓                    ↓                   ↓                   ↓                    ↓                   ↓
   gemini.md          GSD+Ralph          verify scripts         tools/*.py          dashboard          deployment
   task_plan.md       skill-triggers     connectivity            navigation          formatting         automation
```

---

## 🔧 Resource Requirements

### Development Environment

- **CPU:** 4+ cores
- **RAM:** 16+ GB
- **Storage:** 50+ GB SSD
- **Network:** 100+ Mbps

### Production Environment

- **CPU:** 8+ cores
- **RAM:** 32+ GB
- **Storage:** 500+ GB SSD
- **Network:** 1+ Gbps

### External Services

- PostgreSQL 15 (managed or self-hosted)
- Neo4j 5.15 (managed or self-hosted)
- Temporal 1.22 (optional, for workflows)
- Presidio (optional, for ML analysis)

---

## 📝 Change Log

| Date | Phase | Change | Author |
|------|-------|--------|--------|
| 2026-01-22 | Blueprint | Initial plan creation (CORRECTED) | AI Analysis |
| 2026-01-22 | Blueprint | Added all discovery answers (CORRECTED) | AI Analysis |
| 2026-01-22 | Blueprint | Defined data schemas (CORRECTED - each source has unique params) | AI Analysis |
| 2026-01-22 | Blueprint | Corrected connection.yml.sample reference | AI Analysis |
| 2026-02-10 | Agentic | Deep codebase analysis (12 modules, 14 connectors, 12 routes) | AI Analysis |
| 2026-02-10 | Agentic | Mapped agent tools → B.L.A.S.T. phases, 90+ skills identified | AI Analysis |
| 2026-02-10 | Agentic | Updated gemini.md v3.0.0 with tool-to-phase + skill-trigger rules | AI Analysis |

---

## 🎯 Next Actions

### Immediate (This Week)

1. **Run connectivity verification**
   ```bash
   docker-compose up -d
   ```

2. **Test backend startup**
   ```bash
   cd apps/backend && go run cmd/server/main.go
   ```

3. **Test frontend startup**
   ```bash
   cd apps/frontend && npm run dev
   ```

4. **Verify scanner configuration**
   ```bash
   cat apps/scanner/config/connection.yml.sample  # CRITICAL - use SAMPLE file
   ```

### Short-Term (This Month)

1. Complete Phase 2 (Link)
2. Build verification scripts
3. Begin Phase 3 (Architect)

### Medium-Term (This Quarter)

1. Complete all B.L.A.S.T. phases
2. Deploy to staging environment
3. User acceptance testing

---

**CRITICAL REMINDER:** Always reference `apps/scanner/config/connection.yml.sample` for connection schemas. Each data source type requires UNIQUE parameters - they are NOT identical.

*This plan is a living document and should be updated as the project evolves.*
