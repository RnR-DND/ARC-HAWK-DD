# Documentation Index

**ARC-HAWK Platform v3.0.0 — Complete Documentation Guide**

---

## Getting Started

| Document | Description |
|----------|-------------|
| [README.md](../readme.md) | Platform overview, quick start, feature summary |
| [docs/RUNBOOK_E2E.md](./RUNBOOK_E2E.md) | **Start here for ops** — full stack startup, migrations, e2e scan walkthrough |
| [docs/USER_MANUAL.md](./USER_MANUAL.md) | End-user guide for the dashboard |
| [AGENTS.md](../AGENTS.md) | AI agent development guide and build commands |

---

## Architecture & Design

| Document | Description |
|----------|-------------|
| [docs/architecture/ARCHITECTURE.md](./architecture/ARCHITECTURE.md) | System architecture, components, data flow, security model |
| [docs/architecture/overview.md](./architecture/overview.md) | Concise architecture overview with lineage hierarchy |
| [docs/architecture/WORKFLOW.md](./architecture/WORKFLOW.md) | Operational workflows — scan execution, ingestion, compliance |
| [docs/architecture/INTEGRATION.md](./architecture/INTEGRATION.md) | Architecture-level integration contracts and API reference |
| [docs/development/TECH_STACK.md](./development/TECH_STACK.md) | Technology choices — Go, Next.js, PostgreSQL, Neo4j, Temporal, swaggo |

---

## Integration & APIs

| Document | Description |
|----------|-------------|
| [docs/INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md) | **End-to-end integration walkthrough** for embedding ARC-HAWK as a microservice |
| [docs/WEBHOOKS.md](./WEBHOOKS.md) | Webhook event catalog, HMAC signing, retry policy, receiver example |
| [docs/API.md](./API.md) | Full REST API endpoint reference |
| [docs/openapi/openapi.yaml](./openapi/openapi.yaml) | OpenAPI 3.0 spec (coming soon — generated via swaggo) |

---

## Scanner

| Document | Description |
|----------|-------------|
| [docs/SCANNER_REFERENCE.md](./SCANNER_REFERENCE.md) | Go scanner disambiguation, auth, env vars, connector list, streaming protocol |

---

## Operations & Deployment

| Document | Description |
|----------|-------------|
| [docs/RUNBOOK_E2E.md](./RUNBOOK_E2E.md) | Unified end-to-end runbook — supersedes SEAMLESS_SCANNING.md and phase1_deployment.md |
| [DEPLOYMENT_RUNBOOK.md](../DEPLOYMENT_RUNBOOK.md) | Kubernetes and production deployment procedures |
| [docs/deployment/guide.md](./deployment/guide.md) | Detailed deployment guide |
| [docs/MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) | Upgrade procedures (v2.x → v3.0.0) |
| [docs/FAILURE_MODES.md](./FAILURE_MODES.md) | Troubleshooting guide for common failure modes |

---

## Release Notes

| Document | Description |
|----------|-------------|
| [docs/releases/v3.0.0.md](./releases/v3.0.0.md) | v3.0.0 release notes — breaking changes, new features, migration notes |
| [CHANGELOG.md](../CHANGELOG.md) | Full version history |

---

## Implementation & Algorithms

| Document | Description |
|----------|-------------|
| [docs/development/TECHNICAL_SPECIFICATIONS.md](./development/TECHNICAL_SPECIFICATIONS.md) | Database schemas, API specs, performance benchmarks, system requirements |
| [docs/deployment/MATHEMATICAL_IMPLEMENTATION.md](./deployment/MATHEMATICAL_IMPLEMENTATION.md) | Verhoeff, Luhn, Modulo-26 validation algorithms; risk scoring |
| [docs/deployment/LIMITATIONS_AND_IMPROVEMENTS.md](./deployment/LIMITATIONS_AND_IMPROVEMENTS.md) | Known limitations, technical debt, future roadmap |

---

## Security & Compliance

| Document | Description |
|----------|-------------|
| [SECURITY.md](../SECURITY.md) | Security policy, vulnerability reporting, best practices |
| [docs/dpdpa-mapping.md](./dpdpa-mapping.md) | DPDPA 2023 obligation mapping |
| [RISK.md](../RISK.md) | Risk assessments, blast radius, deferred work |

---

## Development

| Document | Description |
|----------|-------------|
| [docs/development/setup.md](./development/setup.md) | Development environment setup |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Contribution guidelines and PR process |
| [TODO.md](../TODO.md) | Known issues, open P0–P2 items, production readiness tracking |

---

## Testing

| Document | Description |
|----------|-------------|
| [tests/e2e/README.md](../tests/e2e/README.md) | E2E test harness — prerequisites, full-scan.sh (coming soon), manual curl walkthrough |

---

## Architecture SOPs

| Document | Description |
|----------|-------------|
| [docs/architecture/sops/scanning-sop.md](./architecture/sops/scanning-sop.md) | Scanning standard operating procedure |
| [docs/architecture/sops/ingestion-sop.md](./architecture/sops/ingestion-sop.md) | Ingestion standard operating procedure |
| [docs/architecture/sops/lineage-sop.md](./architecture/sops/lineage-sop.md) | Lineage standard operating procedure |
| [docs/architecture/sops/compliance-sop.md](./architecture/sops/compliance-sop.md) | Compliance standard operating procedure |

---

## Operational Runbooks

| Document | Description |
|----------|-------------|
| [docs/runbooks/runbook-archive-bomb.md](./runbooks/runbook-archive-bomb.md) | Handling archive bomb scan failures |
| [docs/runbooks/runbook-corrupt-file.md](./runbooks/runbook-corrupt-file.md) | Handling corrupt file scan failures |
| [docs/runbooks/runbook-encrypted-pdf.md](./runbooks/runbook-encrypted-pdf.md) | Handling encrypted PDF files |
| [docs/runbooks/runbook-expired-credentials.md](./runbooks/runbook-expired-credentials.md) | Rotating expired connection credentials |
| [docs/runbooks/runbook-kafka-lag.md](./runbooks/runbook-kafka-lag.md) | Resolving Kafka consumer lag |
| [docs/runbooks/runbook-low-ocr.md](./runbooks/runbook-low-ocr.md) | Low OCR confidence handling |
| [docs/runbooks/runbook-schema-change.md](./runbooks/runbook-schema-change.md) | Schema change impact on scanning |

---

## Integrations

| Document | Description |
|----------|-------------|
| [docs/integrations/agent-toolchain.md](./integrations/agent-toolchain.md) | Agent toolchain integration |
| [docs/integrations/supermemory.md](./integrations/supermemory.md) | Supermemory.ai memory layer integration |
| [docs/agent-install.md](./agent-install.md) | Hawk agent installation (Linux, macOS, Windows) |
| [docs/custom-regex.md](./custom-regex.md) | Custom PII pattern definition and management |

---

## Deprecated / Historical

| Document | Description |
|----------|-------------|
| [docs/SEAMLESS_SCANNING.md](./SEAMLESS_SCANNING.md) | **Deprecated** — superseded by RUNBOOK_E2E.md |
| [docs/phase1_deployment.md](./phase1_deployment.md) | **Historical** — Phase 1 deployment; superseded by RUNBOOK_E2E.md |

---

## Learning Paths

### New User (30 min)
1. [README.md](../readme.md) — overview
2. [docs/RUNBOOK_E2E.md](./RUNBOOK_E2E.md) — start the system
3. [docs/USER_MANUAL.md](./USER_MANUAL.md) — use the dashboard

### Developer Onboarding (4 hrs)
1. [README.md](../readme.md)
2. [AGENTS.md](../AGENTS.md)
3. [docs/architecture/ARCHITECTURE.md](./architecture/ARCHITECTURE.md)
4. [docs/development/TECH_STACK.md](./development/TECH_STACK.md)
5. [docs/SCANNER_REFERENCE.md](./SCANNER_REFERENCE.md)
6. [docs/INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md)
7. [docs/development/TECHNICAL_SPECIFICATIONS.md](./development/TECHNICAL_SPECIFICATIONS.md)

### Platform Integrator (2 hrs)
1. [docs/INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md)
2. [docs/WEBHOOKS.md](./WEBHOOKS.md)
3. [docs/API.md](./API.md)
4. [docs/RUNBOOK_E2E.md](./RUNBOOK_E2E.md)

### DevOps / SRE (2 hrs)
1. [docs/RUNBOOK_E2E.md](./RUNBOOK_E2E.md)
2. [DEPLOYMENT_RUNBOOK.md](../DEPLOYMENT_RUNBOOK.md)
3. [docs/FAILURE_MODES.md](./FAILURE_MODES.md)
4. [SECURITY.md](../SECURITY.md)
5. [docs/MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md)

---

**Last Updated**: 2026-04-22
**Platform Version**: v3.0.0
