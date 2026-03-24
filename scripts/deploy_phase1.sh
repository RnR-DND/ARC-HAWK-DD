#!/bin/bash

# Phase 1 Deployment Script
# This script sets up the environment for Phase 1: Database-Backed Connections

set -e

echo "🚀 Phase 1: Database-Backed Connections - Deployment"
echo "===================================================="

# Step 1: Generate encryption key (32 bytes for AES-256)
echo ""
echo "📝 Step 1: Generating encryption key..."
ENCRYPTION_KEY=$(openssl rand -base64 32 | head -c 32)
echo "Generated encryption key: $ENCRYPTION_KEY"

# Step 2: Update .env file
echo ""
echo "📝 Step 2: Updating .env file..."
cd apps/backend

if [ ! -f .env ]; then
    echo "Creating new .env file..."
    cat > .env << EOF
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=arc_hawk
DB_SSLMODE=disable

# Neo4j Configuration
NEO4J_URI=bolt://localhost:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=password123

# Temporal Configuration
TEMPORAL_ADDRESS=localhost:7233

# Server Configuration
PORT=8080
GIN_MODE=debug
ALLOWED_ORIGINS=http://localhost:3000

# Encryption Configuration (Phase 1)
ENCRYPTION_KEY=$ENCRYPTION_KEY
EOF
else
    # Append encryption key if not exists
    if ! grep -q "ENCRYPTION_KEY" .env; then
        echo "" >> .env
        echo "# Encryption Configuration (Phase 1)" >> .env
        echo "ENCRYPTION_KEY=$ENCRYPTION_KEY" >> .env
        echo "Added ENCRYPTION_KEY to existing .env file"
    else
        echo "ENCRYPTION_KEY already exists in .env file"
    fi
fi

# Step 3: Run database migration
echo ""
echo "📝 Step 3: Running database migration..."
echo "Make sure PostgreSQL is running on localhost:5432"
read -p "Press Enter to continue with migration..."

# Source .env file
set -a
source .env
set +a

# Run migration
go run cmd/server/main.go &
SERVER_PID=$!
sleep 3
kill $SERVER_PID 2>/dev/null || true

echo ""
echo "✅ Phase 1 deployment complete!"
echo ""
echo "📋 Next steps:"
echo "1. Verify database migration: psql -U postgres -d arc_hawk -c '\\dt connections'"
echo "2. Start backend: cd apps/backend && go run cmd/server/main.go"
echo "3. Test connection creation: curl -X POST http://localhost:8080/api/v1/connections -d '{...}'"
echo ""
echo "🔐 IMPORTANT: Save your encryption key securely!"
echo "ENCRYPTION_KEY=$ENCRYPTION_KEY"
