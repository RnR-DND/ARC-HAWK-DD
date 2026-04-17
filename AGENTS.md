# AGENTS.md - AI Agent Development Guide

This file provides essential information for AI agents working on the ARC-Hawk codebase.

## Project Overview

ARC-Hawk is an enterprise-grade PII (Personally Identifiable Information) discovery, classification, and lineage tracking platform built with a microservices architecture.

**Technology Stack:**
- **Backend**: Go 1.24+ (Gin framework), PostgreSQL 15, Neo4j 5.15, Temporal workflow engine
- **Frontend**: Next.js 14.0.4, TypeScript 5.3.3, ReactFlow, Cytoscape, Tailwind CSS
- **Scanner**: Python 3.9+, spaCy NLP, custom validation algorithms (Verhoeff, Luhn)
- **Infrastructure**: Docker, Docker Compose, Kubernetes

## Build Commands

### Frontend (Next.js)
```bash
cd apps/frontend
npm install          # Install dependencies
npm run dev          # Development server (localhost:3000)
npm run build        # Production build
npm run start        # Production server
npm run lint         # ESLint linting
```

### Backend (Go)
```bash
cd apps/backend
go mod tidy         # Update dependencies
go run cmd/server/main.go    # Development server (localhost:8080)
go test ./...       # Run all tests
go test ./modules/scanning    # Run specific module tests
go build            # Build binary
```

### Scanner (Python)
```bash
cd apps/scanner
pip install -r requirements.txt
python -m hawk_scanner.main fs --connection config/connection.yml --json output.json
python -m pytest tests/  # Run tests
python -m pytest tests/test_specific.py  # Run specific test
```

### Infrastructure
```bash
docker-compose up -d    # Start all services
docker-compose down    # Stop all services
```

### Testing Commands
```bash
# Run all tests
./scripts/testing/run-tests.sh

# Individual component tests
cd apps/backend && go test ./...
cd apps/frontend && npm test -- --passWithNoTests
cd apps/scanner && python -m pytest

# Run single test file
cd apps/backend && go test ./modules/scanning -v
cd apps/scanner && python -m pytest tests/test_validation.py -v
```

## Code Style Guidelines

### Go Backend
- **Formatting**: Use `gofmt` (standard Go formatting)
- **Architecture**: Modular monolith with domain-driven modules
- **Module Structure**: 8 core modules (scanning, assets, lineage, compliance, masking, analytics, connections, remediation) + 1 utility module (fplearning)
- **Import Organization**: Standard Go conventions, third-party imports grouped
- **Naming**: CamelCase for exported, camelCase for unexported
- **Error Handling**: Explicit error returns, use fmt.Errorf for wrapped errors

### Frontend (TypeScript/Next.js)
- **TypeScript**: Strict mode enabled (`"strict": true`)
- **Imports**: Use absolute imports with `@/*` alias for internal modules
- **Component Structure**: Functional components with React hooks
- **Styling**: Tailwind CSS for utility classes, CSS Modules for component-specific styles
- **File Naming**: PascalCase for components, kebab-case for utilities
- **Type Definitions**: Use TypeScript interfaces for all data structures

### Python Scanner
- **Formatting**: Follow PEP 8 standard Python conventions
- **Package Structure**: setuptools-based packaging with `hawk_scanner` module
- **Dependencies**: Pin specific versions for critical packages (e.g., pymongo==4.6.3)
- **Testing**: Use pytest framework
- **Documentation**: Docstrings for all public functions and classes

## Directory Structure

```
ARC-Hawk/
├── apps/
│   ├── scanner/              # Python PII detection engine
│   │   ├── sdk/             # Validation algorithms & recognizers
│   │   ├── hawk_scanner/    # Multi-source scanning logic
│   │   ├── tests/           # Test suites
│   │   └── config/          # Scanner configurations
│   ├── backend/             # Go API server
│   │   ├── modules/         # 7 business modules
│   │   ├── cmd/server/      # Application entry point
│   │   └── tests/           # Backend tests
│   └── frontend/            # Next.js Dashboard
│       ├── app/             # Pages (App Router)
│       ├── components/      # Reusable UI components
│       └── services/        # API clients
├── docs/                    # Comprehensive documentation
├── infra/                   # Docker & Kubernetes configs
└── scripts/                 # Automation & testing scripts
```

## Development Workflow

### Quick Start (5 minutes)
1. `docker-compose up -d` - Start infrastructure services
2. `cd apps/backend && go run cmd/server/main.go` - Start backend server
3. `cd apps/frontend && npm run dev` - Start frontend dashboard
4. `cd apps/scanner && python -m hawk_scanner.main fs --connection config/connection.yml` - Run scan

### Key Development Principles
- **Intelligence-at-Edge**: Scanner SDK is sole authority for PII validation
- **Modular Architecture**: Clear separation between scanning, assets, lineage, compliance, masking, analytics, and connections
- **API-First**: Backend provides RESTful APIs for all frontend operations
- **Multi-Source Support**: Scanner supports filesystem, PostgreSQL, MySQL, MongoDB, S3, GCS, Redis, Slack

## Testing Strategy

### Backend Testing
- Unit tests for each module using Go's built-in testing package
- Integration tests for database operations
- Use `go test ./modules/<module_name>` for module-specific tests

### Frontend Testing
- Component tests using React Testing Library
- Use `npm test -- --passWithNoTests` (currently no tests configured)
- Add tests in `__tests__` directories alongside components

### Scanner Testing
- Unit tests for validation algorithms using pytest
- Integration tests for different data sources
- Use `python -m pytest tests/` for all scanner tests

## Configuration Files

### Key Configuration Locations
- `docker-compose.yml` - Multi-service infrastructure setup
- `apps/backend/.env.example` - Backend environment template
- `apps/scanner/sdk/config.yml` - Scanner SDK configuration
- `apps/frontend/next.config.js` - Next.js configuration
- `apps/frontend/tsconfig.json` - TypeScript configuration with strict mode

### Environment Variables
- Backend: Use `.env` file in `apps/backend/` directory
- Frontend: Use `process.env.API_URL` for API endpoint configuration
- Scanner: Use YAML configuration files in `config/` directory

## Common Patterns

### Error Handling
- **Go**: Return explicit errors, use structured error responses
- **TypeScript**: Use try-catch with proper error types, display user-friendly messages
- **Python**: Use exceptions with meaningful error messages

### Database Operations
- **PostgreSQL**: Use GORM for ORM operations in Go backend
- **Neo4j**: Use official Neo4j driver for graph operations
- **Connections**: Handle connection pooling and proper resource cleanup

### API Design
- RESTful endpoints with consistent response format
- Use HTTP status codes appropriately
- Include pagination for list endpoints
- Provide detailed error messages with error codes

## Performance Considerations

- Scanner throughput: 200-350 files/second
- Backend: Use connection pooling, implement proper caching
- Frontend: Use React.memo for expensive components, implement virtualization for large lists
- Database: Optimize queries, use proper indexing

## Security Guidelines

- Never commit secrets or API keys
- Use environment variables for sensitive configuration
- Implement proper authentication and authorization
- Validate all input data
- Use HTTPS for all API communications

## Deployment

- Docker-based deployment with multi-stage builds
- Use GitHub Actions for CI/CD (`.github/workflows/`)
- Kubernetes configurations in `infra/` directory
- Follow semantic versioning for releases

---

## 🧰 Agentic Toolchain

This project ships 4 agent subsystems in `.agent/` + a memory layer integration.
Full runbook: `docs/integrations/agent-toolchain.md`. Summary:

### Antigravity Awesome Skills (`.agent/skills/`) — 1414 `@name` cards
Expert markdown skills, loaded with `@skill-name` in prompts. No install.
- Backend: `@golang-pro`, `@api-design-principles`, `@backend-security-coder`
- Frontend: `@nextjs-best-practices`, `@react-best-practices`, `@typescript-expert`
- Infra: `@docker-expert`, `@postgres-best-practices`
- Security: `@api-security-best-practices`
- Testing: `@test-driven-development`, `@e2e-testing-patterns`

### Ralph (`.agent/ralph/`) — autonomous PRD loop
`prd.json` with user stories → bash loop implements + tests + commits each.
Seeded at `.agent/ralph/prd.arc-hawk-dd.json` with Session 14 deferred work.
```bash
cp .agent/ralph/prd.arc-hawk-dd.json .agent/ralph/prd.json && bash .agent/ralph/ralph.sh
```

### Hive (`.agent/hive/`) — YC-backed multi-agent runtime
Apache 2.0. 102 MCP tools, self-healing agent graphs. Requires uv + bun first-run.
```bash
cd .agent/hive && uv sync && bun install
```

### Agentsys (`.agent/agentsys/`) — strict orchestration runtime
Plain-text output, no emoji, strict PR discipline. Cooperates with Claude Code.
```bash
cd .agent/agentsys && node bin/cli.js --help
```

### Game-studios (`.agent/game-studios/`) — NOT APPLICABLE
Game-dev subagents (Godot/Unity/Unreal). Kept in tree for completeness; ignore
unless pivoting.

### Supermemory.ai — memory + hybrid search layer
Backend module: `apps/backend/modules/memory/`. Free tier: 1M tokens + 10K queries/mo.
Auto-records scan summaries; exposes `/api/v1/memory/search` for hybrid RAG.
MCP server wired into Claude Code via `.claude/settings.local.json` (gitignored).
Setup + rotation: `docs/integrations/supermemory.md`.

**To enable locally:**
```bash
# apps/backend/.env
SUPERMEMORY_ENABLED=true
SUPERMEMORY_API_KEY=sm_your_key_here
```