# ARC-Hawk Platform

<div align="center">

[![Production Status](https://img.shields.io/badge/status-development-yellow)](./TODO.md)
[![Version](https://img.shields.io/badge/version-3.0.0-blue)](./CHANGELOG.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-lightgrey)](./LICENSE)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen)](apps/backend)
[![Node.js](https://img.shields.io/badge/node-18+-green)](apps/frontend)
[![Go Scanner](https://img.shields.io/badge/go--scanner-1.24+-00ADD8)](apps/goScanner)

**Enterprise-grade PII Discovery, Classification, and Lineage Tracking Platform**

[Quick Start](#-quick-start) • [Documentation](#-documentation) • [Features](#-key-features) • [Architecture](#-architecture) • [Support](#-support)

</div>

---

> **Status**: ARC-Hawk v3.0.0 is in active development. See [TODO.md](./TODO.md) for known open items and [AGENTS.md](./AGENTS.md) for AI agent development guidance.

---

## 🎯 What is ARC-Hawk?

ARC-Hawk is a **comprehensive platform** that automatically discovers, validates, and tracks Personally Identifiable Information (PII) across your entire data infrastructure. Built with an **Intelligence-at-Edge** architecture, it provides:

- ✅ **Context-Aware Detection** - Smart engine distinguishes real code (`key="value"`) from test data/comments
- ✅ **Entropy Validation** - Rejects low-entropy secrets ("password123") vs high-entropy keys
- ✅ **Mathematical Validation** - 11 India-specific PII types with 100% accuracy (Verhoeff, Luhn algorithms)
- ✅ **Multi-Source Scanning** - Filesystem, PostgreSQL, MySQL, MongoDB, S3, GCS, Redis, Slack, and more
- ✅ **Semantic Lineage** - Visual graph showing where PII flows across your systems
- ✅ **Compliance Ready** - DPDPA 2023 (India) mapping with consent and retention tracking
- ✅ **Real-Time Monitoring** - Live scan progress and system health tracking
- ✅ **Automated Remediation** - One-click masking and deletion of sensitive data

---

## 🚀 Quick Start

### Prerequisites

- **Docker** & **Docker Compose** (v2.0+)
- **4GB+ RAM** recommended
- **Go 1.24+** (for backend development)
- **Node.js 18+** (for frontend development)

### Installation

```bash
# 1. Clone repository
git clone https://github.com/your-org/arc-hawk.git
cd arc-hawk

# 2. Start infrastructure services
docker-compose up -d postgres neo4j temporal

# 3. Start backend server
cd apps/backend
go mod download
go run cmd/server/main.go

# 4. Start frontend (in new terminal)
cd apps/frontend
npm install
npm run dev

# 5. Access the Dashboard
# Open http://localhost:3000 in your browser
```

**Services Available:**
- **Frontend Dashboard**: `http://localhost:3000`
- **Backend API**: `http://localhost:8080`
- **API Documentation**: `http://localhost:8080/swagger/index.html`
- **Temporal UI**: `http://localhost:8088` (Workflows)
- **Neo4j Browser**: `http://localhost:7474` (Graph DB)

### Install the Hawk Agent

The agent runs on the machine where your data lives and streams scan results back to the platform.

```bash
# macOS
curl -Lo hawk-agent-mac https://releases.arc-hawk.io/latest/hawk-agent-mac
chmod +x hawk-agent-mac
sudo HAWK_SERVER_URL=http://localhost:8080 HAWK_AGENT_ID=my-agent HAWK_AGENT_CLIENT_SECRET=secret ./hawk-agent-mac
```

See [docs/agent-install.md](./docs/agent-install.md) for Linux (systemd), Windows (Task Scheduler), config file options, and offline buffer details.

### Add a Custom PII Pattern

Define your own PII types beyond the built-in 11 in under a minute.

```bash
curl -X POST http://localhost:8080/api/v1/patterns \
  -H "Content-Type: application/json" \
  -d '{"name": "employee_id", "display_name": "Employee ID", "regex": "EMP-[0-9]{6}", "category": "HR"}'
```

The pattern is validated for ReDoS safety on write and picked up by the scanner within 60 seconds — no restart needed.

See [docs/custom-regex.md](./docs/custom-regex.md) for validation rules, test runner usage, sensitivity levels, auto-deactivation, and bulk import/export.

---

## 🏗️ Architecture

ARC-Hawk uses a modern, distributed architecture with **Intelligence-at-Edge** principles:

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Frontend      │────▶│   Backend API    │────▶│   PostgreSQL    │
│  (Next.js 14)   │◄────│    (Go/Gin)      │◄────│   (Relational)  │
└─────────────────┘     └──────────────────┘     └─────────────────┘
         │                       │                          │
         │              ┌────────┴────────┐                 │
         │              │                 │                 │
         │         ┌────▼────┐      ┌────▼────┐            │
         └────────▶│ Neo4j   │      │Temporal │            │
                   │ (Graph) │      │(Workflow│            │
                   └─────────┘      │ Engine) │            │
                                    └────┬────┘            │
                                         │                 │
                                    ┌────▼────┐            │
                                    │ Scanner │            │
                                    │ (Go/Gin)│            │
                                    └────┬────┘            │
                                         │                 │
                    ┌────────────────────┼─────────────────┘
                    │                    │
              ┌─────▼──────┐  ┌──────────▼──────────┐
              │Filesystem  │  │   Data Sources      │
              │   S3/GCS   │  │ (PostgreSQL, MySQL, │
              │   etc.     │  │  MongoDB, Redis...) │
              └────────────┘  └─────────────────────┘
```

### Core Components

1. **Frontend (Next.js 14)**: Real-time dashboard with ReactFlow visualization for lineage, compliance tracking, and remediation actions
2. **Backend (Go/Gin)**: Modular monolith with 10 business modules (Assets, Scanning, Lineage, Compliance, Discovery, Remediation, Masking, Analytics, Connections, Auth)
3. **Orchestrator (Temporal)**: Manages long-running workflows with reliable retries and state management
4. **Go Scanner (Go/Gin)**: High-performance PII detection engine at `apps/goScanner/` (port 8001) — canonical scanner; Python scanner retired
5. **Storage**:
   - **PostgreSQL**: Relational data (Assets, Findings, Configs, Compliance)
   - **Neo4j**: Graph data (Lineage, Relationships, Data Flow)

---

## ✨ Key Features

### 🔍 Context-Aware Intelligence Engine
**No more false positives.** The scanner uses a multi-stage validation pipeline:
1. **Context Analysis**: Understands variable assignments vs comments vs configuration
2. **Entropy Check**: Validates the randomness of secrets using Shannon entropy
3. **Mathematical Validation**: Verifies checksums using industry-standard algorithms
4. **PII Classification**: 11 locked India-specific PII types with zero false positives

**Supported PII Types**:
- Aadhaar (Verhoeff algorithm)
- PAN (Weighted Modulo 26)
- Passport
- Credit Cards (Luhn algorithm)
- Bank Account Numbers
- IFSC Codes
- UPI IDs
- Driving License
- Voter ID
- Phone Numbers
- Email Addresses

### 🌐 Multi-Source Scanning
Unified interface for scanning across diverse data sources:

| Source | Status | Features |
|--------|--------|----------|
| **Filesystem** | ✅ Production | Local, network mounts, archives |
| **AWS S3** | ✅ Ready | Buckets, prefixes, multipart |
| **Google Cloud Storage** | ✅ Ready | Buckets, objects |
| **Azure Blob Storage** | ✅ Ready | Containers, blobs |
| **PostgreSQL** | ✅ Production | Tables, columns, row sampling |
| **MySQL** | ✅ Ready | Tables, columns |
| **MongoDB** | ✅ Ready | Collections, documents |
| **Redis** | ✅ Ready | Keys, values |
| **MSSQL** | ✅ Ready | Tables, columns |
| **SQLite** | ✅ Ready | Tables, columns (local dev) |
| **BigQuery** | ✅ Ready | Datasets, tables, partitions |
| **Redshift** | ✅ Ready | Schemas, tables |
| **Snowflake** | ✅ Ready | Databases, schemas, tables |
| **AWS Kinesis** | ✅ Ready | Stream snapshot sampling |
| **Kafka** | ✅ Ready | Topic snapshot sampling |
| **Slack** | ✅ Ready | Message history |
| **Salesforce** | ✅ Ready | Objects, fields |
| **HubSpot** | ✅ Ready | Contacts, companies, deals |
| **Jira** | ✅ Ready | Issues, comments |
| **MS Teams** | ✅ Ready | Message history |
| **Avro** | ✅ Ready | Schema + record scanning |
| **Parquet** | ✅ Ready | Column-level scanning |
| **PPTX** | ✅ Ready | Slide text extraction |
| **HTML** | ✅ Ready | Text and attribute scanning |
| **Email (.eml/.msg)** | ✅ Ready | Headers, body, attachments |
| **Firebase** | 🔄 Beta | Realtime DB, Firestore |
| **Google Drive** | 🔄 Beta | Documents, sheets |

### 📊 Semantic Lineage Tracking
Interactive graph visualization showing exactly where PII flows:

```
System (e.g., "Production DB")
  ↓ SYSTEM_OWNS_ASSET
Asset (e.g., "users_table")
  ↓ EXPOSES
PII Type (e.g., "Aadhaar", "PAN")
```

- **3-Level Hierarchy**: Simplified semantic model for clarity
- **ReactFlow Visualization**: Interactive, zoomable, exportable graphs
- **Impact Analysis**: See what systems are affected by data changes
- **Compliance Mapping**: Track DPDPA 2023 requirements across lineage

### 🛡️ Automated Remediation
- **Masking**: Redact sensitive data at the source with configurable policies
- **Deletion**: Securely remove non-compliant data with audit trails
- **Audit Trail**: Full history of all remediation actions
- **One-Click Actions**: Execute remediation directly from findings

### 🔎 Enterprise Data Discovery
Inventory and risk-score all data assets across your infrastructure:

- **Asset Risk Scoring**: Time-series risk scores with spike detection and trend analysis
- **Discovery Heatmap**: Visual breakdown of PII exposure by asset type and data category
- **Semantic Lineage**: Track exactly how sensitive fields flow between systems
- **Upstream Inventory**: Per-source finding aggregates with configurable thresholds

### 🧩 Custom PII Patterns
Define your own PII types beyond the built-in 11:

- **CRUD API**: Create, list, and delete patterns at `/api/v1/patterns`
- **ReDoS Protection**: Patterns with nested quantifiers or >512 chars are rejected at submission
- **Per-Scan Hot-Reload**: New patterns apply to the next scan — no service restart needed
- **Inline Validation**: Pattern errors surface in the scan config UI before a scan starts

### ⚖️ Compliance & Governance
Built-in support for **DPDPA 2023** (India's Digital Personal Data Protection Act):

- **Obligation Mapping**: Automatically maps PII findings to DPDPA provisions
- **Gap Reports**: Export compliance gap reports in PDF or JSON
- **Consent Tracking**: Track consent status for each PII instance
- **Retention Policies**: Automated retention period monitoring with violation alerts
- **Audit Log Chain**: Hash-linked audit entries for tamper-evident logging
- **Compliance Reports**: Generate audit-ready reports
- **Data Principal Rights**: Support for access, correction, and deletion requests

---

## 📚 Documentation

ARC-Hawk includes **150+ pages** of comprehensive documentation:

### Getting Started
- [**README.md**](./readme.md) (this file) - Platform overview and quick start
- [**Documentation Index**](./docs/INDEX.md) - Complete navigation guide
- [**User Manual**](./docs/USER_MANUAL.md) - End-user guide for the dashboard

### Architecture & Design
- [**Architecture**](./docs/architecture/ARCHITECTURE.md) (~25 pages) - System design and components
- [**Workflow**](./docs/architecture/WORKFLOW.md) (~30 pages) - Operational workflows
- [**Tech Stack**](./docs/development/TECH_STACK.md) (~18 pages) - Technology choices

### Implementation
- [**Technical Specifications**](./docs/development/TECHNICAL_SPECIFICATIONS.md) (~35 pages) - API specs, schemas
- [**Mathematical Implementation**](./docs/deployment/MATHEMATICAL_IMPLEMENTATION.md) (~20 pages) - Validation algorithms

### Operations
- [**Deployment Guide**](./docs/deployment/guide.md) - Deployment procedures
- [**Migration Guide**](./docs/MIGRATION_GUIDE.md) - Upgrade procedures (v2.1 → v3.0)
- [**Failure Modes**](./docs/FAILURE_MODES.md) - Troubleshooting guide
- [**CHANGELOG.md**](./CHANGELOG.md) - Version history and release notes
- [**RISK.md**](./RISK.md) - Risk assessments, blast radius, and deferred work
- [**TODO.md**](./TODO.md) - Known issues and roadmap

### Integration
- [**Integration Guide**](./docs/INTEGRATION_GUIDE.md) - End-to-end microservice integration walkthrough
- [**Webhooks**](./docs/WEBHOOKS.md) - Webhook event stream reference
- [**OpenAPI Spec**](./docs/openapi/openapi.yaml) - OpenAPI 3.0 spec (coming soon)

### Development
- [**AGENTS.md**](./AGENTS.md) - AI Agent development guide
- [**Backend README**](./apps/backend/README.md) - Go backend documentation
- [**Frontend README**](./apps/frontend/README.md) - Next.js frontend documentation
- [**Scanner Reference**](./docs/SCANNER_REFERENCE.md) - Go scanner at apps/goScanner/

---

## 🛠️ Development

### Backend (Go)

```bash
cd apps/backend

# Install dependencies
go mod tidy

# Run server
go run cmd/server/main.go

# Run tests
go test ./... -v

# Build binary
go build -o arc-backend cmd/server/main.go
```

**Key Directories:**
- `modules/` - 10 business modules (assets, scanning, lineage, compliance, discovery, remediation, etc.)
- `pkg/` - Shared packages (validation, normalization)
- `cmd/server/` - Application entry point

### Frontend (Next.js)

```bash
cd apps/frontend

# Install dependencies
npm install

# Run development server
npm run dev

# Build for production
npm run build

# Run linter
npm run lint
```

**Key Directories:**
- `app/` - Next.js pages (App Router)
- `components/` - Reusable React components
- `services/` - Typed API clients
- `types/` - TypeScript definitions

### Go Scanner

The Go scanner runs as an internal sidecar (port `:8001`) and is managed automatically by Docker Compose. For local development:

```bash
cd apps/goScanner

# Run tests
go test ./...

# Build binary
go build -o go-scanner cmd/scanner/main.go

# Run locally
SCANNER_SERVICE_TOKEN=<token> \
BACKEND_URL=http://localhost:8080 \
PRESIDIO_URL=http://localhost:3000 \
./go-scanner
```

**Key Directories:**
- `internal/connectors/` - 36+ source connectors (databases, cloud, SaaS, files)
- `internal/orchestrator/` - Scan orchestration and ingest logic
- `api/` - HTTP handler and auth middleware

See [docs/SCANNER_REFERENCE.md](./docs/SCANNER_REFERENCE.md) for full details.

---

## 🧪 Testing

### Run All Tests
```bash
./scripts/testing/run-tests.sh
```

### Backend Tests
```bash
cd apps/backend
go test ./... -v
go test ./modules/scanning -v  # Specific module
```

### Frontend Tests
```bash
cd apps/frontend
npm test -- --passWithNoTests
```

### Go Scanner Tests
```bash
cd apps/goScanner
go test ./...
```

### Integration Tests
```bash
# End-to-end manual walkthrough
# See tests/e2e/README.md for the curl-based walkthrough

# API endpoint verification
python scripts/verify_endpoints.py
```

---

## 📊 Performance

| Metric | Value | Notes |
|--------|-------|-------|
| **Scanner Throughput** | 200-350 files/sec | Depends on file size and complexity |
| **API Response Time** | <100ms (p95) | For standard CRUD operations |
| **Lineage Query** | <500ms | For graphs with 10k+ nodes |
| **Database Connections** | 25 (PostgreSQL) | Configurable pool size |
| **Supported File Size** | Up to 100MB | Larger files processed in chunks |

---

## 🔒 Security

See [SECURITY.md](./SECURITY.md) for detailed security information.

**Key Security Features:**
- Mathematical validation prevents false positives
- No data leaves your infrastructure
- Environment-based configuration (no hardcoded secrets)
- Audit logging for all actions
- Input validation and sanitization

> **Note**: Authentication/Authorization is not yet implemented (see [TODO.md](./TODO.md)).

---

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

**Ways to Contribute:**
- 🐛 Report bugs via GitHub Issues
- 💡 Suggest features via GitHub Discussions
- 📝 Improve documentation
- 🔧 Submit pull requests

---

## 📄 License

This project is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.

---

## 🆘 Support

- **Documentation**: [docs/INDEX.md](./docs/INDEX.md)
- **Issues**: [GitHub Issues](https://github.com/your-org/arc-hawk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/arc-hawk/discussions)
- **Email**: support@arc-hawk.io

---

## 🗺️ Roadmap

See [TODO.md](./TODO.md) and [Limitations & Improvements](./docs/deployment/LIMITATIONS_AND_IMPROVEMENTS.md) for detailed roadmap.

**High Priority:**
- Authentication & Authorization (JWT/RBAC)
- Complete remediation module
- Comprehensive test suite

**Medium Priority:**
- Multi-source connector testing
- Performance optimizations
- Additional compliance frameworks (GDPR, CCPA)

**Future:**
- ML-based pattern learning
- Real-time streaming detection
- Advanced analytics and reporting

---

<div align="center">

**[⬆ Back to Top](#arc-hawk-platform)**

Built with ❤️ by the ARC-Hawk Team

</div>
