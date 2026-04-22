# Technology Stack

## Backend
- **Language**: Go 1.24
- **Framework**: Gin (HTTP), Gorilla (WebSocket)
- **Orchestration**: Temporal.io (Workflow Engine)
- **Database**:
    - PostgreSQL 15 (Relational)
    - Neo4j 5.15 (Graph)
- **Architecture**: Modular Monolith

## Frontend
- **Framework**: Next.js 14 (App Router)
- **Language**: TypeScript 5.3
- **State**: React Hooks + Context
- **Visualization**: ReactFlow (Graph), Recharts (Charts)
- **Styling**: Tailwind CSS

## Scanner
- **Language**: Go 1.24 (`apps/goScanner/`)
- **NLP**: Presidio (via HTTP to `presidio-analyzer` container)
- **Validation**: Custom Algorithms (Verhoeff, Luhn, Weighted Modulo-26)
- **Connectors**: 36+ (databases, cloud, SaaS, files)
- **Communication**: REST (receives scan jobs from backend, streams findings back)
- **Port**: `:8001` internal Docker network only
- **Note**: Python scanner (`apps/scanner/`) removed in v3.0.0

## Infrastructure
- **Containerization**: Docker, Docker Compose v2+
- **CI/CD**: GitHub Actions (planned)
- **Monitoring**: Prometheus + Grafana (metrics endpoints at `/metrics`)
- **Secret Management**: HashiCorp Vault (dev mode in docker-compose)

## Developer Tools
- **OpenAPI**: swaggo (`github.com/swaggo/swag`) — generates `docs/openapi/openapi.yaml` from Go annotations
- **Migrations**: golang-migrate (`github.com/golang-migrate/migrate/v4`)
- **Go Version**: 1.24+
