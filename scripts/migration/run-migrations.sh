#!/bin/bash
set -e

echo "🔄 Running Database Migrations..."

# Use minimal docker-compose for migrations if available, or just exec into running container
docker-compose exec -T postgres psql -U postgres -d arc_platform -f /docker-entrypoint-initdb.d/01-schema.sql

echo "✅ Migrations applied."
