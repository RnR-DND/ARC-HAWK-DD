# 📋 DEPLOYMENT_RUNBOOK.md - ARC-Hawk Deployment Guide

**Version:** 2.2.0
**Date:** 2026-01-22
**Status:** Production Ready with CI/CD

---

## 🎯 Overview

This runbook documents deployment procedures for ARC-Hawk PII Discovery Platform. It covers local development, staging, and production deployments.

## 📁 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    ARC-Hawk Platform                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌──────────┐    ┌──────────┐    ┌──────────────────────────┐ │
│   │ Scanner  │───▶│ Backend  │───▶│ PostgreSQL + Neo4j       │ │
│   │ (Python)│    │  (Go)    │    │ (Data Storage & Graph)   │ │
│   └──────────┘    └──────────┘    └──────────────────────────┘ │
│         │               │                     │                │
│         │               │                     │                │
│         ▼               ▼                     ▼                │
│   ┌─────────────────────────────────────────────────────────┐ │
│   │              Next.js Dashboard (Frontend)               │ │
│   └─────────────────────────────────────────────────────────┘ │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  Infrastructure: Docker Compose (local) / Kubernetes (prod)    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start (Local Development)

### Prerequisites

```bash
✓ Docker & Docker Compose
✓ Go 1.24+
✓ Node.js 18+
✓ Python 3.9+
```

### Step 1: Start Infrastructure

```bash
cd /Users/prathameshyadav/ARC-Hawk

# Start all services
docker-compose up -d

# Verify services
docker ps --format "table {{.Names}}\t{{.Status}}"
```

**Expected Output:**
```
NAMES                   STATUS
arc-platform-neo4j      Up (healthy)
arc-platform-presidio   Up (healthy)
arc-platform-db         Up (healthy)
```

### Step 2: Start Backend

```bash
cd apps/backend

# Clean build
go mod tidy
go build -o server ./cmd/server

# Run
NEO4J_URI=bolt://localhost:7687 ./server
```

**Backend runs on:** `http://localhost:8081/api/v1`

### Step 3: Start Frontend

```bash
cd apps/frontend

# Install dependencies (first time only)
npm install

# Development mode
npm run dev

# Production build
npm run build
npm run start
```

**Frontend runs on:** `http://localhost:3000`

### Step 4: Run Scanner

```bash
cd apps/scanner

# Install dependencies (first time only)
pip install -r requirements.txt

# Run filesystem scan
python -m hawk_scanner.main fs \
  --connection config/connection.yml \
  --json output.json
```

---

## 🐳 Docker Deployment

### Development (docker-compose)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Stop and remove volumes (data loss)
docker-compose down -v
```

### Service-Specific Commands

```bash
# PostgreSQL only
docker-compose up -d postgres

# Neo4j only
docker-compose up -d neo4j

# Backend only
docker-compose up -d backend

# Frontend only
docker-compose up -d frontend
```

### Temporal Configuration Fix

If you encounter Temporal startup issues, ensure:

1. PostgreSQL is healthy before Temporal starts
2. Use correct environment variables:
   ```yaml
   environment:
     - DB=postgresql
     - DB_PORT=5432
     - POSTGRES_USER=postgres
     - POSTGRES_PWD=postgres
     - POSTGRES_SEEDS=postgres  # Not postgres:5432
   ```
3. Dynamic config is mounted:
   ```yaml
   volumes:
     - ./temporal/config:/etc/temporal/config:ro
   ```

---

## ☸️ Kubernetes Deployment

### Prerequisites

```bash
✓ kubectl configured
✓ Access to cluster
✓ Helm 3.x (optional)
```

### Deploy to Cluster

```bash
# Apply Kubernetes configurations
kubectl apply -f infra/k8s/deployment.yaml

# Check deployments
kubectl get deployments -n arc-platform

# Check services
kubectl get svc -n arc-platform

# Check pods
kubectl get pods -n arc-platform
```

### Namespace Setup

```bash
# Create namespace if not exists
kubectl create namespace arc-platform

# Verify
kubectl get namespace arc-platform
```

### ConfigMap Management

```bash
# View config
kubectl get configmap arc-config -n arc-platform -o yaml

# Update config (requires pod restart)
kubectl apply -f infra/k8s/deployment.yaml
```

---

## 🔧 Configuration Management

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_HOST` | PostgreSQL host | localhost | Yes |
| `DATABASE_PORT` | PostgreSQL port | 5432 | No |
| `DATABASE_USER` | PostgreSQL user | postgres | Yes |
| `DATABASE_PASSWORD` | PostgreSQL password | postgres | Yes |
| `DATABASE_NAME` | Database name | arc_platform | Yes |
| `NEO4J_URI` | Neo4j connection URI | bolt://localhost:7687 | Yes |
| `NEO4J_USERNAME` | Neo4j user | neo4j | Yes |
| `NEO4J_PASSWORD` | Neo4j password | password123 | Yes |
| `GIN_MODE` | Gin mode (debug/release) | debug | No |
| `NEXT_PUBLIC_API_URL` | Frontend API URL | http://localhost:8080/api/v1 | Yes |

### Connection Profiles (Scanner)

**Location:** `apps/scanner/config/connection.yml.sample`

Each data source requires UNIQUE connection parameters:

```yaml
# PostgreSQL Example
postgresql:
  my_profile:
    host: "pg-host"
    port: 5432
    user: "user"
    password: "pass"
    database: "db"
    tables: ["table1", "table2"]

# AWS S3 Example  
s3:
  my_profile:
    access_key: "key"
    secret_key: "secret"
    bucket_name: "bucket"
```

---

## 🔄 CI/CD Pipeline

### GitHub Actions Workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci-cd.yml` | Push/PR to main | Full CI/CD pipeline with lint, test, build, push, and deploy |
| `pypi.yml` | Tag release | Publish scanner to PyPI |
| `regression.yml` | Manual | Run regression tests |

### CI/CD Pipeline Stages

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Code      │───▶│   Lint      │───▶│   Build     │───▶│   Test      │───▶│   Push      │
│   Push      │    │   & Scan    │    │   Docker    │    │   & Verify  │    │   Images    │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                                        │
                                                                        ▼
                                                              ┌─────────────────┐
                                                              │    Deploy       │
                                                              │  (Manual Trigger)│
                                                              └─────────────────┘
```

### Automated Deployment (GitHub Actions)

1. **Push to main branch** triggers:
   - Linting (Go, Node.js, Python)
   - Unit tests
   - Docker builds
   - Integration tests
   - Docker image push to registry

2. **Manual trigger** (`workflow_dispatch`) deploys to:
   - Staging environment
   - Production environment (requires confirmation)

### Local Deployment Commands

```bash
# Using Makefile
make start          # Start all services
make stop           # Stop all services
make restart        # Restart services
make build          # Build images
make rebuild        # Rebuild with no cache
make status         # Check health
make test           # Run all tests
make deploy         # Deploy to current env
make deploy-staging # Deploy to staging
make deploy-production  # Deploy to production

# Using deploy script directly
./scripts/deploy.sh              # Deploy to default (staging)
./scripts/deploy.sh staging      # Deploy to staging
./scripts/deploy.sh production   # Deploy to production (with confirmation)
```

### GitHub Secrets Required

For CI/CD to work, configure these secrets in GitHub repository:

| Secret | Description |
|--------|-------------|
| `DOCKERHUB_USERNAME` | Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub access token |
| `SERVER_HOST` | Production server IP/hostname |
| `SERVER_USER` | SSH user |
| `SERVER_SSH_KEY` | SSH private key |

### Running CI Locally

```bash
# Backend build
cd apps/backend && go build ./...

# Frontend build
cd apps/frontend && npm run build

# Scanner build
cd apps/scanner && python setup.py sdist

# Run all tests
make test

---

## 🧪 Testing Procedures

### Unit Tests

```bash
# Backend tests
cd apps/backend && go test ./...

# Scanner tests
cd apps/scanner && python -m pytest tests/

# Frontend tests
cd apps/frontend && npm test
```

### Integration Tests

```bash
# Run all integration tests
./scripts/testing/run-tests.sh

# Or individually
cd apps/backend && go test ./modules/scanning -v
cd apps/scanner && python -m pytest tests/test_validation.py -v
```

### End-to-End Tests

```bash
# Trigger from GitHub Actions
# Or run locally with Cypress
cd apps/frontend && npx cypress run
```

---

## 📊 Monitoring & Observability

### Health Checks

```bash
# Backend health
curl http://localhost:8081/health

# PostgreSQL health
docker exec arc-platform-db pg_isready -U postgres

# Neo4j health
curl http://localhost:7474/
```

### Logs

```bash
# Backend logs
docker logs arc-platform-backend -f

# All service logs
docker-compose logs -f
```

### Metrics

| Metric | Endpoint | Description |
|--------|----------|-------------|
| API Health | `/health` | Service status |
| Classification | `/classification/summary` | PII counts |
| Findings | `/findings` | Paginated findings |

---

## 🚨 Troubleshooting

### Common Issues

#### 1. PostgreSQL Connection Failed

```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Check logs
docker logs arc-platform-db

# Restart PostgreSQL
docker-compose restart postgres
```

#### 2. Neo4j Connection Refused

```bash
# Check Neo4j status
docker ps | grep neo4j

# Check logs
docker logs arc-platform-neo4j

# Restart Neo4j
docker restart arc-platform-neo4j
```

#### 3. Backend Won't Start

```bash
# Check for port conflicts
lsof -i :8080

# Check environment variables
echo $NEO4J_URI

# Check logs
docker logs arc-platform-backend
```

#### 4. Frontend Build Failed

```bash
# Clear cache
cd apps/frontend && rm -rf .next node_modules

# Reinstall
npm install

# Build again
npm run build
```

---

## 🔒 Security Checklist

Before production deployment:

- [ ] Change default passwords
- [ ] Enable SSL/TLS
- [ ] Set up authentication (JWT)
- [ ] Configure rate limiting
- [ ] Enable audit logging
- [ ] Set up secrets management
- [ ] Configure CORS properly
- [ ] Enable database encryption
- [ ] Set up network policies

---

## 📋 Rollback Procedures

### Docker Compose Rollback

```bash
# View previous versions
docker-compose down
git checkout <previous-version>
docker-compose up -d
```

### Kubernetes Rollback

```bash
# View deployment history
kubectl rollout history deployment/arc-backend -n arc-platform

# Rollback to previous version
kubectl rollout undo deployment/arc-backend -n arc-platform

# Verify rollback
kubectl rollout status deployment/arc-backend -n arc-platform
```

---

## 📞 Emergency Contacts

| Role | Contact |
|------|---------|
| Platform Owner | See README.md |
| DevOps Team | Internal Slack #devops |
| Security Team | Internal Slack #security |

---

## 📚 Related Documentation

- `gemini.md` - Project constitution
- `task_plan.md` - Project blueprint
- `AGENTS.md` - AI agent development guide
- `docs/architecture/ARCHITECTURE.md` - System architecture
- `docs/deployment/guide.md` - Detailed deployment guide
- `docs/FAILURE_MODES.md` - Troubleshooting guide

---

*This runbook should be updated whenever deployment procedures change.*
