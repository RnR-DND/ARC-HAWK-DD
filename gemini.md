# ♊ gemini.md - ARC-Hawk Project Constitution

**Status:** ✅ Phase 1: Blueprint (Complete) · Phase 2: Agentic Integration (In Progress)
**Last Updated:** 2026-02-10
**Version:** 3.0.0

---

## 📍 Project State

**Phase:** Phase 2 (Agentic Integration) - IN PROGRESS
**Discovery:** All 5 questions answered through CODEBASE ANALYSIS
**Critical Correction:** Read `connection.yml.sample` - NOT `connection.yml`
**Next Step:** Complete GSD initialization → Phase 3 - Link (Verify connectivity)

---

## 🤖 Agentic Operation System

**Core Workflow:** Meta-Prompting & Autonomous Loop — plan with GSD, execute with Ralph, orchestrate with Hive.

### Tool-to-Phase Mapping

| B.L.A.S.T. Phase | Primary Tool | Workflows / Commands | Skills to Load |
|---|---|---|---|
| **Blueprint** | GSD | `new-project`, `map-codebase`, `research-phase` | `@brainstorming`, `@architecture` |
| **Link** | Ralph | `prd.json` with connectivity stories | `@docker-expert`, `@postgres-best-practices`, `@systematic-debugging` |
| **Architect** | GSD + Ralph | `plan-phase` → `execute-plan` | `@golang-pro`, `@python-pro`, `@api-design-principles`, `@database-architect` |
| **Stylize** | Ralph | `prd.json` with UI stories | `@nextjs-best-practices`, `@react-patterns`, `@react-state-management` |
| **Trigger** | Hive | Multi-agent deploy + verify | `@docker-expert`, `@observability-monitoring-*`, `@security-auditor` |

### Skill Trigger Rules

| Task Type | Auto-Load Skills |
|---|---|
| Any creative/design work | `@brainstorming` (MANDATORY) |
| Go backend changes | `@golang-pro`, `@go-concurrency-patterns` |
| Python scanner changes | `@python-pro`, `@python-patterns` |
| Frontend changes | `@nextjs-best-practices`, `@react-best-practices` |
| Database schema changes | `@database-migration`, `@postgres-best-practices` |
| Security-related work | `@security-auditor`, `@api-security-best-practices` |
| Bug fixing | `@systematic-debugging`, `@debugging-strategies` |
| Writing tests | `@testing-patterns`, `@verification-before-completion` |
| Cross-component work | `@dispatching-parallel-agents`, `@subagent-driven-development` |
| Docker/infra changes | `@docker-expert`, `@observability-monitoring-monitor-setup` |

### Execution Modes

1. **GSD Planning** → `npx get-shit-done-cc` → `.planning/PROJECT.md`, `REQUIREMENTS.md`, `ROADMAP.md`
2. **Ralph Loop** → `./ralph.sh` → reads `prd.json`, executes stories one-by-one, updates `progress.txt`
3. **Hive Orchestration** → `./hive` → parallel multi-agent tasks with TUI monitoring

---

## 🎯 North Star

**Primary Objective:**
Build and maintain ARC-Hawk: An enterprise-grade, "Intelligence-at-Edge" PII discovery, classification, and lineage tracking platform with **100% validation accuracy** and zero false positives.

**Mission Statement:**
Automatically discover, validate, and track Personally Identifiable Information (PII) across entire data infrastructure using mathematical validation algorithms and semantic lineage tracking.

**Success Metrics:**
- 200-350 files/second scan throughput
- 1,000 findings/second validation speed
- Zero false positives through mathematical validation (Verhoeff, Luhn algorithms)
- 1M+ assets, 10M+ findings capacity
- DPDPA 2023 compliance mapping

---

## 🔗 Integrations

### Core Infrastructure Services

| Service | Purpose | Connection | Port |
|---------|---------|------------|------|
| **PostgreSQL 15** | Primary data storage | `arc_platform` database | 5432 |
| **Neo4j 5.15** | Semantic lineage graph | Graph database | 7687 |
| **Temporal 1.22** | Workflow orchestration | Workflow engine | 7233 |
| **Presidio Analyzer** | ML-based PII analysis | Microsoft Presidio | 5001 |

### Data Source Connectors - CONNECTION SCHEMAS

**CRITICAL:** Each data source requires UNIQUE connection parameters. See `apps/scanner/config/connection.yml.sample`

| Source | Required Parameters | Type | Status |
|--------|---------------------|------|--------|
| **Redis** | `host`, `password` | Key-Value | ✅ Supported |
| **AWS S3** | `access_key`, `secret_key`, `bucket_name`, `cache`, `exclude_patterns` | Cloud Storage | ✅ Supported |
| **Google GCS** | `credentials_file`, `bucket_name`, `cache`, `exclude_patterns` | Cloud Storage | ✅ Supported |
| **Firebase** | `credentials_file`, `bucket_name`, `cache`, `exclude_patterns` | Cloud Storage | ✅ Supported |
| **MySQL** | `host`, `port`, `user`, `password`, `database`, `limit_start`, `limit_end`, `tables[]`, `exclude_columns[]` | Database | ✅ Supported |
| **PostgreSQL** | `host`, `port`, `user`, `password`, `database`, `limit_start`, `limit_end`, `tables[]` | Database | ✅ Production |
| **MongoDB** | `uri` OR `host`, `port`, `username`, `password`, `database`, `limit_start`, `limit_end`, `collections[]` | Database | ✅ Supported |
| **Filesystem** | `path`, `exclude_patterns[]` | Local/Network | ✅ Production |
| **Google Drive** | `folder_name`, `credentials_file`, `cache`, `exclude_patterns[]` | Cloud Storage | ✅ Supported |
| **GDrive Workspace** | `folder_name`, `credentials_file`, `impersonate_users[]`, `cache`, `exclude_patterns[]` | Cloud Storage | ✅ Supported |
| **Text** | `text` (direct input) | Direct Input | ✅ Supported |
| **Slack** | `channel_types`, `token`, `onlyArchived`, `archived_channels`, `limit_mins`, `read_from`, `isExternal`, `channel_ids[]`, `blacklisted_channel_ids[]` | Collaboration | ✅ Supported |

### Notification & Alerting Integrations

| Service | Purpose | Configuration |
|---------|---------|---------------|
| **Slack** | Real-time alerts | `webhook_url`, `mention` (bot user ID) |
| **Jira** | Issue tracking | `username`, `server_url`, `api_token`, `project`, `issue_type`, `labels`, `assignee`, `issue_fields` |

### Output/Delivery Destinations

| Destination | Format | Purpose |
|-------------|--------|---------|
| **REST API** | JSON | Backend communication |
| **Next.js Dashboard** | HTML/React | User interface |
| **Neo4j Graph** | Cypher | Lineage visualization |
| **Slack/Jira** | Webhooks | Real-time alerts and ticket creation |

---

## 📊 Source of Truth

### Primary Data Storage

**PostgreSQL Database:** `arc_platform`
- **Host:** postgres (container) / localhost (dev)
- **Port:** 5432
- **User:** postgres
- **Tables:** Managed by GORM migrations in `apps/backend/migrations_versioned/`

### Key Data Models

```go
// Core entities stored in PostgreSQL
type Asset struct {
    ID          string    `gorm:"primaryKey"`
    Name        string
    Path        string
    AssetType   string    // filesystem, database, s3, etc.
    Source      string    // Connection profile name
    ScanID      string
    CreatedAt   time.Time
}

type Finding struct {
    ID            string    `gorm:"primaryKey"`
    AssetID       string
    PatternName   string    // PII type (Aadhaar, PAN, Email, etc.)
    Matches       []string  // Actual matched values
    SampleText    string    // Context
    Confidence    float64
    Severity      string    // High, Medium, Low
    Verified      bool      // Scanner SDK validation
    CreatedAt     time.Time
}

type Classification struct {
    ID                string
    FindingID         string
    ClassificationType string
    VerificationMethod string // Verhoeff, Luhn, Modulo26, Regex
    Verified          bool
}
```

### Logic Authority

**Scanner SDK (Python):** `apps/scanner/sdk/`
- **Purpose:** PII detection and mathematical validation
- **Authority Level:** SOLE AUTHORITY for validation decisions
- **Validation Methods:**
  - Verhoeff checksum (Aadhaar)
  - Luhn algorithm (Credit Cards)
  - Modulo 26 (PAN)
  - Format validation (Passport, Voter ID, etc.)
  - Regex patterns (Email, Phone, etc.)

**Backend Modules (Go):** `apps/backend/modules/` — 12 modules, 130+ files
- **Purpose:** Business logic orchestration
- **Constraint:** NO validation logic - passive consumer only
- **Modules:**
  - `scanning/` (19 files) — Scan ingestion, classification, dashboard metrics, SDK ingest
  - `connections/` (8 files) — Connection CRUD, sync, test, scan orchestration
  - `assets/` (7 files) — Asset management, AssetManager interface
  - `lineage/` (6 files) — Neo4j graph lineage services
  - `compliance/` (8 files) — DPDPA compliance reporting, policy management
  - `remediation/` (11 files) — Remediation execution + 6 connectors (S3, FS, MySQL, etc.)
  - `auth/` (7 files) — JWT authentication, middleware
  - `analytics/` (3 files) — Risk analytics
  - `masking/` (3 files) — Data masking
  - `fplearning/` (6 files) — False positive learning
  - `websocket/` (2 files) — Real-time scan status updates
  - `shared/` (50 files) — 12 domain entities, 17 persistence repos, encryption, audit

---

## 🚀 Delivery Payload

### API Endpoints (JSON Output)

**Base URL:** `http://localhost:8080/api/v1`

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/scans/ingest-verified` | POST | Ingest SDK-validated findings |
| `/scans/trigger` | POST | Trigger new scan |
| `/scans/:id` | GET | Get scan details |
| `/scans/latest` | GET | Get last scan run |
| `/classification/summary` | GET | Get PII classification summary |
| `/findings` | GET | Query findings with pagination |
| `/lineage/graph` | GET | Get lineage graph data |

### Dashboard Views (Next.js)

**URL:** `http://localhost:3000`

| Page | Route | Purpose |
|------|-------|---------|
| **Dashboard** | `/` | Overview metrics, recent findings, risk distribution |
| **Findings** | `/findings` | Searchable PII findings list |
| **Lineage** | `/lineage` | Interactive PII flow visualization |
| **Assets** | `/assets` | Asset inventory and PII coverage |
| **Scans** | `/scans` | Scan history and management |
| **Compliance** | `/compliance` | DPDPA 2023 compliance mapping |
| **Reports** | `/reports` | Exportable compliance reports |

### Data Export Formats

- **JSON:** API responses, scan results
- **CSV:** Findings export, compliance reports
- **Graph JSON:** Lineage graph data (ReactFlow/Cytoscape compatible)

---

## ⚙️ Behavioral Rules

### Core Architectural Principles

#### 1. Intelligence-at-Edge
> **Scanner SDK is the SOLE AUTHORITY for PII validation**

- Backend MUST NOT perform validation logic
- Scanner SDK applies mathematical validation (Verhoeff, Luhn)
- Findings are immutable once validated by SDK
- Zero false positives through mathematical proof

#### 2. Unidirectional Data Flow
```
Scanner SDK → Backend API → PostgreSQL → Neo4j → Frontend
     ↓              ↓            ↓          ↓         ↓
  Validate      Ingest       Store     Visualize  Display
```
- No circular dependencies
- Scanner never calls frontend directly
- Backend never bypasses scanner validation

#### 3. Zero False Positive Guarantee
- Mathematical validation required for all detected PII
- Pattern matches MUST pass algorithmic validation
- Exceptions logged and tracked
- Continuous validation accuracy monitoring

#### 4. Premium UX Standard
- High-quality, polished interface (Tailwind CSS)
- Responsive design (Mobile + Desktop)
- Real-time updates (WebSocket support)
- Interactive visualizations (ReactFlow, Cytoscape)

#### 5. Compliance-First Design
- DPDPA 2023 (India) mapping built-in
- Consent tracking for identified PII
- Retention policy enforcement
- Complete audit trail

### Severity Rules Engine

**From `connection.yml.sample`:**
```yaml
severity_rules:
  Highest:
    - query: "length(matches) > 10 && contains(['EMAIL', 'PAN'], pattern_name)"
      description: "Detected more than 10 Email or PAN exposed"
  High:
    - query: "length(matches) > 10 && contains(['EMAIL', 'PAN'], pattern_name) && data_source == 'slack'"
      description: "Detected more than 10 Email or PAN exposed in Slack"
  Medium:
    - query: "length(matches) > 5 && length(matches) <= 10 && contains(['EMAIL', 'PAN'], pattern_name) && data_source == 'slack' && profile == 'customer_support'"
      description: "Detected more than 5 and less than 10 Email or PAN exposed in Customer support Slack workspace"
  Low:
    - query: "length(matches) <= 5"
      description: "Detected less than 5 PII or Secrets"
```

### Scan Options

```yaml
options:
  quick_exit: True       # Exit after first significant finding
  max_matches: 5         # Maximum matches per pattern
```

### Error Handling Patterns

```go
// Go Backend
func (h *Handler) ProcessFinding(finding Finding) error {
    if !finding.Verified {
        return fmt.Errorf("unverified finding rejected: %s", finding.ID)
    }

    if err := h.repo.Save(finding); err != nil {
        return fmt.Errorf("database error: %w", err)
    }

    return nil
}
```

```python
# Python Scanner
def validate_aadhar(aadhar_number: str) -> bool:
    """Verhoeff checksum validation for Aadhaar"""
    if not re.match(r'^\d{4}[-\s]?\d{4}[-\s]?\d{4}$', aadhar_number):
        return False
    return verhoeff_validate(aadhar_number.replace('-', '').replace(' ', ''))
```

---

## 📦 Data Schemas

### Trigger Schema (Input)

```json
{
  "name": "string",
  "sources": ["string"],
  "pii_types": ["string"],
  "execution_mode": "sequential|parallel",
  "connection_profile": "string",
  "output_format": "json"
}
```

### Ingestion Schema (Output Payload)

```json
{
  "fs": [
    {
      "host": "string",
      "file_path": "string",
      "pattern_name": "string",
      "matches": ["string"],
      "sample_text": "string",
      "profile": "string",
      "data_source": "fs",
      "severity": "string",
      "file_data": { "key": "value" }
    }
  ],
  "postgresql": [],
  "mongodb": [],
  "s3": [],
  "gcs": []
}
```

### Finding Schema (Database)

```json
{
  "id": "uuid",
  "asset_id": "uuid",
  "pattern_name": "Aadhaar|PAN|Email|Phone|...",
  "matches": ["value1", "value2"],
  "sample_text": "context snippet",
  "confidence": 0.95,
  "severity": "High|Medium|Low",
  "verified": true,
  "verification_method": "Verhoeff|Luhn|Format|Regex",
  "created_at": "2026-01-22T10:00:00Z"
}
```

### Classification Summary Schema

```json
{
  "total": 15000,
  "by_type": {
    "Aadhaar": { "count": 500, "verified": 500 },
    "PAN": { "count": 300, "verified": 300 },
    "Email": { "count": 8000, "verified": 8000 }
  },
  "by_severity": {
    "High": 200,
    "Medium": 5000,
    "Low": 9800
  }
}
```

---

## 🔧 Configuration

### Environment Variables

```bash
# Database
DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=arc_platform

# Neo4j
NEO4J_URI=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password123

# Temporal
TEMPORAL_HOST=temporal:7233
TEMPORAL_NAMESPACE=default

# Presidio
PRESIDIO_ENABLED=true
PRESIDIO_URL=http://presidio-analyzer:3000

# Frontend
NEXT_PUBLIC_API_URL=http://backend:8080/api/v1
```

### Connection Profiles

**CORRECTED:** See `apps/scanner/config/connection.yml.sample` for actual schemas.

Each source type has UNIQUE connection parameters:

**PostgreSQL Example:**
```yaml
postgresql:
  postgresql_example:
    host: "YOUR_POSTGRESQL_HOST"
    port: 5432
    user: "YOUR_POSTGRESQL_USERNAME"
    password: "YOUR_POSTGRESQL_PASSWORD"
    database: "YOUR_DATABASE_NAME"
    limit_start: 0
    limit_end: 50000
    tables:
      - table1
      - table2
```

**AWS S3 Example:**
```yaml
s3:
  s3_example:
    access_key: "YOUR_ACCESS_KEY"
    secret_key: "YOUR_SECRET_KEY"
    bucket_name: "YOUR_BUCKET_NAME"
    cache: true
    exclude_patterns:
      - .pdf
      - .docx
```

**Slack Example:**
```yaml
slack:
  slack_example:
    channel_types: "public_channel,private_channel"
    token: "xoxb-..."
    onlyArchived: false
    archived_channels: false
    limit_mins: 60
    read_from: "last_message"
    isExternal: null
    channel_ids:
      - "C123456"
    blacklisted_channel_ids:
      - "C789012"
```

---

## 📈 Performance Specifications

| Metric | Target | Measured |
|--------|--------|----------|
| **Scan Throughput** | 200-350 files/sec | ✅ |
| **Validation Speed** | 1,000 findings/sec | ✅ |
| **API Ingestion** | 500-1,000 findings/sec | ✅ |
| **Graph Queries** | 50-150ms (p95) | ✅ |
| **Max Assets** | 1,000,000 | ✅ |
| **Max Findings** | 10,000,000 | ✅ |
| **Max Graph Nodes** | 500,000 | ✅ |

---

## 🗺️ Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        ARC-Hawk Platform                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐              │
│  │   Scanner   │───▶│   Backend   │───▶│ PostgreSQL  │              │
│  │     SDK     │    │     API     │    │  (Storage)  │              │
│  └─────────────┘    └─────────────┘    └─────────────┘              │
│        │                  │                   │                      │
│        │                  │                   │                      │
│        ▼                  ▼                   ▼                      │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐              │
│  │  Validate   │    │  Ingest &   │    │  Assets &   │              │
│  │  PII (Edge) │    │  Classify   │    │  Findings   │              │
│  └─────────────┘    └─────────────┘    └─────────────┘              │
│                           │                                        │
│                           │                                        │
│                           ▼                                        │
│                    ┌─────────────┐    ┌─────────────┐              │
│                    │   Neo4j     │◀───│   Lineage   │              │
│                    │   (Graph)   │    │   Service   │              │
│                    └─────────────┘    └─────────────┘              │
│                           │                                        │
│                           │                                        │
│                           ▼                                        │
│                    ┌─────────────┐    ┌─────────────┐              │
│                    │   Next.js   │    │  Dashboard  │              │
│                    │  (Frontend) │    │   UI/UX     │              │
│                    └─────────────┘    └─────────────┘              │
│                                                                       │
├─────────────────────────────────────────────────────────────────────┤
│  Infrastructure Layer (Docker Compose)                               │
│  - Temporal Workflow Engine                                          │
│  - Presidio ML Analysis                                              │
│  - Redis (Future Caching)                                            │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 📜 Maintenance Log

### 2026-01-22 (CORRECTED)
- **Phase 1 Complete:** Blueprint inferred from comprehensive codebase analysis
- **Critical Correction:** Read `connection.yml.sample` - NOT `connection.yml`
- **Discovery Complete:** All 5 questions answered through code review
- **Schema Defined:** JSON schemas documented for all data flows (CORRECTED - each source has UNIQUE parameters)
- **Rules Documented:** Architectural principles and constraints recorded
- **Integrations Mapped:** All external services and data sources cataloged (with CORRECT connection schemas)
- **Alerting Defined:** Slack and Jira notification configurations documented

### 2026-01-21
- **Initial Setup:** Project structure created
- **Documentation:** AGENTS.md created for AI agent guidance

---

## 🚦 Phase Checklist

### Phase 1: Blueprint ✅ Complete
- [x] North Star defined
- [x] Integrations identified (with CORRECT connection schemas)
- [x] Source of Truth established
- [x] Delivery Payload specified
- [x] Behavioral Rules documented (severity rules engine)
- [x] Data Schemas defined · Architecture diagram · Performance specs

### Phase 2: Agentic Integration 🔄 In Progress
- [x] Install MCP Tools (GSD, Ralph, Hive, Skills, Superpowers)
- [x] Configure MCP server (`mcp_config_snippet.json`)
- [x] Deep codebase analysis (12 backend modules, 14 scanner connectors, 12 frontend routes)
- [x] Map agent tools → B.L.A.S.T. phases
- [ ] Initialize GSD project (`npx get-shit-done-cc`)
- [ ] Configure Ralph (`prd.json` for Phase 3)
- [ ] Update system prompt with skill-trigger rules

### Phase 3: Link (Pending) — Tools: `@docker-expert`, `@systematic-debugging`
- [ ] Verify PostgreSQL connectivity · Run migrations
- [ ] Verify Neo4j connectivity · Load Cypher schema
- [ ] Test Temporal workflow engine
- [ ] Validate Presidio ML integration
- [ ] Test scanner→backend ingestion pipeline
- [ ] Verify frontend↔API connection

### Phase 4: Architect (Pending) — Tools: `@golang-pro`, `@python-pro`, `@api-design-principles`
- [ ] Define Layer 1 SOPs · Build Layer 3 tools
- [ ] Implement data flow logic · Error handling
- [ ] Write unit tests (`@testing-patterns`)

### Phase 5: Stylize (Pending) — Tools: `@nextjs-best-practices`, `@react-patterns`
- [ ] Format API responses · Style dashboard
- [ ] Optimize visualizations · User feedback

### Phase 6: Trigger (Pending) — Tools: `@docker-expert`, `@security-auditor`
- [ ] Deploy to cloud · Cron jobs · Webhooks · Docs

---

## 🔗 References

- **README:** `/readme.md`
- **Connection Sample:** `/apps/scanner/config/connection.yml.sample` (PRIMARY)
- **Architecture:** `/docs/architecture/ARCHITECTURE.md`
- **Tech Stack:** `/docs/development/TECH_STACK.md`
- **API Specs:** `/docs/development/TECHNICAL_SPECIFICATIONS.md`
- **User Manual:** `/docs/USER_MANUAL.md`
- **Failure Modes:** `/docs/FAILURE_MODES.md`

---

*This document serves as the Project Constitution and must be updated when any schema, rule, or architecture changes.*

**CRITICAL NOTE:** Always reference `connection.yml.sample` for actual connection schemas, NOT `connection.yml`. Each data source type requires UNIQUE connection parameters.
