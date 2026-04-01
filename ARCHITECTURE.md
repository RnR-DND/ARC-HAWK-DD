# ARC-Hawk Architecture

## System Overview

ARC-Hawk is an enterprise PII discovery and remediation platform. It scans data sources for sensitive information (PAN, Aadhaar, credit cards, email, phone, etc.), classifies findings, and provides compliance dashboards and masking/remediation workflows.

## High-Level Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Browser                             │
│              Next.js Frontend  (port 3000)                      │
└────────────────────────┬────────────────────────────────────────┘
                         │ HTTP / WebSocket
┌────────────────────────▼────────────────────────────────────────┐
│              Go Backend — Modular Monolith (port 8080)          │
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────────┐ │
│  │  Auth    │ │ Scanning │ │  Assets  │ │    Compliance      │ │
│  │ Module   │ │  Module  │ │  Module  │ │     Module         │ │
│  └──────────┘ └──────────┘ └──────────┘ └────────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────────┐ │
│  │ Lineage  │ │ Masking  │ │Analytics │ │   Connections      │ │
│  │  Module  │ │  Module  │ │  Module  │ │     Module         │ │
│  └──────────┘ └──────────┘ └──────────┘ └────────────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                        │
│  │Remediat. │ │FPLearning│ │WebSocket │                        │
│  │  Module  │ │  Module  │ │  Module  │                        │
│  └──────────┘ └──────────┘ └──────────┘                        │
└───┬──────────────┬─────────────────────────────────────────────┘
    │              │
    ▼              ▼
┌───────┐    ┌─────────┐
│Postgres│   │  Neo4j  │
│  (SQL) │   │ (Graph) │
└───────┘    └─────────┘
                         ┌───────────────────────────────────────┐
                         │    Python Scanner Service (port 5002) │
                         │  hawk_scanner CLI + SDK recognizers   │
                         │  (PAN, Aadhaar, CC, Email, Phone...)  │
                         └──────────────┬────────────────────────┘
                                        │ POST /api/v1/scans/ingest-verified
                                        ▼
                                   Go Backend

Optional (not enabled by default):
  ┌──────────┐   ┌────────────────────────────┐
  │ Temporal │   │  Presidio Analyzer (ML PII) │
  │Workflow  │   │       (port 3001)           │
  │  Engine  │   └────────────────────────────┘
  └──────────┘
```

## Module Dependency Graph

```
Assets ──────────────────────────────────────────┐
   └── FindingsProvider ──► Lineage               │
                              └── LineageSync ──►─┤
Assets ──► AssetManager ────────────────────────►─┤
                                                  ▼
                        Scanning, Auth, Compliance, Masking,
                        Analytics, Connections, Remediation,
                        FPLearning, WebSocket
```

Initialization order enforced in `cmd/server/main.go`:
1. Assets
2. Lineage (requires FindingsProvider from Assets)
3. All remaining modules (require AssetManager + LineageSync)

## Data Flow

```
User connects data source
        │
        ▼
Connections Module → stores encrypted credentials (AES-GCM) in Postgres
        │
        ▼
Scan triggered → Scanner Service executes hawk_scanner CLI
        │
        ▼
Scanner POSTs VerifiedScanInput → Backend /api/v1/scans/ingest-verified
        │
        ▼
Scanning Module → classify findings (rules + context + entropy weights)
        │
        ├── Stores findings in Postgres (findings table)
        ├── Updates asset risk scores
        ├── Syncs data lineage to Neo4j graph
        └── Broadcasts via WebSocket to connected frontends
```

## External Service Dependencies

| Service | Purpose | Port | Required |
|---------|---------|------|----------|
| PostgreSQL 15 | Primary database (findings, assets, users, tenants) | 5432 | Yes |
| Neo4j 5.15 | Data lineage graph storage | 7687 | Yes |
| Python Scanner | PII scanning via hawk_scanner CLI | 5002 | Yes |
| Presidio Analyzer | ML-based PII detection (Microsoft) | 3001 | Optional |
| Temporal | Workflow orchestration for long-running scans | 7233 | Optional |
| Prometheus | Metrics collection | 9090 | Optional |
| Grafana | Metrics dashboards | 3002 | Optional |

## Environment / Config Requirements

See `apps/backend/.env.example` for the full list. Required variables:

- `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME`
- `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
- `NEO4J_URI`, `NEO4J_USERNAME`, `NEO4J_PASSWORD`
- `JWT_SECRET` (min 32 chars, random)
- `ENCRYPTION_KEY` (exactly 32 chars)
- `AUTH_REQUIRED` (default: `true`)

## Entry Points

| Component | How to Run |
|-----------|-----------|
| Full stack | `docker-compose up` from project root |
| Backend only | `cd apps/backend && go run ./cmd/server/main.go` |
| Scanner only | `cd apps/scanner && python scanner_api.py` |
| Frontend only | `cd apps/frontend && npm run dev` |
| DB migrations | `cd apps/backend && migrate -path migrations_versioned -database $DSN up` |

## Tech Stack

- **Backend**: Go 1.24, Gin, golang-migrate, Neo4j Go Driver, Temporal SDK
- **Frontend**: Next.js (App Router), TypeScript, Tailwind CSS, Framer Motion
- **Scanner**: Python 3, Flask, hawk_scanner CLI, Presidio SDK recognizers
- **Databases**: PostgreSQL 15, Neo4j 5.15 Community
- **Infrastructure**: Docker Compose, Prometheus, Grafana, Kubernetes (infra/k8s/)
