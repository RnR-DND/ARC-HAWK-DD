> **Historical**: This document describes Phase 1 deployment. For current deployment, see [docs/RUNBOOK_E2E.md](./RUNBOOK_E2E.md).

# Phase 1: Database-Backed Connections - Deployment Guide

## Prerequisites

- PostgreSQL running on `localhost:5432`
- Go 1.24+ installed
- Backend dependencies installed (`go mod download`)

## Quick Start

### Option 1: Automated Deployment (Recommended)

```bash
cd /Users/prathameshyadav/ARC-Hawk
./scripts/deploy_phase1.sh
```

This script will:
1. Generate a secure 32-byte encryption key
2. Update `.env` file with encryption key
3. Guide you through database migration

### Option 2: Manual Deployment

#### Step 1: Generate Encryption Key

```bash
# Generate 32-byte key for AES-256
openssl rand -base64 32 | head -c 32
```

#### Step 2: Update Environment Variables

Add to `apps/backend/.env`:

```bash
ENCRYPTION_KEY=<your-32-byte-key-here>
```

**CRITICAL**: The encryption key MUST be exactly 32 bytes. Save it securely!

#### Step 3: Run Database Migration

```bash
cd apps/backend
go run cmd/server/main.go
```

The migration `000009_add_connections_table.up.sql` will run automatically on startup.

#### Step 4: Verify Migration

```bash
psql -U postgres -d arc_hawk -c "\d connections"
```

Expected output:
```
                Table "public.connections"
      Column       |           Type           | Nullable
-------------------+--------------------------+----------
 id                | uuid                     | not null
 source_type       | character varying(50)    | not null
 profile_name      | character varying(255)   | not null
 config_encrypted  | bytea                    | not null
 validation_status | character varying(50)    | 
 ...
```

## Testing Phase 1

### Test 1: Add Connection via API

```bash
curl -X POST http://localhost:8080/api/v1/connections \
  -H "Content-Type: application/json" \
  -d '{
    "source_type": "database",
    "profile_name": "test-db",
    "config": {
      "host": "localhost:5432",
      "username": "testuser",
      "password": "testpass",
      "database": "testdb"
    }
  }'
```

Expected response:
```json
{
  "id": "uuid-here",
  "status": "success",
  "message": "Connection added successfully. Validation pending."
}
```

### Test 2: List Connections

```bash
curl http://localhost:8080/api/v1/connections
```

Expected response:
```json
{
  "connections": [
    {
      "id": "uuid-here",
      "source_type": "database",
      "profile_name": "test-db",
      "validation_status": "pending",
      "created_at": "2026-01-22T11:00:00Z"
    }
  ]
}
```

**Note**: Credentials are NOT returned for security.

### Test 3: Verify Encryption

```bash
psql -U postgres -d arc_hawk -c "SELECT id, profile_name, config_encrypted FROM connections LIMIT 1;"
```

The `config_encrypted` column should show binary data (not readable plain text).

## Rollback

If you need to rollback Phase 1:

```bash
cd apps/backend
migrate -path migrations_versioned -database "postgres://postgres:postgres@localhost:5432/arc_hawk?sslmode=disable" down 1
```

This will drop the `connections` table.

## Troubleshooting

### Error: "ENCRYPTION_KEY not set"

**Solution**: Add `ENCRYPTION_KEY` to `.env` file (must be exactly 32 bytes)

### Error: "ENCRYPTION_KEY must be exactly 32 bytes"

**Solution**: Regenerate key using `openssl rand -base64 32 | head -c 32`

### Error: "failed to encrypt config"

**Solution**: Verify encryption key is set correctly and is 32 bytes

### Error: "duplicate key value violates unique constraint"

**Solution**: Connection with same `source_type` and `profile_name` already exists. Use different name or delete existing connection.

## Next Steps

After Phase 1 is verified:
- **Phase 2**: Scan Entity Creation (Week 2)
- **Phase 3**: Temporal Workflow Integration (Week 3)
- **Phase 4**: Scanner Configuration Passing (Week 4)
- **Phase 5**: False Positive Learning (Week 5)
- **Phase 6**: Remediation State Tracking (Week 6)

## Security Notes

1. **Encryption Key Storage**: Store encryption key securely (e.g., AWS Secrets Manager, HashiCorp Vault)
2. **Key Rotation**: Not implemented in Phase 1 (future enhancement)
3. **Credential Access**: Only `GetConnectionWithConfig()` decrypts credentials (internal use only)
4. **API Security**: Credentials are never returned via API endpoints
