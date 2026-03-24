# Technical Specifications

## System Requirements

### Minimum Hardware
- **CPU**: 4 Cores (Recommended for concurrent scanning)
- **RAM**: 8GB (4GB for app, 4GB for databases)
- **Storage**: 20GB SSD

### Software Dependencies
- **Docker**: 24.0+
- **Docker Compose**: 2.0+

---

## Capacity Limits
- **Assets**: Tested up to 1,000,000 assets.
- **Graph Nodes**: Tested up to 500,000 nodes (Neo4j Community Edition).
- **Concurrency**: Temporal allows unlimited horizontal scaling of workers.

---

## API Specifications

### Base URL
`http://localhost:8080/api/v1`

### Endpoints

#### Scans
- `POST /scans/trigger` - Start Scan
- `POST /scans/ingest` - Worker Ingestion

#### Findings
- `GET /findings` - List Findings
- `PATCH /findings/:id/feedback` - False Positive

#### Remediation
- `POST /remediation/execute` - Start Remediation Workflow
- `GET /remediation/history` - View Actions

#### Lineage
- `GET /lineage` - Graph Data (Nodes/Edges)

#### Health
- `GET /health` - System Status
