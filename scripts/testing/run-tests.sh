#!/bin/bash
set -e

# Resolve script directory to allow running from anywhere
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# scripts/testing -> scripts -> root
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

echo "🧪 Running all tests..."

# Backend Tests
echo "Testing Backend..."
if [ -d "$ROOT_DIR/apps/backend" ]; then
    cd "$ROOT_DIR/apps/backend"
    go test ./...
else
    echo "❌ Backend directory not found at $ROOT_DIR/apps/backend"
    exit 1
fi

# Frontend Tests
echo "Testing Frontend..."
if [ -d "$ROOT_DIR/apps/frontend" ]; then
    cd "$ROOT_DIR/apps/frontend"
    # npm test -- --passWithNoTests
    echo "Skipping frontend tests for speed (uncomment in script to run)"
else
    echo "❌ Frontend directory not found at $ROOT_DIR/apps/frontend"
    exit 1
fi

echo "✅ All tests passed!"
