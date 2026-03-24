#!/bin/bash
# Seamless Scan Script for ARC-Hawk
# Automatically scans and ingests to backend

set -e

# Configuration
SCANNER_DIR="/Users/prathameshyadav/ARC-Hawk/apps/scanner"
BACKEND_URL="http://localhost:8080/api/v1/scans/ingest"
CONNECTION_FILE="${SCANNER_DIR}/config/connection.yml"
FINGERPRINT_FILE="/Users/prathameshyadav/ARC-Hawk/fingerprint.yml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}================================${NC}"
echo -e "${GREEN}🦅 ARC-Hawk Seamless Scanner${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

# Check backend health
echo -e "${YELLOW}[1/3]${NC} Checking backend status..."
if curl -s -f "http://localhost:8080/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Backend is running${NC}"
else
    echo -e "${RED}❌ Backend is not running!${NC}"
    echo -e "   Start it with: ${YELLOW}cd apps/backend && go run cmd/server/main.go${NC}"
    exit 1
fi

# Get scan type (default: fs)
SCAN_TYPE="${1:-fs}"
echo ""
echo -e "${YELLOW}[2/3]${NC} Running ${SCAN_TYPE} scan..."

# Run scan without --json or --csv to enable auto-ingest
cd "$SCANNER_DIR"
python3 hawk_scanner/main.py "$SCAN_TYPE" \
  --connection "$CONNECTION_FILE" \
  --fingerprint "$FINGERPRINT_FILE" \
  --ingest-url "$BACKEND_URL" \
  --ingest-retry 3 \
  --ingest-timeout 30

echo ""
echo -e "${YELLOW}[3/3]${NC} Viewing results..."
echo -e "${GREEN}✅ Complete! View dashboard at: ${BLUE}http://localhost:3000${NC}"
echo -e "${BLUE}================================${NC}"
