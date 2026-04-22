# ARC-HAWK End-to-End Runbook

**Version**: v3.0.0  
**Supersedes**: `docs/SEAMLESS_SCANNING.md`, `docs/phase1_deployment.md`

This runbook covers everything needed to start, operate, and verify a complete ARC-HAWK stack from scratch.

---

## Prerequisites

| Requirement | Minimum Version | Check |
|-------------|----------------|-------|
| Docker Desktop / Docker Engine | Latest stable | `docker --version` |
| Docker Compose (v2 plugin) | 2.0+ | `docker compose version` |
| Go | 1.24+ | `go version` |
| Node.js | 18+ | `node --version` |
| npm | 9+ | `npm --version` |

**Hardware**: 8 GB RAM recommended (Neo4j 1 GB + Temporal 1 GB + backend 1 GB + frontend 512 MB + scanner 512 MB).

---

## Environment Setup

Copy and edit the env file before first run:

```bash
cp .env.example .env
# Edit .env and fill in:
#   POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB
#   NEO4J_USERNAME, NEO4J_PASSWORD
#   ENCRYPTION_KEY  (exactly 32 bytes, generate with: openssl rand -hex 16)
#   SCANNER_SERVICE_TOKEN  (min 32 chars, generate with: openssl rand -hex 32)
#   JWT_SECRET
```

---

## Step 1 — Start Infrastructure

Start all backing services (PostgreSQL, Neo4j, Temporal, Presidio, Vault):

```bash
docker compose up -d postgres neo4j presidio-analyzer temporal temporal-ui vault
```

Wait for services to become healthy:

```bash
docker compose ps
# All critical services should show "healthy" or "Up"
```

Expected healthy services before proceeding:

- `arc-platform-db` (PostgreSQL)
- `arc-platform-neo4j` (Neo4j)
- `arc-platform-presidio` (Presidio)
- `arc-platform-temporal` (Temporal)
- `arc-platform-vault` (HashiCorp Vault)

---

## Step 2 — Run Database Migrations

Migrations run automatically on backend startup. To run them manually:

```bash
cd apps/backend

# Install golang-migrate if not present
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run all up migrations
migrate \
  -path migrations_versioned \
  -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${POSTGRES_DB}?sslmode=disable" \
  up
```

Current migration level: `000045` (fp_learning).

To check current version:

```bash
migrate \
  -path migrations_versioned \
  -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${POSTGRES_DB}?sslmode=disable" \
  version
```

To roll back one step:

```bash
migrate ... down 1
```

---

## Step 3 — Start the Go Scanner

The Go scanner runs as a sidecar on `:8001` (internal Docker network only). In Docker Compose, it starts automatically:

```bash
docker compose up -d go-scanner
```

For local development outside Docker:

```bash
cd apps/goScanner
SCANNER_SERVICE_TOKEN=<your-token> \
BACKEND_URL=http://localhost:8080 \
PRESIDIO_URL=http://localhost:3000 \
SCAN_TIMEOUT_SECONDS=1800 \
go run cmd/scanner/main.go
```

---

## Step 4 — Start the Backend

```bash
docker compose up -d backend
```

For local development:

```bash
cd apps/backend
go mod tidy
go run cmd/server/main.go
```

The backend starts on `:8080` and runs all pending migrations automatically.

---

## Step 5 — Start the Frontend

```bash
docker compose up -d frontend
```

For local development:

```bash
cd apps/frontend
npm install
npm run dev
```

Frontend available at `http://localhost:3000`.

---

## Step 6 — Verify Health

```bash
# Backend liveness (always 200 while process runs)
curl http://localhost:8080/livez

# Backend readiness (checks PostgreSQL + Neo4j)
curl http://localhost:8080/readyz

# Go scanner health (only accessible from Docker internal network)
# From the backend container:
docker exec arc-platform-backend wget -qO- http://go-scanner:8001/health

# PostgreSQL
docker exec arc-platform-db pg_isready -U postgres

# Neo4j
curl http://localhost:7474/
```

Expected `readyz` response when all dependencies are up:

```json
{
  "status": "ready",
  "service": "arc-platform-backend",
  "db_healthy": true,
  "neo4j_healthy": true
}
```

---

## Step 7 — End-to-End Scan Walkthrough

### 7.1 Register and Login

```bash
# Register an admin user (first-time only)
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"changeme123!"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"changeme123!"}' \
  | jq -r .token)

echo "Got token: ${TOKEN:0:20}..."
```

### 7.2 Create a Connection

```bash
CONN_ID=$(curl -s -X POST http://localhost:8080/api/v1/connections \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "local-postgres",
    "source_type": "postgresql",
    "config": {
      "host": "postgres",
      "port": 5432,
      "database": "arc_platform",
      "username": "postgres",
      "password": "'"${POSTGRES_PASSWORD}"'"
    }
  }' | jq -r .id)

echo "Connection ID: $CONN_ID"
```

### 7.3 Trigger a Scan

```bash
SCAN_ID=$(curl -s -X POST http://localhost:8080/api/v1/scans/trigger \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "connection_id": "'"$CONN_ID"'",
    "scan_name": "e2e-test-scan"
  }' | jq -r .scan_id)

echo "Scan ID: $SCAN_ID"
```

### 7.4 Poll Status

```bash
while true; do
  STATUS=$(curl -s "http://localhost:8080/api/v1/scans/$SCAN_ID/status" \
    -H "Authorization: Bearer $TOKEN" | jq -r .status)
  echo "Status: $STATUS"
  if [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]]; then
    break
  fi
  sleep 5
done
```

### 7.5 View Findings

```bash
curl -s "http://localhost:8080/api/v1/findings?scan_id=$SCAN_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '{total: .total, sample: .findings[:3]}'
```

### 7.6 Open the Dashboard

Navigate to `http://localhost:3000` in your browser. The scan results appear automatically in the Findings and Dashboard panels.

---

## All-in-One (Docker Compose Only)

If you want to run everything through Docker Compose without any local Go/Node tooling:

```bash
# Start everything
docker compose up -d

# Watch logs
docker compose logs -f backend go-scanner

# Stop everything
docker compose down

# Stop and wipe all data volumes
docker compose down -v
```

---

## Service Ports

| Service | Port | Access |
|---------|------|--------|
| Frontend (Next.js) | 3000 | Host |
| Backend API | 8080 | Host |
| Go Scanner | 8001 | Internal Docker network only |
| PostgreSQL | 5432 | Host (127.0.0.1 only) |
| Neo4j Browser | 7474 | Host (127.0.0.1 only) |
| Neo4j Bolt | 7687 | Host (127.0.0.1 only) |
| Temporal | 7233 | Host (127.0.0.1 only) |
| Temporal UI | 8088 | Host (127.0.0.1 only) |
| Vault | 8200 | Host (127.0.0.1 only) |
| Presidio | 3000 | Internal Docker network only |
| Prometheus | 9090 | Host (127.0.0.1 only) |
| Grafana | 3002 | Host (127.0.0.1 only) |

---

## Troubleshooting

### PostgreSQL will not start

```bash
docker compose logs postgres | tail -30
# Common fix: remove stale volume
docker compose down -v && docker compose up -d postgres
```

### Neo4j exits(1) on first start

Expected behavior — Neo4j 5.x with APOC plugin exits once after initial install, then restarts cleanly. The `restart: on-failure` policy handles this automatically.

```bash
docker compose logs neo4j | tail -20
# If still failing after 2 restarts, check memory limits:
# Increase docker desktop memory allocation to 8 GB+
```

### Temporal fails to connect to PostgreSQL

Ensure `POSTGRES_SEEDS=postgres` (not `postgres:5432`) in your `.env`. The seed value is the hostname only.

### Backend fails: "SCANNER_SERVICE_TOKEN must be at least 32 characters in release mode"

Generate a proper token:

```bash
openssl rand -hex 32
# Add to .env: SCANNER_SERVICE_TOKEN=<output>
```

### "connection refused" calling go-scanner from backend

The scanner is only reachable within the Docker bridge network. If running the backend locally (not in Docker), set `SCANNER_URL=http://localhost:8001` and start the scanner locally too.

### Frontend shows stale data

Hard-refresh the browser (`Cmd+Shift+R` / `Ctrl+Shift+R`). The frontend uses `fetch` with `no-cache` on scan status endpoints.

---

## Related Documents

- [docs/INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md) — API integration reference
- [docs/SCANNER_REFERENCE.md](./SCANNER_REFERENCE.md) — Go scanner details
- [docs/releases/v3.0.0.md](./releases/v3.0.0.md) — What changed in v3.0.0
- [DEPLOYMENT_RUNBOOK.md](../DEPLOYMENT_RUNBOOK.md) — Kubernetes and production deployment
