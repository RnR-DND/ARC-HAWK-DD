#!/bin/bash
# verify-scanner.sh - Verify Scanner to Backend integration
# Usage: ./verify-scanner.sh [backend_host] [backend_port]

set -e

# Configuration
BACKEND_HOST="${1:-localhost}"
BACKEND_PORT="${2:-8080}"
BACKEND_URL="http://$BACKEND_HOST:$BACKEND_PORT/api/v1"

echo "============================================"
echo "Scanner to Backend Integration Verification"
echo "============================================"
echo "Backend URL: $BACKEND_URL"
echo "============================================"

# Test 1: Verify backend is running
echo ""
echo "🔄 Step 1: Verify backend is accessible..."

HEALTH_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "$BACKEND_URL/health" 2>&1 || echo "000")

if [ "$HEALTH_RESPONSE" = "200" ]; then
    echo "✅ Backend is accessible"
else
    echo "❌ Backend is not accessible (HTTP $HEALTH_RESPONSE)"
    echo "   Please start the backend first: cd apps/backend && go run cmd/server/main.go"
    exit 1
fi

# Test 2: Create test finding payload
echo ""
echo "🔄 Step 2: Create test PII finding..."

TEST_FINDING=$(cat <<EOF
{
  "fs": [
    {
      "host": "test-host",
      "file_path": "/tmp/test-file.txt",
      "pattern_name": "Email",
      "matches": ["test@example.com"],
      "sample_text": "Contact us at test@example.com",
      "profile": "test_profile",
      "data_source": "fs",
      "severity": "Low",
      "file_data": {}
    }
  ]
}
EOF
)

# Test 3: Send test finding to backend
echo ""
echo "🔄 Step 3: Ingest test finding..."

START_TIME=$(date +%s%N)

INGEST_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BACKEND_URL/scans/ingest-verified" \
    -H "Content-Type: application/json" \
    -d "$TEST_FINDING" 2>&1)

END_TIME=$(date +%s%N)
LATENCY=$(( (END_TIME - START_TIME) / 1000000 ))

HTTP_CODE=$(echo "$INGEST_RESPONSE" | tail -n1)
BODY=$(echo "$INGEST_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo "✅ Test finding ingestion SUCCESSFUL"
    echo "   HTTP Status: $HTTP_CODE"
    echo "   Latency: ${LATENCY}ms"
    echo "   Response: $BODY" | head -c 500
elif [ "$HTTP_CODE" = "400" ]; then
    echo "⚠️  Backend rejected the finding (validation error)"
    echo "   This may be expected if strict validation is enabled"
    echo "   HTTP Status: $HTTP_CODE"
    echo "   Response: $BODY" | head -c 500
else
    echo "❌ Finding ingestion FAILED"
    echo "   HTTP Status: $HTTP_CODE"
    echo "   Response: $BODY"
fi

# Test 4: Verify Go scanner health
echo ""
echo "🔄 Step 4: Verify Go scanner health..."

GO_SCANNER_URL="http://localhost:8001"
SCANNER_HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "$GO_SCANNER_URL/health" 2>&1 || echo "000")

if [ "$SCANNER_HEALTH" = "200" ]; then
    echo "✅ Go scanner is healthy"
else
    echo "⚠️  Go scanner not reachable at $GO_SCANNER_URL (HTTP $SCANNER_HEALTH)"
    echo "   Start with: docker-compose up -d go-scanner"
fi

echo ""
echo "============================================"
echo "Scanner-Backend Integration Verification"
echo "============================================"
echo ""
echo "📋 Summary:"
echo "   Backend: $([ "$HEALTH_RESPONSE" = "200" ] && echo "✅ Running" || echo "❌ Not running")"
echo "   Ingestion: $([ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ] && echo "✅ Working" || echo "⚠️ Check response")"
echo "   Go Scanner: $([ "$SCANNER_HEALTH" = "200" ] && echo "✅ Running" || echo "⚠️ Not running")"
echo ""
echo "🔧 To run a full scanner test:"
echo "   1. Ensure Docker is running: docker-compose up -d"
echo "   2. Start backend: cd apps/backend && go run cmd/server/main.go"
echo "   3. Start scanner: docker-compose up -d go-scanner"
