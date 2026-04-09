# Migration Guide

---

## v2.1 → v3.0.0

### Overview

v3.0.0 adds 9 new Postgres migrations, a custom patterns engine, DPDPA compliance expansion, enterprise discovery, and 14 new scanner sources.

**Migration Time**: ~10 minutes  
**Downtime Required**: Rolling (migrations are additive, no column drops)  
**Rollback Available**: Yes (each migration has a `.down.sql`)

### Breaking Changes

- `DELETE /api/v1/scans/:id` now requires `user_role == "admin"`. Non-admin callers receive HTTP 403. Update any automation that deletes scans.

### Step 1: Run Postgres Migrations

```bash
cd apps/backend

# Using golang-migrate
migrate -path migrations_versioned -database "postgres://..." up

# Or run each file manually in order:
# 000025_custom_patterns.up.sql
# 000026_retention_policies.up.sql
# 000027_encrypt_pii_samples.up.sql
# 000028_api_keys.up.sql
# 000029_audit_log_chain.up.sql
# 000030_consent_records.up.sql
# 000031_risk_score_history.up.sql
# 000032_custom_patterns_hardening.up.sql
# 000033_asset_dpdpa_fields.up.sql
```

All migrations are additive. No data loss. No lock issues on Postgres 11+.

### Step 2: Update Environment Variables

New optional env vars (all have defaults):

```env
# Encryption key for PII samples at rest (32-byte hex)
PII_ENCRYPTION_KEY=your-32-byte-hex-key

# Max custom pattern length (default: 512)
CUSTOM_PATTERN_MAX_LEN=512
```

### Step 3: Deploy Backend + Scanner

```bash
# Backend
cd apps/backend
go build -o arc-backend cmd/server/main.go
./arc-backend

# Scanner (install new dependencies)
cd apps/scanner
pip install -r requirements.txt
```

### Step 4: Verify

```bash
# Patterns API
curl http://localhost:8080/api/v1/patterns

# Discovery API
curl http://localhost:8080/api/v1/discovery

# Confirm DELETE /scans/:id returns 403 for non-admin tokens
```

### Rollback

```bash
migrate -path migrations_versioned -database "postgres://..." down 9
```

---

## v2.0 → v2.1.0

## Overview

This guide covers the migration from ARC-Hawk v2.0 (4-level lineage hierarchy) to v2.1.0 (3-level lineage hierarchy).

**Migration Time**: ~15-30 minutes  
**Downtime Required**: Yes (~5 minutes)  
**Rollback Available**: Yes (see Rollback section)

---

## What's Changing

### Lineage Hierarchy

**Before (v2.0)**:
```
System → Asset → DataCategory → PII_Category
Edges: CONTAINS, HAS_CATEGORY
```

**After (v2.1.0)**:
```
System → Asset → PII_Category
Edges: SYSTEM_OWNS_ASSET, EXPOSES
```

### Code Changes

- ✅ **Removed**: 4 backend service files (790 lines)
- ✅ **Updated**: 11 files across backend and frontend
- ✅ **Added**: Versioned Neo4j schema contract

---

## Prerequisites

Before starting the migration:

1. **Backup your data**
   ```bash
   # Backup PostgreSQL
   docker exec arc-hawk-postgres pg_dump -U postgres arc_hawk > backup_postgres_v2.0.sql
   
   # Backup Neo4j
   docker exec arc-hawk-neo4j neo4j-admin dump \
     --database=neo4j \
     --to=/backups/neo4j_v2.0_$(date +%Y%m%d).dump
   ```

2. **Verify current version**
   ```bash
   curl http://localhost:8080/health
   # Should show version 2.0.x
   ```

3. **Stop all services**
   ```bash
   # Stop backend
   pkill -f "go run cmd/server/main.go"
   
   # Stop frontend
   pkill -f "npm run dev"
   
   # Stop Docker containers
   docker-compose down
   ```

---

## Migration Steps

### Step 1: Update Code

```bash
# Pull latest changes
cd /Users/prathameshyadav/ARC-Hawk
git pull origin main

# Verify you're on v2.1.0
git log -1 --oneline
# Should show: "feat: Migrate to 3-level lineage hierarchy..."
```

### Step 2: Rebuild Backend

```bash
cd apps/backend

# Clean previous builds
go clean

# Download dependencies
go mod download

# Build
go build -o bin/server ./cmd/server

# Verify build
./bin/server --version
# Should show: v2.1.0
```

### Step 3: Rebuild Frontend

```bash
cd apps/frontend

# Install dependencies (in case of updates)
npm install

# Build for production
npm run build

# Verify build
ls -la .next/
# Should show fresh build directory
```

### Step 4: Migrate Neo4j Schema

```bash
# Start only Neo4j
docker-compose up -d neo4j

# Wait for Neo4j to be ready
sleep 10

# Apply new schema
cat apps/backend/migrations_versioned/neo4j_semantic_contract_v1.cypher | \
  docker exec -i arc-hawk-neo4j cypher-shell \
    -u neo4j \
    -p your_password_here

# Verify schema
docker exec -i arc-hawk-neo4j cypher-shell \
  -u neo4j \
  -p your_password_here \
  "CALL db.schema.visualization()"
```

**Expected Output**:
- Nodes: `System`, `Asset`, `PII_Category`
- Edges: `SYSTEM_OWNS_ASSET`, `EXPOSES`

### Step 5: Clean Old Data (Optional)

> [!WARNING]
> This step will delete all existing lineage data. Only proceed if you want a fresh start.

```bash
# Delete old hierarchy nodes and edges
docker exec -i arc-hawk-neo4j cypher-shell \
  -u neo4j \
  -p your_password_here \
  "MATCH (n:DataCategory) DETACH DELETE n"

# Verify cleanup
docker exec -i arc-hawk-neo4j cypher-shell \
  -u neo4j \
  -p your_password_here \
  "MATCH (n:DataCategory) RETURN count(n)"
# Should return: 0
```

### Step 6: Start Services

```bash
# Start infrastructure
docker-compose up -d

# Wait for services to be ready
sleep 5

# Start backend
cd apps/backend
go run cmd/server/main.go > ../../backend.log 2>&1 &

# Wait for backend
sleep 3

# Start frontend
cd ../frontend
npm run dev > ../../frontend.log 2>&1 &
```

### Step 7: Verify Migration

```bash
# Check backend health
curl http://localhost:8080/health
# Expected: {"status":"healthy","version":"2.1.0"}

# Check Neo4j connectivity
curl http://localhost:8080/api/v1/health/neo4j
# Expected: {"status":"connected"}

# Check lineage endpoint
curl http://localhost:8080/api/v1/lineage/v2
# Expected: JSON with 3-level hierarchy

# Check frontend
open http://localhost:3000
# Navigate to Lineage page and verify graph renders
```

---

## Re-scanning Data

After migration, you'll need to re-scan your data sources to populate the new hierarchy:

```bash
cd apps/scanner

# Run a test scan
python -m hawk_eye.cli scan \
  --profile filesystem \
  --path /path/to/test/data \
  --auto-ingest

# Verify ingestion
curl http://localhost:8080/api/v1/lineage/v2 | jq .
# Should show new nodes and edges
```

---

## Rollback Procedure

If you encounter issues, you can rollback to v2.0:

### Step 1: Stop Services

```bash
pkill -f "go run cmd/server/main.go"
pkill -f "npm run dev"
docker-compose down
```

### Step 2: Restore Code

```bash
# Checkout previous version
git checkout 02e31f8  # v2.0 commit hash

# Rebuild backend
cd apps/backend
go build ./cmd/server

# Rebuild frontend
cd ../frontend
npm run build
```

### Step 3: Restore Databases

```bash
# Restore PostgreSQL
docker-compose up -d postgres
cat backup_postgres_v2.0.sql | docker exec -i arc-hawk-postgres psql -U postgres arc_hawk

# Restore Neo4j
docker-compose up -d neo4j
docker exec arc-hawk-neo4j neo4j-admin load \
  --from=/backups/neo4j_v2.0_YYYYMMDD.dump \
  --database=neo4j \
  --force
```

### Step 4: Restart Services

```bash
docker-compose restart
cd apps/backend && go run cmd/server/main.go &
cd apps/frontend && npm run dev &
```

---

## Troubleshooting

### Issue: "DataCategory not found" errors

**Cause**: Old code trying to access deprecated nodes  
**Solution**: Ensure you've pulled latest code and rebuilt all services

```bash
git status  # Should show "nothing to commit"
cd apps/backend && go build ./cmd/server
cd apps/frontend && npm run build
```

### Issue: Lineage graph shows no data

**Cause**: Neo4j schema not migrated or data not re-scanned  
**Solution**: Run schema migration and re-scan

```bash
# Apply schema
cat apps/backend/migrations_versioned/neo4j_semantic_contract_v1.cypher | \
  docker exec -i arc-hawk-neo4j cypher-shell -u neo4j -p password

# Re-scan
cd apps/scanner
python -m hawk_eye.cli scan --profile filesystem --path /data --auto-ingest
```

### Issue: Frontend shows TypeScript errors

**Cause**: Outdated node_modules or build cache  
**Solution**: Clean and rebuild

```bash
cd apps/frontend
rm -rf node_modules .next
npm install
npm run build
npm run dev
```

### Issue: Backend fails to start

**Cause**: Port 8080 already in use or zombie process  
**Solution**: Kill existing processes

```bash
# Find process on port 8080
lsof -i :8080

# Kill it
kill -9 <PID>

# Restart backend
cd apps/backend
go run cmd/server/main.go
```

---

## Performance Validation

After migration, verify performance improvements:

```bash
# Benchmark lineage query (v2.0 baseline: ~300ms)
time curl http://localhost:8080/api/v1/lineage/v2

# Expected: ~200ms (30-40% improvement)
```

---

## Support

If you encounter issues not covered in this guide:

1. Check logs:
   ```bash
   tail -f backend.log
   tail -f frontend.log
   docker-compose logs neo4j
   ```

2. Verify schema:
   ```bash
   docker exec -i arc-hawk-neo4j cypher-shell -u neo4j -p password \
     "MATCH (n) RETURN labels(n), count(n)"
   ```

3. Review CHANGELOG.md for detailed changes

---

## Post-Migration Checklist

- [ ] All services running without errors
- [ ] Backend health check returns v2.1.0
- [ ] Neo4j shows 3-level hierarchy
- [ ] Frontend lineage graph renders correctly
- [ ] Re-scan completed successfully
- [ ] Performance metrics improved
- [ ] Backups stored safely
- [ ] Team notified of migration completion

---

**Migration Complete!** 🎉

Your ARC-Hawk platform is now running v2.1.0 with the optimized 3-level lineage hierarchy.
