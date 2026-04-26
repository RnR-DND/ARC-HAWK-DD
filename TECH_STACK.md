# ARC-HAWK-DD Technology Stack

Generated from: `apps/backend/go.mod`, `apps/goScanner/go.mod`, `apps/agent/go.mod`, `apps/frontend/package.json`, `docker-compose.yml`, all `Dockerfile`s, `.github/workflows/`, `Makefile`, `infra/prometheus/`.

---

## Languages & Runtimes

| Component | Language | Version |
|-----------|----------|---------|
| Backend API | Go | 1.25 |
| Go Scanner | Go | 1.22 |
| Edge Agent | Go | 1.22 |
| Frontend | Node.js | 20 (Alpine) |
| Frontend | TypeScript | 5.3.3 |
| Frontend Framework | Next.js | ^16.1.6 |
| Frontend UI | React | ^19.2.4 |

---

## Backend (Go) — `apps/backend/go.mod`

### Web & API
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/gin-gonic/gin` | v1.9.1 | HTTP router/framework |
| `github.com/gin-contrib/cors` | v1.5.0 | CORS middleware |
| `github.com/gorilla/websocket` | v1.5.3 | WebSocket support |
| `github.com/swaggo/swag` | v1.16.6 | OpenAPI spec generation |
| `github.com/swaggo/gin-swagger` | v1.6.1 | Swagger UI for Gin |
| `github.com/swaggo/files` | v1.0.1 | Swagger static files |

### Auth & Security
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/golang-jwt/jwt/v5` | v5.2.1 | JWT authentication |
| `github.com/hashicorp/vault/api` | v1.23.0 | HashiCorp Vault secrets |
| `golang.org/x/crypto` | v0.48.0 | Cryptographic primitives |

### Databases & Storage
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/lib/pq` | v1.10.9 | PostgreSQL driver |
| `github.com/go-sql-driver/mysql` | v1.7.1 | MySQL driver |
| `go.mongodb.org/mongo-driver` | v1.14.0 | MongoDB driver |
| `github.com/neo4j/neo4j-go-driver/v5` | v5.28.4 | Neo4j graph DB driver |
| `github.com/redis/go-redis/v9` | v9.18.0 | Redis client |
| `github.com/golang-migrate/migrate/v4` | v4.17.1 | DB schema migrations |

### Cloud & External Services
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/aws/aws-sdk-go` | v1.49.6 | AWS SDK v1 |

### Workflow & Async
| Package | Version | Purpose |
|---------|---------|---------|
| `go.temporal.io/sdk` | v1.25.0 | Temporal workflow SDK |

### Observability
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/prometheus/client_golang` | v1.23.2 | Prometheus metrics |
| `go.opentelemetry.io/otel` | v1.41.0 | OpenTelemetry core |
| `go.opentelemetry.io/otel/sdk` | v1.35.0 | OTel SDK |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.24.0 | OTLP gRPC trace exporter |
| `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` | v0.49.0 | Gin OTel instrumentation |

### Utilities
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/google/uuid` | v1.6.0 | UUID generation |
| `github.com/joho/godotenv` | v1.5.1 | `.env` file loading |
| `github.com/go-pdf/fpdf` | v0.9.0 | PDF generation |
| `github.com/xuri/excelize/v2` | v2.10.1 | Excel file generation |

### Testing
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/stretchr/testify` | v1.11.1 | Test assertions |
| `github.com/DATA-DOG/go-sqlmock` | v1.5.2 | SQL mock for tests |
| `github.com/testcontainers/testcontainers-go` | v0.42.0 | Docker-based integration tests |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | v0.42.0 | Postgres testcontainer |

---

## Scanner (Go) — `apps/goScanner/go.mod`

### Web & API
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/gin-gonic/gin` | v1.10.0 | HTTP router/framework |

### Databases & Data Sources (Connectors)
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/lib/pq` | v1.10.9 | PostgreSQL connector |
| `github.com/go-sql-driver/mysql` | v1.8.1 | MySQL connector |
| `github.com/microsoft/go-mssqldb` | v1.7.2 | SQL Server connector |
| `go.mongodb.org/mongo-driver` | v1.15.0 | MongoDB connector |
| `github.com/snowflakedb/gosnowflake` | v1.10.1 | Snowflake connector |
| `github.com/redis/go-redis/v9` | v9.5.1 | Redis connector |
| `modernc.org/sqlite` | v1.29.9 | SQLite (local scanner storage) |

### Cloud Connectors
| Package | Version | Purpose |
|---------|---------|---------|
| `cloud.google.com/go/bigquery` | v1.61.0 | Google BigQuery connector |
| `cloud.google.com/go/storage` | v1.40.0 | Google Cloud Storage connector |
| `github.com/aws/aws-sdk-go-v2` | v1.27.0 | AWS SDK v2 core |
| `github.com/aws/aws-sdk-go-v2/service/s3` | v1.54.1 | AWS S3 connector |
| `github.com/aws/aws-sdk-go-v2/service/kinesis` | v1.27.4 | AWS Kinesis connector |
| `github.com/Azure/azure-sdk-for-go/sdk/storage/azblob` | v1.0.0 | Azure Blob Storage (indirect) |
| `google.golang.org/api` | v0.175.0 | Google APIs client |

### File Format Support
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/linkedin/goavro/v2` | v2.12.0 | Apache Avro format |
| `github.com/parquet-go/parquet-go` | v0.23.0 | Apache Parquet format |
| `github.com/scritchley/orc` | v0.0.0-20210513144143 | Apache ORC format |
| `github.com/xuri/excelize/v2` | v2.8.1 | Excel format |

### Messaging & Streaming
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/segmentio/kafka-go` | v0.4.47 | Apache Kafka connector |

### Integrations
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/andygrunwald/go-jira/v2` | v2.0.0 | Jira integration |
| `github.com/slack-go/slack` | v0.13.0 | Slack notifications |

### Observability
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/prometheus/client_golang` | v1.20.0 | Prometheus metrics |
| `go.opentelemetry.io/otel` | v1.24.0 | OpenTelemetry (indirect) |

### Utilities
| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/joho/godotenv` | v1.5.1 | `.env` file loading |
| `golang.org/x/sync` | v0.10.0 | Concurrency utilities |

---

## Edge Agent (Go) — `apps/agent/go.mod`

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/gin-gonic/gin` | v1.10.0 | HTTP API |
| `github.com/mattn/go-sqlite3` | v1.14.24 | Local SQLite storage (CGO) |
| `github.com/robfig/cron/v3` | v3.0.1 | Scheduled scan jobs |
| `go.uber.org/zap` | v1.27.0 | Structured logging |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML config parsing |

---

## Frontend (Next.js) — `apps/frontend/package.json`

### Production Dependencies
| Package | Version | Purpose |
|---------|---------|---------|
| `next` | ^16.1.6 | Next.js framework |
| `react` | ^19.2.4 | React UI library |
| `react-dom` | ^19.2.4 | React DOM renderer |
| `axios` | ^1.6.2 | HTTP client |
| `@radix-ui/react-avatar` | ^1.1.11 | Accessible avatar component |
| `@radix-ui/react-dialog` | ^1.1.15 | Accessible modal/dialog |
| `@radix-ui/react-dropdown-menu` | ^2.1.16 | Dropdown menu primitive |
| `@radix-ui/react-scroll-area` | ^1.2.10 | Custom scroll area |
| `@radix-ui/react-separator` | ^1.1.8 | Visual separator |
| `@radix-ui/react-slot` | ^1.2.4 | Slot composition primitive |
| `lucide-react` | ^0.562.0 | Icon library |
| `framer-motion` | ^12.28.1 | Animation library |
| `class-variance-authority` | ^0.7.1 | Variant-based class generation (CVA) |
| `clsx` | ^2.1.1 | Conditional class names |
| `tailwind-merge` | ^2.6.1 | Tailwind class merging |
| `tailwindcss-animate` | ^1.0.7 | Tailwind animation utilities |
| `recharts` | ^3.7.0 | Charting library |
| `reactflow` | ^11.10.3 | Node-based graph/flow UI |
| `cytoscape` | ^3.33.1 | Graph theory/visualization |
| `dagre` | ^0.8.5 | Directed graph layout |
| `@types/dagre` | ^0.7.53 | TypeScript types for dagre |
| `date-fns` | ^4.1.0 | Date utility library |

### Dev Dependencies
| Package | Version | Purpose |
|---------|---------|---------|
| `typescript` | ^5.3.3 | TypeScript compiler |
| `eslint` | ^9.39.2 | JavaScript linter |
| `eslint-config-next` | ^16.1.6 | Next.js ESLint config |
| `tailwindcss` | ^3.4.17 | CSS utility framework |
| `postcss` | ^8.4.38 | CSS post-processor |
| `autoprefixer` | ^10.4.19 | CSS vendor prefixing |
| `jest` | ^27.5.1 | Unit test runner |
| `jest-environment-jsdom` | ^27.5.1 | DOM environment for Jest |
| `@testing-library/react` | ^13.4.0 | React component testing |
| `@testing-library/jest-dom` | ^5.17.0 | DOM assertion matchers |
| `@testing-library/user-event` | ^13.5.0 | User interaction simulation |
| `@playwright/test` | ^1.57.0 | E2E test framework |
| `@types/node` | ^20.10.5 | Node.js TypeScript types |
| `@types/react` | ^18.2.45 | React TypeScript types |
| `@types/react-dom` | ^18.2.18 | React DOM TypeScript types |

---

## Databases & Storage

| System | Version | Role | Port |
|--------|---------|------|------|
| PostgreSQL | 15-alpine | Primary relational DB (tenants, users, findings, assets, scans) | 5432 |
| Neo4j Community | 5.15 + APOC | Graph DB — data lineage, relationships | 7474 / 7687 |
| Redis | (via go-redis/v9) | Caching, classify queue, pub/sub | — |
| MongoDB | (via mongo-driver) | Document storage connector | — |
| MySQL | (via go-sql-driver) | External data source connector | — |
| Microsoft SQL Server | (via go-mssqldb) | External data source connector | — |
| Snowflake | (via gosnowflake) | External data warehouse connector | — |
| SQLite | modernc (scanner) / mattn CGO (agent) | Local storage (scanner metadata, agent state) | — |
| AWS S3 | (via aws-sdk-go-v2) | External object storage connector | — |
| Google Cloud Storage | (via cloud.google.com/go/storage) | External object storage connector | — |
| Google BigQuery | (via cloud.google.com/go/bigquery) | External data warehouse connector | — |
| Azure Blob Storage | (via Azure SDK) | External object storage connector | — |

---

## Infrastructure & DevOps

| Tool | Version | Purpose |
|------|---------|---------|
| Docker | — | Container runtime |
| Docker Compose | — | Local multi-service orchestration |
| GitHub Actions | — | CI/CD pipeline |
| Helm | — | Kubernetes packaging (`helm/arc-hawk-dd/`) |
| Temporal | 1.22.0 | Workflow orchestration engine |
| Temporal UI | 2.21.0 | Temporal web dashboard |
| HashiCorp Vault | 1.15 | Secrets management |
| Microsoft Presidio | latest | PII detection / data anonymization |
| `appleboy/ssh-action` | v1.0.3 | SSH-based production deploy |
| `docker/setup-buildx-action` | v3 | Docker BuildKit setup |
| `docker/build-push-action` | v6 | Docker image build & push |
| `docker/login-action` | v3 | Docker Hub authentication |
| `docker/metadata-action` | v5 | Docker image tagging |
| `actions/setup-go` | v5 | Go toolchain setup in CI |
| `actions/setup-node` | v4 | Node.js toolchain setup in CI |
| Registry | ghcr.io / Docker Hub | Container image registry |

### Docker Images Used
| Image | Version | Service |
|-------|---------|---------|
| `golang` | 1.25-alpine / 1.22-bookworm | Build stages |
| `alpine` | 3.20 | Runtime (backend, scanner) |
| `debian` | bookworm-slim | Runtime (agent) |
| `node` | 20-alpine | Runtime (frontend) |
| `postgres` | 15-alpine (dev), 16-alpine (CI) | Database |
| `neo4j` | 5.15-community | Graph database |
| `temporalio/auto-setup` | 1.22.0 | Temporal server |
| `temporalio/ui` | 2.21.0 | Temporal UI |
| `hashicorp/vault` | 1.15 | Secrets |
| `mcr.microsoft.com/presidio-analyzer` | latest | PII detection |
| `prom/prometheus` | latest | Metrics |
| `grafana/grafana` | latest | Dashboards |

---

## Observability

| Tool | Version | Purpose |
|------|---------|---------|
| Prometheus | latest | Metrics scraping & storage |
| Grafana | latest | Metrics dashboards |
| OpenTelemetry (OTel) | v1.41.0 (backend) / v1.24.0 (scanner) | Distributed tracing |
| OTLP/gRPC exporter | v1.24.0 | Trace export to collector |
| `otelgin` instrumentation | v0.49.0 | Auto-instrument Gin routes |
| Prometheus SLO rules | — | Scan success rate, Presidio latency alerts |

### SLO Alerts Defined
- `ScanSuccessRateLow` — warn < 95% for 10m
- `ScanSuccessRateCritical` — critical < 80% for 5m
- `PresidioLatencyHigh` — warn P99 > 5s for 5m
- `PresidioLatencyCritical` — critical P99 > 30s for 2m
- `ScanWatchdogStalled` — warn active scans > 0 for 2h

---

## Security

| Tool / Package | Version | Purpose |
|----------------|---------|---------|
| `gosec` | latest (CI) | Go static security analysis (SARIF upload) |
| `govulncheck` | latest (CI) | Go known vulnerability scanning |
| Trivy (`aquasecurity/trivy-action`) | master | Container CVE scanning (SARIF upload) |
| GitHub Dependency Review | v4 | PR-level dependency vulnerability check |
| `github.com/golang-jwt/jwt/v5` | v5.2.1 | JWT token generation/validation |
| `github.com/hashicorp/vault/api` | v1.23.0 | Vault-backed secrets at runtime |
| `golang.org/x/crypto` | v0.48.0 | bcrypt, AES, key derivation |
| `github.com/go-jose/go-jose/v4` | v4.1.1 | JOSE/JWK support (indirect) |
| Presidio | latest | PII detection before storage |
| Auth safety gate | — | CI check: no `AUTH_REQUIRED=false` in release mode |

---

## Testing

### Backend (Go)
| Tool | Version | Scope |
|------|---------|-------|
| `testify` | v1.11.1 | Unit test assertions |
| `go-sqlmock` | v1.5.2 | SQL interaction mocking |
| `testcontainers-go` | v0.42.0 | Integration tests with real Postgres in Docker |
| `go test -short` | — | Unit tests (CI fast path, skips DB-dependent) |
| `go test -tags=integration` | — | Integration tests (CI service Postgres) |
| `go vet` | — | Static analysis |
| Classifier quality gate | — | Regression: ≥90% pass rate required |

### Frontend (JavaScript)
| Tool | Version | Scope |
|------|---------|-------|
| Jest | v27.5.1 | Unit/component tests |
| `jest-environment-jsdom` | v27.5.1 | DOM simulation |
| `@testing-library/react` | v13.4.0 | React component rendering |
| `@testing-library/jest-dom` | v5.17.0 | DOM assertion matchers |
| `@testing-library/user-event` | v13.5.0 | User interaction simulation |
| Playwright | v1.57.0 | E2E browser tests (`e2e/playwright.config.ts`) |
| ESLint | v9.39.2 | Code quality linting |

### CI/CD Test Matrix
| Workflow | Trigger | Jobs |
|----------|---------|------|
| `build.yml` | push/PR to main | backend lint+test+build, scanner lint+test+build, frontend lint+build |
| `ci-cd.yml` | push main/develop, PR main, manual dispatch | lint → docker build → integration tests → push images → deploy |
| `regression.yml` | push/PR main/develop | classifier regression (≥90% gate), connector regression, orchestrator regression |
| `security.yml` | push/PR main, weekly Monday 08:00 UTC | gosec, govulncheck, Trivy container scan, GitHub dependency review |
