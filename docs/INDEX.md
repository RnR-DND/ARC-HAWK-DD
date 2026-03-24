# Documentation Index

**ARC-Hawk Platform - Complete Documentation Guide**

This index provides a comprehensive overview of all available documentation and guides you to the right resources based on your needs.

---

## 📖 Documentation Overview

The ARC-Hawk platform has **200+ pages** of comprehensive technical documentation organized into the following categories:

- **Getting Started**: Quick start guides and overviews
- **Core Documentation**: Architecture, implementation, workflows, specifications
- **API Reference**: Complete API documentation
- **User Guides**: Setup, usage, troubleshooting
- **Developer Resources**: Development guides, contribution guidelines
- **Operations**: Deployment, monitoring, maintenance
- **Security**: Security practices and compliance

---

## 🚀 Getting Started (Start Here!)

### 5-Minute Quick Start
1. Read [README.md](../readme.md) - Platform overview
2. Follow [Installation](#installation-guides) - Set up the system
3. Run your [First Scan](#first-scan-guide) - Start scanning

### Learning Paths

#### **New User (30 minutes)**
1. [README.md](../readme.md) - Overview (5 min)
2. [Workflow - System Setup](architecture/WORKFLOW.md#system-setup-workflow) - Installation (15 min)
3. [User Manual](USER_MANUAL.md) - Dashboard usage (10 min)

#### **Developer Onboarding (4 hours)**
1. [README.md](../readme.md) - Overview (10 min)
2. [AGENTS.md](../AGENTS.md) - Development guide (15 min)
3. [Architecture](architecture/ARCHITECTURE.md) - System design (60 min)
4. [Tech Stack](development/TECH_STACK.md) - Technologies (30 min)
5. [Mathematical Implementation](deployment/MATHEMATICAL_IMPLEMENTATION.md) - Algorithms (60 min)
6. [Technical Specifications](development/TECHNICAL_SPECIFICATIONS.md) - Schemas and APIs (60 min)
7. [API Reference](API.md) - API usage (30 min)

#### **DevOps/Operations (2 hours)**
1. [Technical Specifications - Requirements](development/TECHNICAL_SPECIFICATIONS.md#system-requirements) (15 min)
2. [Deployment Guide](deployment/guide.md) - Installation (45 min)
3. [Failure Modes](FAILURE_MODES.md) - Troubleshooting (30 min)
4. [Security Policy](../SECURITY.md) - Security practices (15 min)
5. [Migration Guide](MIGRATION_GUIDE.md) - Upgrades (15 min)

#### **Security/Compliance Officer (2 hours)**
1. [Security Policy](../SECURITY.md) - Security overview (30 min)
2. [Architecture - Security](architecture/ARCHITECTURE.md#security-architecture) - Security design (30 min)
3. [Mathematical Implementation](deployment/MATHEMATICAL_IMPLEMENTATION.md) - Validation algorithms (45 min)
4. [Compliance Module](../apps/backend/modules/compliance/) - DPDPA compliance (15 min)

---

## 📚 Root-Level Documentation

### Essential Reading

| Document | Purpose | Audience |
|----------|---------|----------|
| [README.md](../readme.md) | Platform overview, quick start, features | Everyone |
| [AGENTS.md](../AGENTS.md) | AI agent development guide | Developers |
| [TODO.md](../TODO.md) | Production readiness tracking, known issues | Everyone |
| [CHANGELOG.md](../CHANGELOG.md) | Version history, migration guides | Everyone |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Contribution guidelines | Contributors |
| [SECURITY.md](../SECURITY.md) | Security policy, best practices | Everyone |
| [PROJECT_REPORT.md](../PROJECT_REPORT.md) | Executive summary | Stakeholders |

---

## 📘 Core Documentation (docs/)

### 1. [User Manual](USER_MANUAL.md) (~25 pages)
**What it covers:**
- Dashboard navigation and features
- Finding exploration and management
- Scan configuration and execution
- Remediation workflows
- Compliance tracking
- Best practices

**When to read**: Getting started with the platform, daily usage

---

### 2. [Architecture](architecture/ARCHITECTURE.md) (~25 pages)
**What it covers:**
- System architecture overview
- Intelligence-at-Edge principles
- Component breakdown (Scanner, Backend, Databases, Frontend)
- Data flow architecture
- 3-level lineage hierarchy
- Deployment architecture (dev and production)
- Security architecture
- Scalability considerations

**When to read**: Understanding system design, architectural decisions, component interactions

---

### 3. [Workflow](architecture/WORKFLOW.md) (~30 pages)
**What it covers:**
- System setup workflow (step-by-step installation)
- Scan execution workflow
- Data ingestion workflow
- Classification workflow
- Lineage synchronization workflow
- Frontend visualization workflow
- Compliance reporting workflow
- Troubleshooting workflow

**When to read**: Setting up the system, running scans, troubleshooting issues

---

### 4. [Technical Specifications](development/TECHNICAL_SPECIFICATIONS.md) (~35 pages)
**What it covers:**
- Minimum and recommended system requirements
- Maximum capacity limits (tested)
- File size limits and processing times
- Performance limits (throughput, latency)
- Complete PostgreSQL schema (all tables)
- Complete Neo4j schema (nodes, relationships)
- Full API specifications (all endpoints)
- Performance benchmarks
- Scaling guidelines

**When to read**: Planning deployment, capacity planning, API integration, database design

---

### 5. [Tech Stack](development/TECH_STACK.md) (~18 pages)
**What it covers:**
- Backend technologies (Go, Gin, database drivers)
- Frontend technologies (Next.js, React, TypeScript)
- Scanner technologies (Python, spaCy, validators)
- Database technologies (PostgreSQL, Neo4j)
- Infrastructure technologies (Docker, Docker Compose)
- Development tools
- Technology decision matrix
- Dependency management

**When to read**: Understanding technology choices, evaluating alternatives, dependency management

---

### 6. [Mathematical Implementation](deployment/MATHEMATICAL_IMPLEMENTATION.md) (~20 pages)
**What it covers:**
- Verhoeff algorithm (Aadhaar validation) with complete tables
- Luhn algorithm (Credit card validation)
- PAN checksum (Weighted Modulo 26)
- All 11 PII type validators
- Risk scoring algorithms
- Deduplication algorithms
- Context extraction
- Performance optimizations

**When to read**: Understanding validation logic, implementing new validators, debugging false positives

---

### 7. [Limitations & Future Improvements](deployment/LIMITATIONS_AND_IMPROVEMENTS.md) (~22 pages)
**What it covers:**
- Current limitations (10 major limitations)
- Performance bottlenecks (4 critical bottlenecks)
- Known issues (with status and fixes)
- Future improvements roadmap (8 phases, Q2 2026 - Q3 2027)
- Technical debt assessment
- Strategic priorities

**When to read**: Understanding current constraints, planning future enhancements, roadmap planning

---

### 8. [API Reference](API.md) (~50 pages)
**What it covers:**
- Complete API endpoint reference
- Request/response formats
- Authentication (when implemented)
- Error handling
- Rate limiting
- WebSocket API
- SDK examples (JavaScript, Python, Go)
- Pagination and filtering

**When to read**: API integration, building custom clients, automation

---

## 🔧 Application Documentation

### Backend (apps/backend/)

| Document | Purpose |
|----------|---------|
| [Backend README](../apps/backend/README.md) | Go backend overview, API endpoints, module structure |
| [Backend API](../apps/backend/README.md#api-endpoints) | Detailed API documentation |

**Key Components:**
- 8 Business Modules (assets, scanning, lineage, compliance, remediation, masking, analytics, connections)
- Temporal workflow integration
- Neo4j graph operations
- PostgreSQL data persistence

### Frontend (apps/frontend/)

| Document | Purpose |
|----------|---------|
| [Frontend README](../apps/frontend/README.md) | Next.js frontend overview, component structure |
| [Development Guide](../apps/frontend/README.md#development) | Frontend development guide |

**Key Features:**
- ReactFlow lineage visualization
- Real-time WebSocket updates
- DPDPA compliance dashboard
- Remediation actions

### Scanner (apps/scanner/)

| Document | Purpose |
|----------|---------|
| [Scanner README](../apps/scanner/README.md) | Python scanner overview, CLI usage |
| [Configuration](../apps/scanner/README.md#configuration) | Connection and scanner configuration |
| [SDK Architecture](../apps/scanner/README.md#sdk-architecture) | Intelligence-at-Edge SDK |

**Key Features:**
- 11+ PII type validators
- 10+ data source connectors
- Mathematical validation algorithms
- Context-aware detection

---

## 🛠️ Development Guides

### Getting Started

- [Development Setup](development/setup.md) - Complete development environment setup
- [AGENTS.md](../AGENTS.md) - AI agent development guide with build commands

### Contributing

- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
- [Code of Conduct](../CONTRIBUTING.md#code-of-conduct) - Community standards
- [Pull Request Process](../CONTRIBUTING.md#pull-request-process) - PR guidelines

### Testing

- [Testing Strategy](../AGENTS.md#testing-strategy) - Overview
- [Backend Testing](../apps/backend/README.md#testing) - Go testing
- [Frontend Testing](../apps/frontend/README.md#testing) - React testing
- [Scanner Testing](../apps/scanner/README.md#testing) - Python testing

---

## 🚀 Operations Guides

### Deployment

- [Deployment Guide](deployment/guide.md) - Complete deployment procedures
- [Phase 1 Deployment](phase1_deployment.md) - Phase 1 specific deployment
- [Migration Guide](MIGRATION_GUIDE.md) - Upgrade procedures

### Monitoring & Maintenance

- [Failure Modes](FAILURE_MODES.md) - Troubleshooting guide
- [TODO.md](../TODO.md) - Known issues and roadmap
- [CHANGELOG.md](../CHANGELOG.md) - Version history

### Security

- [Security Policy](../SECURITY.md) - Security practices and vulnerability reporting
- [Architecture - Security](architecture/ARCHITECTURE.md#security-architecture) - Security design
- [Backend Security](../apps/backend/README.md#security) - API security

---

## 📊 Advanced Topics

### Seamless Scanning

- [Seamless Scanning Guide](SEAMLESS_SCANNING.md) - Advanced scanning configurations
- Multi-source scanning
- Performance tuning
- Custom patterns

### Compliance

- DPDPA 2023 compliance mapping
- Consent tracking
- Retention policies
- Compliance reporting

### Lineage

- 3-level semantic hierarchy
- Graph visualization
- Impact analysis
- Lineage synchronization

---

## 🔍 Documentation by Topic

### Architecture & Design
- [Architecture](architecture/ARCHITECTURE.md)
- [Workflow](architecture/WORKFLOW.md)
- [Tech Stack](development/TECH_STACK.md)
- [PROJECT_REPORT.md](../PROJECT_REPORT.md)

### Implementation & Algorithms
- [Mathematical Implementation](deployment/MATHEMATICAL_IMPLEMENTATION.md)
- [Technical Specifications](development/TECHNICAL_SPECIFICATIONS.md)
- [SDK Architecture](../apps/scanner/README.md#sdk-architecture)

### Setup & Operations
- [Development Setup](development/setup.md)
- [Deployment Guide](deployment/guide.md)
- [Technical Specifications - Requirements](development/TECHNICAL_SPECIFICATIONS.md#system-requirements)
- [Migration Guide](MIGRATION_GUIDE.md)

### Usage & Features
- [User Manual](USER_MANUAL.md)
- [Workflow - Scan Execution](architecture/WORKFLOW.md#scan-execution-workflow)
- [Seamless Scanning](SEAMLESS_SCANNING.md)

### API & Integration
- [API Reference](API.md)
- [Technical Specifications - API](development/TECHNICAL_SPECIFICATIONS.md#api-specifications)
- [Workflow - Data Ingestion](architecture/WORKFLOW.md#data-ingestion-workflow)
- [Backend API](../apps/backend/README.md#api-endpoints)

### Database & Schema
- [Technical Specifications - Schemas](development/TECHNICAL_SPECIFICATIONS.md#database-schemas)
- [Architecture - Database](architecture/ARCHITECTURE.md#relational-database-postgresql)

### Troubleshooting
- [Failure Modes](FAILURE_MODES.md)
- [TODO.md](../TODO.md)
- [Workflow - Troubleshooting](architecture/WORKFLOW.md#troubleshooting-workflow)

### Performance & Scaling
- [Technical Specifications - Benchmarks](development/TECHNICAL_SPECIFICATIONS.md#performance-benchmarks)
- [Architecture - Scalability](architecture/ARCHITECTURE.md#scalability-considerations)
- [Limitations & Improvements](deployment/LIMITATIONS_AND_IMPROVEMENTS.md)

### Security & Compliance
- [Security Policy](../SECURITY.md)
- [Architecture - Security](architecture/ARCHITECTURE.md#security-architecture)
- [Mathematical Implementation](deployment/MATHEMATICAL_IMPLEMENTATION.md)

### Future Planning
- [Limitations & Improvements](deployment/LIMITATIONS_AND_IMPROVEMENTS.md)
- [TODO.md](../TODO.md)
- [CHANGELOG.md](../CHANGELOG.md)

---

## 📊 Documentation Statistics

| Document | Pages | Sections | Code Examples | Tables/Diagrams |
|----------|-------|----------|---------------|-----------------|
| Architecture | 25 | 15 | 5 | 3 |
| Mathematical Implementation | 20 | 12 | 15 | 2 |
| Workflow | 30 | 8 | 25 | 1 |
| Technical Specifications | 35 | 10 | 20 | 15 |
| Tech Stack | 18 | 8 | 3 | 5 |
| Limitations & Improvements | 22 | 10 | 0 | 3 |
| API Reference | 50 | 12 | 40 | 20 |
| User Manual | 25 | 8 | 10 | 5 |
| **Total Core Docs** | **225** | **83** | **118** | **54** |

---

## 🎓 Additional Resources

### Application READMEs
- [Backend README](../apps/backend/README.md) - Go backend
- [Frontend README](../apps/frontend/README.md) - Next.js frontend
- [Scanner README](../apps/scanner/README.md) - Python scanner

### Architecture SOPs
- [Scanning SOP](../architecture/scanning-sop.md)
- [Lineage SOP](../architecture/lineage-sop.md)
- [Compliance SOP](../architecture/compliance-sop.md)
- [Ingestion SOP](../architecture/ingestion-sop.md)

### Test Data Documentation
- [Ground Truth README](../testdata/ground_truth/README.md)
- [Aadhaar Numbers](../testdata/ground_truth/aadhaar_numbers.md)
- [PAN Numbers](../testdata/ground_truth/pan_numbers.md)
- [SSN Numbers](../testdata/ground_truth/ssn_numbers.md)
- [Credit Cards](../testdata/ground_truth/credit_cards.md)
- [Emails](../testdata/ground_truth/emails.md)
- [Negative Samples](../testdata/ground_truth/negative_samples.md)

---

## 📝 Documentation Maintenance

### Last Updated
- **Date**: February 10, 2026
- **Version**: 2.1.0
- **Status**: Current and verified

### Update Frequency
- **Core Documentation**: Updated with each major release
- **API Documentation**: Updated with each API change
- **User Guides**: Updated as needed for feature changes
- **Troubleshooting**: Updated as issues are discovered and resolved

### Contributing to Documentation
1. Follow markdown best practices
2. Include code examples where applicable
3. Add diagrams for complex concepts (using mermaid)
4. Keep language clear and concise
5. Update this index when adding new documents
6. Follow the [Documentation Style Guide](#documentation-style-guide)

---

## 🆘 Getting Help

### Documentation Issues
If you find errors, outdated information, or missing content:
1. Open an issue on GitHub
2. Tag with `documentation` label
3. Provide specific page and section references

### Questions Not Covered
1. Check [Failure Modes](FAILURE_MODES.md) for troubleshooting
2. Review [TODO.md](../TODO.md) for known issues
3. Search GitHub Issues and Discussions
4. Open a new discussion on GitHub

### Support Channels
- **Documentation**: [docs/INDEX.md](./INDEX.md)
- **GitHub Issues**: [Report bugs/issues](https://github.com/your-org/arc-hawk/issues)
- **GitHub Discussions**: [Ask questions](https://github.com/your-org/arc-hawk/discussions)
- **Email**: support@arc-hawk.io

---

## ✅ Pre-Deployment Checklist

Before deploying or making changes, ensure you've reviewed:

- [ ] [README.md](../readme.md) - Platform overview
- [ ] [Architecture](architecture/ARCHITECTURE.md) - System design
- [ ] [Deployment Guide](deployment/guide.md) - Installation
- [ ] [Technical Specifications](development/TECHNICAL_SPECIFICATIONS.md#system-requirements) - Requirements
- [ ] [Security Policy](../SECURITY.md) - Security practices
- [ ] [Failure Modes](FAILURE_MODES.md) - Troubleshooting
- [ ] [TODO.md](../TODO.md) - Known issues

---

## 🔗 External Resources

### Official Documentation
- [Next.js Documentation](https://nextjs.org/docs)
- [Go Documentation](https://golang.org/doc)
- [Python Documentation](https://docs.python.org/3/)
- [Neo4j Documentation](https://neo4j.com/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Temporal Documentation](https://docs.temporal.io/)

### Tools & Libraries
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [ReactFlow](https://reactflow.dev/docs)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [spaCy](https://spacy.io/usage)

---

**Need help finding something?** Use your browser's search (Ctrl+F / Cmd+F) or refer to the topic-based navigation above.

**Last Updated**: February 10, 2026  
**Documentation Version**: 2.1.0  
**Platform Version**: 2.1.0
