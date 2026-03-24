# Deployment Guide

Complete guide for deploying ARC-Hawk to production environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Deployment Options](#deployment-options)
- [Docker Compose Deployment](#docker-compose-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Cloud Deployment](#cloud-deployment)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### System Requirements

#### Minimum Requirements
- **CPU**: 4 cores
- **RAM**: 8GB
- **Disk**: 50GB SSD
- **Network**: 100Mbps

#### Recommended for Production
- **CPU**: 8+ cores
- **RAM**: 16GB+
- **Disk**: 100GB+ SSD
- **Network**: 1Gbps
- **PostgreSQL**: Dedicated instance
- **Neo4j**: Dedicated instance

### Software Requirements

- **Docker** 24.0+ & Docker Compose 2.20+
- **Git**
- **OpenSSL** (for certificate generation)

### Security Prerequisites

> **⚠️ Important**: Before deploying to production:
> 1. Change all default passwords
> 2. Generate strong secrets
> 3. Set up TLS/SSL certificates
> 4. Configure firewall rules
> 5. Review [Security Policy](../../SECURITY.md)

---

## Deployment Options

### Option 1: Docker Compose (Recommended for Small-Medium Deployments)

Best for:
- Single server deployments
- Development/staging environments
- Teams getting started

### Option 2: Kubernetes (Recommended for Large Deployments)

Best for:
- High availability requirements
- Auto-scaling needs
- Multi-region deployments

### Option 3: Cloud-Managed Services

Best for:
- Enterprise deployments
- Reduced operational overhead
- Using managed databases

---

## Docker Compose Deployment

### Step 1: Prepare Server

```bash
# Update system
sudo apt update && sudo apt upgrade -y  # Ubuntu/Debian
sudo yum update -y  # CentOS/RHEL

# Install Docker
# Follow: https://docs.docker.com/engine/install/

# Install Docker Compose
# Follow: https://docs.docker.com/compose/install/

# Create deployment directory
mkdir -p /opt/arc-hawk
cd /opt/arc-hawk
```

### Step 2: Download Application

```bash
# Clone repository
git clone https://github.com/your-org/arc-hawk.git
cd arc-hawk

# Checkout specific version (recommended)
git checkout v2.1.0
```

### Step 3: Generate Secrets

```bash
# Create secrets directory
mkdir -p secrets

# Generate PostgreSQL password
openssl rand -base64 32 > secrets/postgres_password

# Generate Neo4j password
openssl rand -base64 32 > secrets/neo4j_password

# Generate JWT secret (for future auth)
openssl rand -base64 64 > secrets/jwt_secret

# Set permissions
chmod 600 secrets/*
```

### Step 4: Configure Environment

Create `production.env`:

```bash
# Server
ENV=production
PORT=8080

# Database - PostgreSQL
DB_HOST=postgres
DB_PORT=5432
DB_USER=arc_hawk
DB_PASSWORD_FILE=/run/secrets/postgres_password
DB_NAME=arc_hawk_production
DB_SSL_MODE=require

# Database - Neo4j
NEO4J_URI=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD_FILE=/run/secrets/neo4j_password

# Temporal
TEMPORAL_ADDRESS=temporal:7233
TEMPORAL_NAMESPACE=arc-hawk

# Scanner
MAX_SCAN_WORKERS=10
SCAN_TIMEOUT_MINUTES=60

# Security (Future)
# JWT_SECRET_FILE=/run/secrets/jwt_secret
# ENABLE_AUTH=false

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Step 5: Configure Docker Compose

Create `docker-compose.production.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: arc_hawk
      POSTGRES_PASSWORD_FILE: /run/secrets/postgres_password
      POSTGRES_DB: arc_hawk_production
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./secrets/postgres_password:/run/secrets/postgres_password:ro
    ports:
      - "5432:5432"
    networks:
      - arc-hawk
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U arc_hawk"]
      interval: 10s
      timeout: 5s
      retries: 5

  neo4j:
    image: neo4j:5.15-community
    environment:
      NEO4J_AUTH_FILE: /run/secrets/neo4j_auth
      NEO4J_dbms_memory_heap_initial__size: 512m
      NEO4J_dbms_memory_heap_max__size: 2G
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
      - ./secrets/neo4j_password:/run/secrets/neo4j_auth:ro
    ports:
      - "7474:7474"
      - "7687:7687"
    networks:
      - arc-hawk
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "cypher-shell", "-u", "neo4j", "-p", "password", "RETURN 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  temporal:
    image: temporalio/auto-setup:1.22
    environment:
      - DB=postgres12
      - DB_PORT=5432
      - POSTGRES_USER=arc_hawk
      - POSTGRES_PWD_FILE=/run/secrets/postgres_password
      - POSTGRES_SEEDS=postgres
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development.yaml
    volumes:
      - ./secrets/postgres_password:/run/secrets/postgres_password:ro
    ports:
      - "7233:7233"
      - "8088:8088"
    networks:
      - arc-hawk
    restart: unless-stopped
    depends_on:
      - postgres

  backend:
    build:
      context: ./apps/backend
      dockerfile: Dockerfile
    env_file:
      - production.env
    volumes:
      - ./secrets:/run/secrets:ro
    ports:
      - "8080:8080"
    networks:
      - arc-hawk
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      neo4j:
        condition: service_healthy
      temporal:
        condition: service_started
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  frontend:
    build:
      context: ./apps/frontend
      dockerfile: Dockerfile
    environment:
      - NEXT_PUBLIC_API_URL=http://backend:8080/api/v1
      - NEXT_PUBLIC_WS_URL=ws://backend:8080/ws
    ports:
      - "3000:3000"
    networks:
      - arc-hawk
    restart: unless-stopped
    depends_on:
      - backend

  scanner:
    build:
      context: ./apps/scanner
      dockerfile: Dockerfile
    environment:
      - API_URL=http://backend:8080
    networks:
      - arc-hawk
    restart: unless-stopped
    depends_on:
      - backend

volumes:
  postgres_data:
  neo4j_data:
  neo4j_logs:

networks:
  arc-hawk:
    driver: bridge
```

### Step 6: Deploy

```bash
# Deploy with production config
docker-compose -f docker-compose.production.yml up -d

# Check status
docker-compose -f docker-compose.production.yml ps

# View logs
docker-compose -f docker-compose.production.yml logs -f

# Check specific service
docker-compose -f docker-compose.production.yml logs -f backend
```

### Step 7: Verify Deployment

```bash
# Test backend
curl http://localhost:8080/health

# Test frontend
curl http://localhost:3000

# Test Neo4j
curl http://localhost:7474

# Test Temporal
curl http://localhost:8088
```

### Step 8: Set Up Reverse Proxy (Nginx)

Create `/etc/nginx/sites-available/arc-hawk`:

```nginx
upstream backend {
    server localhost:8080;
}

upstream frontend {
    server localhost:3000;
}

server {
    listen 80;
    server_name arc-hawk.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name arc-hawk.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Frontend
    location / {
        proxy_pass http://frontend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Backend API
    location /api/ {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket
    location /ws {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Neo4j Browser (optional, restrict access)
    location /neo4j/ {
        auth_basic "Neo4j Browser";
        auth_basic_user_file /etc/nginx/.htpasswd;
        proxy_pass http://localhost:7474/;
    }
}
```

Enable the site:

```bash
sudo ln -s /etc/nginx/sites-available/arc-hawk /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

---

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (1.25+)
- kubectl configured
- Helm 3.x (optional but recommended)

### Step 1: Create Namespace

```bash
kubectl create namespace arc-hawk
```

### Step 2: Create Secrets

```bash
# Create secrets
kubectl create secret generic postgres-secret \
  --from-literal=password=$(openssl rand -base64 32) \
  -n arc-hawk

kubectl create secret generic neo4j-secret \
  --from-literal=password=$(openssl rand -base64 32) \
  -n arc-hawk

kubectl create secret generic jwt-secret \
  --from-literal=secret=$(openssl rand -base64 64) \
  -n arc-hawk
```

### Step 3: Deploy PostgreSQL

```yaml
# postgres-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: arc-hawk
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_USER
          value: arc_hawk
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        - name: POSTGRES_DB
          value: arc_hawk_production
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: arc-hawk
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: arc-hawk
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
```

Deploy:

```bash
kubectl apply -f postgres-deployment.yaml
```

### Step 4: Deploy Neo4j

```yaml
# neo4j-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: neo4j
  namespace: arc-hawk
spec:
  replicas: 1
  selector:
    matchLabels:
      app: neo4j
  template:
    metadata:
      labels:
        app: neo4j
    spec:
      containers:
      - name: neo4j
        image: neo4j:5.15-community
        env:
        - name: NEO4J_AUTH
          value: neo4j/password-from-secret
        - name: NEO4J_dbms_memory_heap_initial__size
          value: 512m
        - name: NEO4J_dbms_memory_heap_max__size
          value: 2G
        ports:
        - containerPort: 7474
        - containerPort: 7687
        volumeMounts:
        - name: neo4j-storage
          mountPath: /data
      volumes:
      - name: neo4j-storage
        persistentVolumeClaim:
          claimName: neo4j-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: neo4j
  namespace: arc-hawk
spec:
  selector:
    app: neo4j
  ports:
  - name: http
    port: 7474
    targetPort: 7474
  - name: bolt
    port: 7687
    targetPort: 7687
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: neo4j-pvc
  namespace: arc-hawk
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
```

Deploy:

```bash
kubectl apply -f neo4j-deployment.yaml
```

### Step 5: Deploy Backend

```yaml
# backend-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
  namespace: arc-hawk
spec:
  replicas: 2
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
      - name: backend
        image: your-registry/arc-hawk-backend:v2.1.0
        env:
        - name: ENV
          value: production
        - name: DB_HOST
          value: postgres
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        - name: NEO4J_PASSWORD
          valueFrom:
            secretKeyRef:
              name: neo4j-secret
              key: password
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: backend
  namespace: arc-hawk
spec:
  selector:
    app: backend
  ports:
  - port: 8080
    targetPort: 8080
```

Deploy:

```bash
kubectl apply -f backend-deployment.yaml
```

### Step 6: Deploy Frontend

```yaml
# frontend-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: arc-hawk
spec:
  replicas: 2
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: frontend
        image: your-registry/arc-hawk-frontend:v2.1.0
        env:
        - name: NEXT_PUBLIC_API_URL
          value: http://backend:8080/api/v1
        - name: NEXT_PUBLIC_WS_URL
          value: ws://backend:8080/ws
        ports:
        - containerPort: 3000
---
apiVersion: v1
kind: Service
metadata:
  name: frontend
  namespace: arc-hawk
spec:
  selector:
    app: frontend
  ports:
  - port: 3000
    targetPort: 3000
```

Deploy:

```bash
kubectl apply -f frontend-deployment.yaml
```

### Step 7: Configure Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: arc-hawk-ingress
  namespace: arc-hawk
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - arc-hawk.yourdomain.com
    secretName: arc-hawk-tls
  rules:
  - host: arc-hawk.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend
            port:
              number: 3000
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: backend
            port:
              number: 8080
      - path: /ws
        pathType: Prefix
        backend:
          service:
            name: backend
            port:
              number: 8080
```

Deploy:

```bash
kubectl apply -f ingress.yaml
```

---

## Cloud Deployment

### AWS Deployment

#### Using ECS (Elastic Container Service)

```bash
# Install AWS CLI and ECS CLI
# Follow: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ECS_CLI_installation.html

# Configure AWS
aws configure

# Create ECS cluster
ecs-cli up --cluster-name arc-hawk --keypair your-keypair --capability-iam --size 2 --instance-type t3.large

# Deploy services
ecs-cli compose -f docker-compose.production.yml up
```

#### Using EKS (Elastic Kubernetes Service)

```bash
# Create EKS cluster
eksctl create cluster \
  --name arc-hawk \
  --region us-west-2 \
  --node-type t3.large \
  --nodes 3

# Deploy using kubectl
kubectl apply -k k8s/
```

### GCP Deployment

#### Using GKE (Google Kubernetes Engine)

```bash
# Create GKE cluster
gcloud container clusters create arc-hawk \
  --zone us-central1-a \
  --num-nodes 3 \
  --machine-type n1-standard-4

# Get credentials
gcloud container clusters get-credentials arc-hawk --zone us-central1-a

# Deploy
kubectl apply -f k8s/
```

### Azure Deployment

#### Using AKS (Azure Kubernetes Service)

```bash
# Create AKS cluster
az aks create \
  --resource-group myResourceGroup \
  --name arc-hawk \
  --node-count 3 \
  --enable-addons monitoring \
  --generate-ssh-keys

# Get credentials
az aks get-credentials --resource-group myResourceGroup --name arc-hawk

# Deploy
kubectl apply -f k8s/
```

---

## Configuration

### Environment Variables

See [Backend README](../../apps/backend/README.md#environment-variables) for complete list.

### Database Configuration

#### PostgreSQL Tuning

For production workloads, tune PostgreSQL:

```sql
-- Connect to PostgreSQL
psql -U arc_hawk -d arc_hawk_production

-- Optimize for 16GB RAM
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET work_mem = '256MB';

-- Restart PostgreSQL
```

#### Neo4j Tuning

```properties
# conf/neo4j.conf
server.memory.heap.initial_size=2G
server.memory.heap.max_size=4G
server.memory.pagecache.size=4G
dbms.memory.transaction.max_size=1G
```

### Backup Configuration

#### PostgreSQL Backup

```bash
#!/bin/bash
# backup-postgres.sh

DATE=$(date +%Y%m%d_%H%M%S)
docker exec arc-hawk-postgres pg_dump -U arc_hawk arc_hawk_production > backup_$DATE.sql

# Upload to S3 (optional)
aws s3 cp backup_$DATE.sql s3://your-backup-bucket/arc-hawk/postgres/

# Keep only last 7 days
find . -name "backup_*.sql" -mtime +7 -delete
```

#### Neo4j Backup

```bash
#!/bin/bash
# backup-neo4j.sh

DATE=$(date +%Y%m%d_%H%M%S)
docker exec arc-hawk-neo4j neo4j-admin dump --database=neo4j --to=/backups/backup_$DATE.dump

# Copy from container
docker cp arc-hawk-neo4j:/backups/backup_$DATE.dump ./

# Upload to S3 (optional)
aws s3 cp backup_$DATE.dump s3://your-backup-bucket/arc-hawk/neo4j/
```

---

## Monitoring

### Health Checks

All services expose health endpoints:

```bash
# Backend
curl http://localhost:8080/health

# Frontend
curl http://localhost:3000/api/health
```

### Metrics

**Prometheus metrics** (backend):

```bash
curl http://localhost:8080/metrics
```

**Key metrics to monitor:**
- Request latency (p50, p95, p99)
- Error rate
- Active connections
- Database query time
- Queue depth (Temporal)

### Logging

**Centralized logging with ELK stack:**

```yaml
# Add to docker-compose.yml
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    environment:
      - discovery.type=single-node
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data

  logstash:
    image: docker.elastic.co/logstash/logstash:8.11.0
    volumes:
      - ./logstash.conf:/usr/share/logstash/pipeline/logstash.conf

  kibana:
    image: docker.elastic.co/kibana/kibana:8.11.0
    ports:
      - "5601:5601"
```

### Alerting

**Configure Prometheus AlertManager:**

```yaml
# alertmanager.yml
groups:
- name: arc-hawk
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: High error rate detected
```

---

## Troubleshooting

### Common Issues

**Services not starting:**
```bash
# Check logs
docker-compose logs

# Check resource usage
docker stats

# Restart services
docker-compose restart
```

**Database connection issues:**
```bash
# Test PostgreSQL connection
docker exec -it arc-hawk-postgres psql -U arc_hawk -c "SELECT 1"

# Test Neo4j connection
docker exec -it arc-hawk-neo4j cypher-shell -u neo4j -p password "RETURN 1"
```

**Out of memory:**
```bash
# Check memory usage
free -h
docker system df

# Increase swap (temporary)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### Performance Issues

**Slow API responses:**
1. Check database indexes
2. Monitor query performance
3. Increase connection pool size
4. Add caching layer

**Scanner slow:**
1. Increase worker threads
2. Check network bandwidth
3. Optimize file patterns
4. Use sampling for large datasets

---

## Maintenance

### Updates

**Update application:**

```bash
# Pull latest changes
git pull origin main

# Rebuild images
docker-compose -f docker-compose.production.yml build --no-cache

# Restart services
docker-compose -f docker-compose.production.yml up -d
```

**Database migrations:**

```bash
# Run migrations
docker exec arc-hawk-backend ./migrate up

# Check status
docker exec arc-hawk-backend ./migrate status
```

### Scaling

**Horizontal scaling (Docker Compose):**

```bash
# Scale backend
docker-compose up -d --scale backend=3
```

**Horizontal scaling (Kubernetes):**

```bash
# Scale deployment
kubectl scale deployment backend --replicas=5 -n arc-hawk

# Enable HPA
kubectl autoscale deployment backend --min=2 --max=10 --cpu-percent=70 -n arc-hawk
```

---

## Security Hardening

### Checklist

- [ ] Changed all default passwords
- [ ] Generated strong secrets
- [ ] Enabled TLS/SSL
- [ ] Configured firewall rules
- [ ] Restricted database access
- [ ] Set up log monitoring
- [ ] Enabled audit logging
- [ ] Configured backups
- [ ] Set up alerting
- [ ] Reviewed [Security Policy](../../SECURITY.md)

### Docker Security

```yaml
# Add to docker-compose.services
security_opt:
  - no-new-privileges:true
cap_drop:
  - ALL
cap_add:
  - CHOWN
  - SETGID
  - SETUID
read_only: true
tmpfs:
  - /tmp:noexec,nosuid,size=100m
```

---

## Resources

- [Architecture Overview](../architecture/ARCHITECTURE.md)
- [Security Policy](../../SECURITY.md)
- [Monitoring Guide](../deployment/monitoring.md)
- [Backup Guide](../deployment/backup.md)

---

**Last Updated**: February 10, 2026  
**Version**: 2.1.0
