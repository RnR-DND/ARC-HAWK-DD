#!/bin/bash
# verify-postgres.sh - Verify PostgreSQL connectivity
# Usage: ./verify-postgres.sh [host] [port] [user] [password] [database]

set -e

# Configuration
HOST="${1:-localhost}"
PORT="${2:-5432}"
USER="${3:-postgres}"
PASSWORD="${4:-postgres}"
DATABASE="${5:-arc_platform}"

echo "============================================"
echo "PostgreSQL Connectivity Verification"
echo "============================================"
echo "Host:     $HOST"
echo "Port:     $PORT"
echo "User:     $USER"
echo "Database: $DATABASE"
echo "============================================"

# Check if psql is available
if ! command -v psql &> /dev/null; then
    if docker ps | grep -q "postgres"; then
        echo "ℹ️  psql not found locally, but found postgres docker container. Using docker exec."
        PSQL_CMD="docker exec -i arc-platform-db psql"
    else
        echo "⚠️  psql command not found. Installing..."
        echo "   Run: brew install postgresql (macOS) or apt-get install postgresql-client (Linux)"
        exit 1
    fi
else
    PSQL_CMD="psql"
fi

# Test connection
echo ""
echo "🔄 Testing PostgreSQL connection..."

START_TIME=$(date +%s%N)

if [[ "$PSQL_CMD" == *"docker"* ]]; then
    CONN_RESULT=$(PGPASSWORD="$PASSWORD" $PSQL_CMD -U "$USER" -d "$DATABASE" -c "SELECT 1 as test;" 2>&1 || true)
else
    CONN_RESULT=$(PGPASSWORD="$PASSWORD" $PSQL_CMD -h "$HOST" -p "$PORT" -U "$USER" -d "$DATABASE" -c "SELECT 1 as test;" 2>&1 || true)
fi

END_TIME=$(date +%s%N)
LATENCY=$(( (END_TIME - START_TIME) / 1000000 ))

if echo "$CONN_RESULT" | grep -q "1 row"; then
    echo "✅ PostgreSQL connection SUCCESSFUL"
    echo "   Latency: ${LATENCY}ms"

    # Get version
    if [[ "$PSQL_CMD" == *"docker"* ]]; then
        VERSION=$(PGPASSWORD="$PASSWORD" $PSQL_CMD -U "$USER" -d "$DATABASE" -c "SELECT version();" 2>&1 | grep "PostgreSQL" | head -1)
        SIZE=$(PGPASSWORD="$PASSWORD" $PSQL_CMD -U "$USER" -d "$DATABASE" -c "SELECT pg_database_size('$DATABASE') as size;" 2>&1 | grep -E "[0-9]+" | tail -1)
    else
        VERSION=$(PGPASSWORD="$PASSWORD" $PSQL_CMD -h "$HOST" -p "$PORT" -U "$USER" -d "$DATABASE" -c "SELECT version();" 2>&1 | grep "PostgreSQL" | head -1)
        SIZE=$(PGPASSWORD="$PASSWORD" $PSQL_CMD -h "$HOST" -p "$PORT" -U "$USER" -d "$DATABASE" -c "SELECT pg_database_size('$DATABASE') as size;" 2>&1 | grep -E "[0-9]+" | tail -1)
    fi
    echo "   Version: $VERSION"
    echo "   Database Size: $(numfmt --to=iec $SIZE 2>/dev/null || echo "$SIZE bytes")"

    echo ""
    echo "✅ PostgreSQL verification PASSED"
    exit 0
else
    echo "❌ PostgreSQL connection FAILED"
    echo "   Error: $CONN_RESULT"
    echo ""
    echo "🔧 Troubleshooting:"
    echo "   1. Ensure Docker is running: docker-compose up -d postgres"
    echo "   2. Check credentials in .env file"
    echo "   3. Verify port $PORT is accessible"
    exit 1
fi
