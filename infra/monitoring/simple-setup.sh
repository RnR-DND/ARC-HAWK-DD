#!/bin/bash

# Simple setup script for basic monitoring
set -e

echo "🔍 Setting up basic ARC-Hawk monitoring..."

# Create data directory
mkdir -p data/prometheus

# Start a simple metrics collector
echo "📈 Starting metrics collector..."
go run infra/monitoring/metrics-collector.go &

# Start Prometheus (simplified)
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $PWD:/etc/prometheus \
  prom/prometheus:v2.45.0 \
  - --config.file=/etc/prometheus/prometheus.yml

echo "✅ Prometheus started on http://localhost:9090"
echo "✅ Metrics collector started"
echo "✅ Basic monitoring ready"
echo ""
echo "📊 Access points:"
echo "  Prometheus: http://localhost:9090"
echo "  Metrics: http://localhost:8080/metrics"
echo ""
echo "💡 To stop: docker stop prometheus"