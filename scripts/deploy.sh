#!/bin/bash
# =============================================================================
# ARC-Hawk Deployment Sync Script
# =============================================================================
# This script syncs all changes to the production/staging environment
# Usage: ./scripts/deploy.sh [staging|production]
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENVIRONMENT=${1:-staging}

# Default tags based on environment
BACKEND_TAG="arc-hawk-backend:latest"
FRONTEND_TAG="arc-hawk-frontend:latest"
SCANNER_TAG="arc-hawk-scanner:latest"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  ARC-Hawk Deployment Sync${NC}"
echo -e "${GREEN}  Environment: $ENVIRONMENT${NC}"
echo -e "${GREEN}========================================${NC}"

# =============================================================================
# PRE-FLIGHT CHECKS
# =============================================================================
echo -e "\n${YELLOW}[1/6] Running pre-flight checks...${NC}"

# Check if docker is running
if ! docker info &> /dev/null; then
    echo -e "${RED}ERROR: Docker is not running${NC}"
    exit 1
fi
echo "  ✓ Docker is running"

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}ERROR: docker-compose is not available${NC}"
    exit 1
fi
echo "  ✓ Docker Compose is available"

# Check if project directory exists
if [ ! -d "$PROJECT_ROOT" ]; then
    echo -e "${RED}ERROR: Project directory not found: $PROJECT_ROOT${NC}"
    exit 1
fi
echo "  ✓ Project directory exists"

# =============================================================================
# BACKUP DATABASE (Production Only)
# =============================================================================
if [ "$ENVIRONMENT" = "production" ]; then
    echo -e "\n${YELLOW}[2/6] Creating database backup...${NC}"
    
    BACKUP_DIR="$PROJECT_ROOT/backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    docker exec arc-platform-db pg_dump -U postgres arc_platform > "$BACKUP_DIR/arc_platform.sql"
    
    echo "  ✓ Database backup created: $BACKUP_DIR/arc_platform.sql"
fi

# =============================================================================
# PULL LATEST IMAGES
# =============================================================================
echo -e "\n${YELLOW}[3/6] Pulling latest images...${NC}"

cd "$PROJECT_ROOT"

# Pull all service images
docker-compose pull postgres
echo "  ✓ PostgreSQL pulled"

docker-compose pull neo4j
echo "  ✓ Neo4j pulled"

docker-compose pull presidio-analyzer
echo "  ✓ Presidio pulled"

docker-compose pull temporal
echo "  ✓ Temporal pulled"

docker-compose pull temporal-ui
echo "  ✓ Temporal UI pulled"

# Rebuild and pull application images
docker-compose build --pull backend
echo "  ✓ Backend rebuilt"

docker-compose build --pull frontend
echo "  ✓ Frontend rebuilt"

docker-compose build --pull scanner
echo "  ✓ Scanner rebuilt"

# =============================================================================
# STOP EXISTING SERVICES
# =============================================================================
echo -e "\n${YELLOW}[4/6] Stopping existing services...${NC}"

docker-compose down --remove-orphans
echo "  ✓ All services stopped"

# =============================================================================
# START SERVICES
# =============================================================================
echo -e "\n${YELLOW}[5/6] Starting services...${NC}"

docker-compose up -d

# Wait for services to be healthy
echo "  Waiting for services to be healthy..."

# PostgreSQL
echo -n "    - PostgreSQL: "
timeout 60 bash -c 'until docker exec arc-platform-db pg_isready -U postgres &>/dev/null; do printf "."; sleep 2; done'
echo "ready"

# Neo4j
echo -n "    - Neo4j: "
timeout 60 bash -c 'until docker exec arc-platform-neo4j cypher-shell -u neo4j -p password123 "RETURN 1" &>/dev/null; do printf "."; sleep 2; done'
echo "ready"

# Presidio
echo -n "    - Presidio: "
timeout 60 bash -c 'until curl -sf http://localhost:5001/health &>/dev/null; do printf "."; sleep 2; done'
echo "ready"

# Temporal
echo -n "    - Temporal: "
timeout 60 bash -c 'until curl -sf http://localhost:7233 &>/dev/null; do printf "."; sleep 2; done'
echo "ready"

# Backend
echo -n "    - Backend: "
timeout 60 bash -c 'until curl -sf http://localhost:8080/health &>/dev/null; do printf "."; sleep 2; done'
echo "ready"

# Frontend
echo -n "    - Frontend: "
timeout 60 bash -c 'until curl -sf http://localhost:3000 &>/dev/null; do printf "."; sleep 2; done'
echo "ready"

# =============================================================================
# VERIFICATION
# =============================================================================
echo -e "\n${YELLOW}[6/6] Verifying deployment...${NC}"

HEALTH_CHECKS=0
TOTAL_CHECKS=6

# Check PostgreSQL
if docker exec arc-platform-db pg_isready -U postgres &>/dev/null; then
    echo "  ✓ PostgreSQL healthy"
    ((HEALTH_CHECKS++))
fi

# Check Neo4j
if docker exec arc-platform-neo4j cypher-shell -u neo4j -p password123 "RETURN 1" &>/dev/null; then
    echo "  ✓ Neo4j healthy"
    ((HEALTH_CHECKS++))
fi

# Check Presidio
if curl -sf http://localhost:5001/health &>/dev/null; then
    echo "  ✓ Presidio healthy"
    ((HEALTH_CHECKS++))
fi

# Check Temporal
if curl -sf http://localhost:7233 &>/dev/null; then
    echo "  ✓ Temporal healthy"
    ((HEALTH_CHECKS++))
fi

# Check Backend
if curl -sf http://localhost:8080/api/v1/health &>/dev/null; then
    echo "  ✓ Backend healthy"
    ((HEALTH_CHECKS++))
fi

# Check Frontend
if curl -sf http://localhost:3000 &>/dev/null; then
    echo "  ✓ Frontend healthy"
    ((HEALTH_CHECKS++))
fi

# =============================================================================
# SUMMARY
# =============================================================================
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}  Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "  Services: $HEALTH_CHECKS/$TOTAL_CHECKS healthy"
echo ""
echo "  URLs:"
echo "    - Frontend:  http://localhost:3000"
echo "    - Backend:   http://localhost:8080"
echo "    - Temporal:  http://localhost:7233"
echo "    - Temporal UI: http://localhost:8088"
echo "    - Neo4j:     http://localhost:7474"
echo ""
echo "  Status: $ENVIRONMENT deployment synced successfully!"
echo ""

if [ $HEALTH_CHECKS -ne $TOTAL_CHECKS ]; then
    echo -e "${YELLOW}WARNING: Some services may not be healthy. Check logs with:${NC}"
    echo "  docker-compose logs -f"
    exit 1
fi
