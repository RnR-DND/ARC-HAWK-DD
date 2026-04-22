# E2E Test Harness

This directory contains end-to-end tests for ARC-HAWK.

---

## Prerequisites

Before running any e2e tests, ensure the full stack is running:

1. Docker Compose services healthy (PostgreSQL, Neo4j, Temporal, Presidio, Go scanner, backend, frontend)
2. At least one test user registered (default: `admin@example.com` / `changeme123!`)
3. Backend reachable at `http://localhost:8080`
4. Frontend reachable at `http://localhost:3000`

Quick stack check:

```bash
curl -sf http://localhost:8080/livez && echo "backend alive"
curl -sf http://localhost:8080/readyz && echo "backend ready"
```

See [docs/RUNBOOK_E2E.md](../../docs/RUNBOOK_E2E.md) for full startup instructions.

---

## Test Scripts

### `full-scan.sh` (coming soon)

A comprehensive shell script that exercises the complete scan lifecycle:

1. Login and obtain JWT
2. Register a test connection
3. Trigger a scan and poll until completion
4. Assert findings are present
5. Assert dashboard metrics are non-zero
6. Clean up test data

The script will be added in a subsequent PR. Track progress in [TODO.md](../../TODO.md).

---

## Manual curl Walkthrough

Use this as a fallback when the automated harness is not yet available.

### Step 1 — Login

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"changeme123!"}' \
  | jq -r .token)

echo "token=${TOKEN:0:30}..."
```

### Step 2 — Create a Connection

```bash
CONN_ID=$(curl -s -X POST http://localhost:8080/api/v1/connections \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "e2e-test-pg",
    "source_type": "postgresql",
    "config": {
      "host": "postgres",
      "port": 5432,
      "database": "arc_platform",
      "username": "postgres",
      "password": "postgres"
    }
  }' | jq -r .id)

echo "connection_id=$CONN_ID"
```

### Step 3 — Trigger Scan

```bash
SCAN_ID=$(curl -s -X POST http://localhost:8080/api/v1/scans/trigger \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"connection_id":"'"$CONN_ID"'","scan_name":"e2e-manual"}' \
  | jq -r .scan_id)

echo "scan_id=$SCAN_ID"
```

### Step 4 — Poll Until Done

```bash
for i in $(seq 1 60); do
  STATUS=$(curl -s "http://localhost:8080/api/v1/scans/$SCAN_ID/status" \
    -H "Authorization: Bearer $TOKEN" | jq -r .status)
  echo "[$i] status=$STATUS"
  if [[ "$STATUS" == "completed" || "$STATUS" == "failed" ]]; then
    break
  fi
  sleep 5
done
```

### Step 5 — Assert Findings

```bash
TOTAL=$(curl -s "http://localhost:8080/api/v1/findings?scan_id=$SCAN_ID" \
  -H "Authorization: Bearer $TOKEN" | jq .total)

echo "total_findings=$TOTAL"
[ "$TOTAL" -gt 0 ] && echo "PASS: findings present" || echo "FAIL: no findings"
```

### Step 6 — Dashboard Metrics

```bash
curl -s http://localhost:8080/api/v1/dashboard/metrics \
  -H "Authorization: Bearer $TOKEN" | jq '{totalPII, highRiskFindings, assetsHit}'
```

---

## Environment Variables for Tests

| Variable | Default | Description |
|----------|---------|-------------|
| `ARC_BASE_URL` | `http://localhost:8080` | Backend API base URL |
| `ARC_FRONTEND_URL` | `http://localhost:3000` | Frontend URL for Playwright tests |
| `ARC_TEST_EMAIL` | `admin@example.com` | Test user email |
| `ARC_TEST_PASSWORD` | `changeme123!` | Test user password |

---

## Playwright Tests (Future)

Browser-based tests using Playwright will live in this directory. They will cover:

- Login flow
- Connection creation UI
- Scan trigger and progress display
- Findings table rendering
- Compliance dashboard

To run when available:

```bash
npx playwright install
npx playwright test tests/e2e/
```
