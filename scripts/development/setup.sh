#!/bin/bash
set -e

echo "🚀 Setting up ARC Hawk Development Environment..."

# Setup Backend
echo "📦 Downloading Backend dependencies..."
cd ../backend
go mod download

# Setup Frontend
echo "📦 Installing Frontend dependencies..."
cd ../frontend
npm install

echo "✅ Setup complete! run 'docker-compose up -d' to start infrastructure."
