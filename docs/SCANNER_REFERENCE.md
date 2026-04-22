# ARC-HAWK Scanner Reference

This document disambiguates the two scanner codebases and serves as the authoritative reference for the Go scanner.

---

## Canonical vs. Retired

| | Canonical (v3.0.0+) | Retired |
|-|---------------------|---------|
| **Location** | `apps/goScanner/` | `apps/scanner/` (Python — deleted) |
| **Language** | Go 1.24 | Python 3.9 (removed in v3.0.0) |
| **Port** | `:8001` | `:5000` |
| **Exposure** | Internal Docker network only | Was host-exposed |
| **Status** | Active | Do not use or reference |

If you see references to `apps/scanner/`, a Flask scanner on port `:5000`, or `python hawk_scanner`, those are stale and should be removed.

---

## Architecture

```
Backend :8080
    │
    │  POST /scan  (X-Scanner-Token)
    ▼
Go Scanner :8001 (arc-platform-go-scanner)
    │
    ├── Connector Pool (36+ connectors)
    │      S3, GCS, Azure Blob, PostgreSQL, MySQL, MongoDB,
    │      Redis, BigQuery, Snowflake, Redshift, Kafka,
    │      Kinesis, Slack, Salesforce, HubSpot, Jira,
    │      MS Teams, Avro, Parquet, CSV, Excel, PPTX,
    │      HTML, Email (.eml/.msg), PDF, Firebase, ...
    │
    ├── PII Classifier
    │      Multi-signal: regex + Presidio + mathematical
    │      validation (Verhoeff/Luhn/Modulo-26)
    │
    └── Ingest loop → POST /api/v1/scans/ingest-verified
             (X-Scanner-Token + X-Tenant-ID headers)
             Batched NDJSON, streamed back to backend
```

The scanner is a **pure sidecar**: it receives scan jobs from the backend, executes them, and streams findings back. It has no direct database access — all persistence goes through the backend API.

---

## Authentication

### Backend → Scanner (triggering a scan)

The backend sends `X-Scanner-Token: <SCANNER_SERVICE_TOKEN>` on every request to the scanner's `/scan` endpoint. The scanner validates this token with constant-time comparison.

```
POST http://go-scanner:8001/scan
X-Scanner-Token: <shared-secret>
Content-Type: application/json
```

In release mode (`GIN_MODE=release`), `SCANNER_SERVICE_TOKEN` must be at least 32 characters and must not equal the default placeholder. The backend enforces this at startup and will `log.Fatal` if the constraint is violated.

### Scanner → Backend (ingesting results)

The scanner sends two headers on every ingest batch:

```
X-Scanner-Token: <SCANNER_SERVICE_TOKEN>
X-Tenant-ID: <tenant-id>
```

The backend auth middleware accepts `X-Scanner-Token` in lieu of a JWT for ingest endpoints.

---

## Streaming Protocol

The scanner uses an internal channel-based streaming pipeline:

1. The connector produces `FieldRecord` items on a Go channel.
2. The PII classifier reads from the channel and scores each field.
3. Confirmed findings are accumulated into batches (default: 100 items).
4. Each batch is POSTed to `POST /api/v1/scans/ingest-verified` as JSON.
5. After all connectors finish, `SendScanComplete` is called to mark the scan done.

This is **not** a public-facing streaming protocol — the scanner and backend communicate over an internal HTTP connection on the Docker bridge network. External consumers use the REST API (see [docs/INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md)).

---

## Port

| Context | Value |
|---------|-------|
| Container port | `8001` |
| Published to host | **No** — `expose` only, no host port mapping |
| Reachable from backend | `http://go-scanner:8001` (Docker bridge) |
| Reachable from host | Not directly — by design |

The scanner port is intentionally not exposed to the host network to prevent bypass of scanner auth by network-adjacent callers.

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `SCANNER_SERVICE_TOKEN` | Yes | Shared secret with backend. Min 32 chars in release mode. |
| `BACKEND_URL` | Yes | Backend base URL, e.g. `http://backend:8080` |
| `PRESIDIO_URL` | Yes | Presidio analyzer URL, e.g. `http://presidio-analyzer:3000` |
| `SCAN_TIMEOUT_SECONDS` | No | Wall-clock cap per scan. Default: `1800` (30 min). |
| `PORT` | No | Listen port. Default: `8001`. |
| `LOG_LEVEL` | No | Logging verbosity: `debug` | `info` | `warn`. Default: `info`. |

---

## Connectors

The Go scanner supports 36+ connectors organized into categories:

**Databases**: PostgreSQL, MySQL, MongoDB, Redis, MSSQL, SQLite, BigQuery, Redshift, Snowflake

**Cloud Storage**: AWS S3, Google Cloud Storage (GCS), Azure Blob Storage

**Streaming**: Apache Kafka (snapshot sampling), AWS Kinesis (snapshot sampling)

**SaaS**: Slack, Salesforce, HubSpot, Jira, MS Teams (requires `tenant_id`)

**Files**: Text, CSV, Excel, PDF, PPTX, HTML, Email (`.eml`/`.msg`), Avro, Parquet

**Beta**: Firebase Realtime DB, Firestore, Google Drive

Each connector implements the `Connector` interface:

```go
type Connector interface {
    Stream(ctx context.Context, out chan<- FieldRecord) error
}
```

---

## Health Check

The scanner exposes a health endpoint on its listen port:

```
GET http://go-scanner:8001/health
```

Response (healthy):

```json
{ "status": "ok", "service": "arc-hawk-go-scanner" }
```

Docker Compose health check:

```yaml
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:8001/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 20s
```

---

## Building Locally

```bash
cd apps/goScanner

# Run tests
go test ./...

# Build binary
go build -o go-scanner cmd/scanner/main.go

# Run
SCANNER_SERVICE_TOKEN=dev-token-32chars-minimum-here \
BACKEND_URL=http://localhost:8080 \
PRESIDIO_URL=http://localhost:3000 \
./go-scanner
```

---

## Related Documents

- [docs/RUNBOOK_E2E.md](./RUNBOOK_E2E.md) — Full system startup guide
- [docs/INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md) — API integration reference
- [docs/releases/v3.0.0.md](./releases/v3.0.0.md) — v3.0.0 release notes (Python scanner removal)
