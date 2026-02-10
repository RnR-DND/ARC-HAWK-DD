# ARC-Hawk Backend

The core API and business logic layer for the ARC-Hawk platform, built with **Go 1.21+** following Clean Architecture principles.

## 🌟 Features

- **Modular Monolith**: 8 distinct business modules with clear separation of concerns
- **Real-time Updates**: WebSocket integration for live scan progress and notifications
- **Workflow Orchestration**: Temporal integration for reliable, long-running scan & remediation jobs
- **Graph Lineage**: Neo4j integration for semantic data mapping and visualization
- **Clean Architecture**: Strict separation of Handler → Service → Repository → Domain
- **Comprehensive API**: RESTful endpoints with consistent response formats

## 📂 Module Structure

```
apps/backend/
├── cmd/
│   ├── server/
│   │   └── main.go              # Application entry point
│   ├── neo4j_migrate/
│   │   └── main.go              # Neo4j schema migrations
│   └── test_data_generator/
│       └── main.go              # Test data generation
├── modules/                      # Business modules
│   ├── assets/                   # Asset inventory management
│   ├── scanning/                 # Scan orchestration & ingestion
│   ├── lineage/                  # Neo4j graph operations
│   ├── compliance/               # DPDPA compliance & consent tracking
│   ├── connections/              # Data source connections
│   ├── remediation/              # Data masking/deletion
│   ├── masking/                  # PII masking service
│   ├── analytics/                # Risk scoring & dashboard stats
│   ├── fplearning/               # ML pattern learning (utility)
│   ├── auth/                     # Authentication (partial)
│   └── websocket/                # Real-time updates
├── pkg/                          # Shared packages
│   ├── validation/               # PII validators (Luhn, Verhoeff)
│   └── normalization/            # Data normalization
├── configs/                      # Configuration files
└── migrations_versioned/         # Database migrations
```

### Module Details

#### 1. Assets Module (`modules/assets/`)
Manages the inventory of data sources and assets.

**Key Components:**
- `api/asset_handler.go` - HTTP handlers
- `service/asset_service.go` - Business logic
- `repository/asset_repository.go` - Data access
- `domain/asset.go` - Domain models

**Features:**
- CRUD operations for assets
- Asset categorization
- Metadata management
- Asset discovery

#### 2. Scanning Module (`modules/scanning/`)
Core scanning functionality with Temporal workflow orchestration.

**Key Components:**
- `api/scan_trigger_handler.go` - Scan initiation
- `api/scan_ingestion_handler.go` - Result ingestion
- `workflows/scan_workflow.go` - Temporal workflows
- `activities/scan_activities.go` - Temporal activities
- `service/ingestion_service.go` - Data ingestion
- `worker/scan_worker.go` - Temporal worker

**Features:**
- Async scan execution
- Real-time progress tracking
- Multi-source result ingestion
- Finding deduplication

#### 3. Lineage Module (`modules/lineage/`)
Neo4j-based graph lineage tracking.

**Key Components:**
- `api/lineage_handler_v2.go` - Lineage API (v2)
- `service/semantic_lineage_service.go` - Graph operations
- `repository/neo4j_repository.go` - Neo4j queries

**Features:**
- 3-level hierarchy (System → Asset → PII Type)
- Graph visualization data
- Impact analysis
- Lineage synchronization

#### 4. Compliance Module (`modules/compliance/`)
DPDPA 2023 compliance tracking and reporting.

**Key Components:**
- `api/compliance_handler.go` - Compliance endpoints
- `service/compliance_service.go` - Compliance logic

**Features:**
- Consent tracking
- Retention policy management
- Compliance reporting
- Data principal rights

#### 5. Connections Module (`modules/connections/`)
Data source connection management.

**Key Components:**
- `api/connection_handler.go` - Connection CRUD
- `service/connection_service.go` - Connection logic

**Features:**
- Multiple source types (PostgreSQL, MySQL, S3, etc.)
- Credential management
- Connection testing
- Profile management

#### 6. Remediation Module (`modules/remediation/`)
Automated remediation actions.

**Key Components:**
- `api/remediation_handler.go` - Remediation endpoints
- `service/remediation_service.go` - Remediation logic

**Features:**
- Masking operations
- Deletion operations
- Preview functionality
- Action history

#### 7. Analytics Module (`modules/analytics/`)
Dashboard analytics and risk scoring.

**Key Components:**
- `api/analytics_handler.go` - Analytics endpoints
- `service/analytics_service.go` - Analytics logic
- `service/risk_scoring.go` - Risk calculations

**Features:**
- Dashboard statistics
- Risk scoring algorithms
- Trend analysis
- Report generation

#### 8. Masking Module (`modules/masking/`)
PII masking operations and policies.

**Key Components:**
- `api/masking_handler.go` - Masking endpoints
- `service/masking_service.go` - Masking logic

**Features:**
- Masking policies
- Masking strategies
- Column-level masking
- Masking previews

## 🚀 Getting Started

### Prerequisites

- **Go 1.21+**
- **PostgreSQL 15+**
- **Neo4j 5.15+**
- **Temporal Server** (or use Docker Compose)

### Environment Variables

Create a `.env` file in `apps/backend/`:

```bash
# Server Configuration
PORT=8080
ENV=development

# Database - PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=arc_hawk
DB_SSL_MODE=disable

# Database - Neo4j
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your_password

# Temporal
TEMPORAL_ADDRESS=localhost:7233
TEMPORAL_NAMESPACE=default
TEMPORAL_TASK_QUEUE=arc-hawk-scans

# Scanner
SCAN_ID_PREFIX=scan
MAX_SCAN_WORKERS=10

# Optional: JWT (when auth is implemented)
JWT_SECRET=your_jwt_secret_min_32_chars
JWT_EXPIRATION_HOURS=24
```

### Running Locally

```bash
# 1. Install Dependencies
go mod tidy

# 2. Run Database Migrations (if needed)
go run cmd/neo4j_migrate/main.go

# 3. Run Server
go run cmd/server/main.go

# Server will start on http://localhost:8080
```

### Running with Docker

```bash
# From project root
docker-compose up -d postgres neo4j temporal

# Then run the backend
cd apps/backend
go run cmd/server/main.go
```

## 🔌 API Endpoints

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication
> **Note**: Authentication is not yet implemented. All endpoints are currently public.

### Response Format
All API responses follow a consistent format:

```json
{
  "success": true,
  "data": { ... },
  "message": "Operation completed successfully"
}
```

Error responses:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message"
  }
}
```

### API Endpoints Reference

#### Scanning

**Trigger a Scan**
```http
POST /api/v1/scans/trigger
Content-Type: application/json

{
  "asset_id": "asset-uuid",
  "scan_type": "full",
  "priority": "normal"
}
```

**Check Scan Status**
```http
GET /api/v1/scans/{id}/status
```

**Ingest Scan Results** (Internal)
```http
POST /api/v1/scans/ingest
Content-Type: application/json

{
  "scan_id": "scan-uuid",
  "findings": [...]
}
```

#### Findings

**List Findings**
```http
GET /api/v1/findings?status=open&asset_id=xxx&pii_type=aadhaar&page=1&limit=50
```

**Get Finding Details**
```http
GET /api/v1/findings/{id}
```

**Update Finding Status**
```http
PATCH /api/v1/findings/{id}
Content-Type: application/json

{
  "status": "false_positive",
  "feedback": "Not actual PII"
}
```

**Submit Feedback**
```http
POST /api/v1/findings/{id}/feedback
Content-Type: application/json

{
  "is_false_positive": true,
  "reason": "Test data"
}
```

#### Assets

**List Assets**
```http
GET /api/v1/assets?page=1&limit=50&source_type=postgresql
```

**Create Asset**
```http
POST /api/v1/assets
Content-Type: application/json

{
  "name": "Production PostgreSQL",
  "source_type": "postgresql",
  "connection_id": "conn-uuid",
  "metadata": {...}
}
```

**Get Asset Details**
```http
GET /api/v1/assets/{id}
```

**Update Asset**
```http
PUT /api/v1/assets/{id}
Content-Type: application/json

{
  "name": "Updated Name",
  "metadata": {...}
}
```

**Delete Asset**
```http
DELETE /api/v1/assets/{id}
```

#### Lineage

**Get Lineage Graph**
```http
GET /api/v1/lineage/v2
```

**Get Lineage for Asset**
```http
GET /api/v1/lineage/asset/{asset_id}
```

**Get PII Distribution**
```http
GET /api/v1/lineage/pii-distribution
```

**Get Impact Analysis**
```http
GET /api/v1/lineage/impact/{asset_id}
```

#### Connections

**List Connections**
```http
GET /api/v1/connections
```

**Create Connection**
```http
POST /api/v1/connections
Content-Type: application/json

{
  "name": "Production DB",
  "type": "postgresql",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "mydb"
  },
  "credentials": {
    "username": "admin",
    "password": "secret"
  }
}
```

**Test Connection**
```http
POST /api/v1/connections/{id}/test
```

**Delete Connection**
```http
DELETE /api/v1/connections/{id}
```

#### Compliance

**Get Compliance Overview**
```http
GET /api/v1/compliance/overview
```

**Get DPDPA Report**
```http
GET /api/v1/compliance/dpdpa-report
```

**Update Consent Status**
```http
PUT /api/v1/compliance/consent/{finding_id}
Content-Type: application/json

{
  "consent_status": "granted",
  "consent_date": "2026-01-15T10:00:00Z"
}
```

#### Remediation

**Get Available Actions**
```http
GET /api/v1/remediation/actions?finding_id=xxx
```

**Execute Remediation**
```http
POST /api/v1/remediation/execute
Content-Type: application/json

{
  "finding_id": "finding-uuid",
  "action": "mask",
  "strategy": "partial",
  "reason": "DPDPA compliance"
}
```

**Get Remediation History**
```http
GET /api/v1/remediation/history?finding_id=xxx
```

**Preview Remediation**
```http
POST /api/v1/remediation/preview
Content-Type: application/json

{
  "finding_id": "finding-uuid",
  "action": "mask",
  "strategy": "partial"
}
```

#### Analytics

**Get Dashboard Stats**
```http
GET /api/v1/analytics/dashboard
```

**Get Risk Score**
```http
GET /api/v1/analytics/risk-score
```

**Get PII Trends**
```http
GET /api/v1/analytics/trends?days=30
```

**Get Asset Statistics**
```http
GET /api/v1/analytics/assets/{asset_id}/stats
```

#### WebSocket

**Connect to Real-time Updates**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Update:', data);
};
```

**Subscribe to Scan Updates**
```javascript
ws.send(JSON.stringify({
  action: 'subscribe',
  channel: 'scan:{scan_id}'
}));
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
go test ./... -v

# Run specific module tests
go test ./modules/scanning/... -v
go test ./modules/lineage/... -v

# Run with coverage
go test ./... -cover

# Run with race detection
go test ./... -race

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Structure

```
modules/
├── scanning/
│   └── *_test.go          # Unit tests
├── lineage/
│   └── *_test.go
└── ...
```

### Integration Tests

```bash
# Run integration tests (requires running services)
go test ./tests/integration/... -v
```

## 📊 Database Schema

### PostgreSQL Tables

**Core Tables:**
- `assets` - Asset inventory
- `findings` - PII findings
- `scans` - Scan executions
- `connections` - Data source connections
- `compliance_records` - Compliance tracking
- `remediation_actions` - Remediation history

**See:** `docs/development/TECHNICAL_SPECIFICATIONS.md` for complete schema

### Neo4j Graph

**Nodes:**
- `System` - Data systems
- `Asset` - Data assets
- `PIIType` - PII categories

**Relationships:**
- `SYSTEM_OWNS_ASSET` - System to asset
- `EXPOSES` - Asset to PII type

**See:** `migrations_versioned/neo4j_semantic_contract_v1.cypher`

## 🔧 Configuration

### Application Configuration

Configuration is loaded from:
1. Environment variables (highest priority)
2. `.env` file
3. Default values

### Feature Flags

```go
// configs/config.go
type Config struct {
    EnableAuth       bool   `env:"ENABLE_AUTH" default:"false"`
    EnableWebsocket  bool   `env:"ENABLE_WEBSOCKET" default:"true"`
    EnableTemporal   bool   `env:"ENABLE_TEMPORAL" default:"true"`
    MaxScanWorkers   int    `env:"MAX_SCAN_WORKERS" default:"10"`
}
```

## 🐛 Troubleshooting

### Common Issues

**1. Database Connection Failed**
```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Check connection string
cat .env | grep DB_
```

**2. Neo4j Connection Failed**
```bash
# Check Neo4j is running
docker-compose ps neo4j

# Verify credentials
cat .env | grep NEO4J_
```

**3. Temporal Connection Failed**
```bash
# Check Temporal is running
docker-compose ps temporal

# Verify address
cat .env | grep TEMPORAL
```

### Logs

```bash
# View logs
tail -f logs/app.log

# Enable debug logging
export LOG_LEVEL=debug
go run cmd/server/main.go
```

## 📚 Additional Documentation

- [Architecture Overview](../docs/architecture/ARCHITECTURE.md)
- [API Specifications](../docs/development/TECHNICAL_SPECIFICATIONS.md)
- [Workflow Documentation](../docs/architecture/WORKFLOW.md)
- [Failure Modes](../docs/FAILURE_MODES.md)

## 🤝 Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines.

## 📝 License

Apache License 2.0 - See [LICENSE](../LICENSE) for details.
