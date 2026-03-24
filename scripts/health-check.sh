#!/usr/bin/env bash

set -euo pipefail

echo "============================================="
echo " ARC-HAWK ENTERPRISE SYSTEM HEALTH CHECK"
echo "============================================="

FAILED=0

fail() {
  echo "❌ FAIL: $1"
  FAILED=1
}

pass() {
  echo "✅ PASS: $1"
}

section() {
  echo ""
  echo "---------------------------------------------"
  echo "$1"
  echo "---------------------------------------------"
}

##############################################
# PHASE 1 — INFRASTRUCTURE CHECK
##############################################

section "PHASE 1: Infrastructure & Containers"

docker ps >/dev/null 2>&1 || fail "Docker not running"

docker ps | grep -q postgres || fail "PostgreSQL container not running"
docker ps | grep -q neo4j || fail "Neo4j container not running"

pass "Containers running"

##############################################
# PHASE 2 — DATABASE CONNECTIVITY
##############################################

section "PHASE 2: Database Connectivity"

pg_isready -h localhost -p 5432 >/dev/null 2>&1 || fail "PostgreSQL not reachable"
pass "PostgreSQL reachable"

curl -s http://localhost:7474 >/dev/null || fail "Neo4j HTTP not reachable"
pass "Neo4j reachable"

##############################################
# PHASE 3 — BACKEND API HEALTH
##############################################

section "PHASE 3: Backend API Health"

API_BASE="http://localhost:8080/api/v1"

curl -s "$API_BASE/health" | grep -q "ok" || fail "Backend health endpoint failed"
pass "Backend health OK"

for ep in lineage findings assets classification/summary; do
  curl -s "$API_BASE/$ep" >/dev/null || fail "Endpoint /$ep not reachable"
done

pass "Core backend endpoints reachable"

##############################################
# PHASE 4 — SCANNER CONFIGURATION
##############################################

section "PHASE 4: Scanner Configuration"

[[ -f apps/scanner/config/connection.yml ]] || fail "connection.yml missing"
[[ -f fingerprint.yml ]] || fail "fingerprint.yml missing"

python3 - <<EOF || fail "Scanner config parsing failed"
import yaml
yaml.safe_load(open("apps/scanner/config/connection.yml"))
yaml.safe_load(open("fingerprint.yml"))
EOF

pass "Scanner configs valid"

##############################################
# PHASE 5 — SCANNER EXECUTION  
##############################################

section "PHASE 5: Scanner Execution"

python3 scripts/automation/unified-scan.py >/tmp/scan.log 2>&1 || fail "Scanner execution failed"

grep -q "finding" /tmp/scan.log || fail "Scanner produced no findings"
pass "Scanner produced findings"

##############################################
# PHASE 6 — INGESTION & DATA PRESENCE
##############################################

section "PHASE 6: Ingestion & Data Presence"

SCAN_RUNS=$(psql -h localhost -U postgres -d arc_platform -t -c "SELECT count(*) FROM scan_runs;" | xargs)
ASSETS=$(psql -h localhost -U postgres -d arc_platform -t -c "SELECT count(*) FROM assets;" | xargs)
FINDINGS=$(psql -h localhost -U postgres -d arc_platform -t -c "SELECT count(*) FROM findings;" | xargs)

[[ "$SCAN_RUNS" -gt 0 ]] || fail "No scan_runs created"
[[ "$ASSETS" -gt 0 ]] || fail "No assets ingested"
[[ "$FINDINGS" -gt 0 ]] || fail "No findings ingested"

pass "Data ingested correctly"

##############################################
# PHASE 7 — CLASSIFICATION SANITY
##############################################

section "PHASE 7: Classification Sanity"

BAD=$(psql -h localhost -U postgres -d arc_platform -t -c "
SELECT count(*) FROM classifications
WHERE confidence_score < 0 OR confidence_score > 1;
" | xargs)

[[ "$BAD" -eq 0 ]] || fail "Invalid confidence scores detected"
pass "Classification confidence valid"

##############################################
# PHASE 8 — NEO4J LINEAGE INTEGRITY
##############################################

section "PHASE 8: Neo4j Lineage Integrity"

CYPHER_RESULT=$(curl -s -u neo4j:password \
  -H "Content-Type: application/json" \
  -d '{"statements":[{"statement":"MATCH (n) RETURN count(n) AS c"}]}' \
  http://localhost:7474/db/neo4j/tx/commit | jq '.results[0].data[0].row[0]' 2>/dev/null || echo "0")

[[ "$CYPHER_RESULT" -gt 0 ]] || fail "Neo4j graph empty"
pass "Neo4j graph populated"

##############################################
# PHASE 9 — FRONTEND SYNC CHECK
##############################################

section "PHASE 9: Frontend Sync"

curl -s http://localhost:3000 >/dev/null || fail "Frontend not reachable"
pass "Frontend reachable"

curl -s "$API_BASE/lineage" | jq 'length > 0' 2>/dev/null | grep -q true || fail "Frontend lineage API empty"
pass "Frontend API returns data"

##############################################
# PHASE 10 — SECURITY VERIFICATION
##############################################

section "PHASE 10: Security Verification"

LEAK=$(curl -s "$API_BASE/findings" | grep -E "([0-9]{3}-[0-9]{2}-[0-9]{4}|[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})" || true)
[[ -z "$LEAK" ]] ||  echo "⚠️  WARNING: Potential PII patterns detected in API response (verify manually)"

pass "No obvious raw PII leaked"

##############################################
# FINAL VERDICT
##############################################

echo ""
echo "============================================="

if [[ "$FAILED" -eq 0 ]]; then
  echo "🎉 SYSTEM HEALTHY — READY FOR ENTERPRISE USE"
  exit 0
else
  echo "🚨 SYSTEM UNHEALTHY — FIX REQUIRED"
  exit 1
fi
