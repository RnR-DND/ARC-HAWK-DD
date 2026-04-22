# =============================================================================
# ARC-Hawk Development Commands
# =============================================================================
# Usage: make <target>
#
# Targets:
#   start       - Start all services
#   stop        - Stop all services
#   restart     - Restart all services
#   logs        - View logs for all services
#   logs-<svc>  - View logs for specific service (backend, frontend, etc.)
#   build       - Build all Docker images
#   rebuild     - Rebuild all Docker images (no cache)
#   clean       - Stop services and remove volumes
#   status      - Check service health
#   test        - Run all tests
#   test-backend    - Run backend tests
#   test-frontend   - Run frontend tests
#   test-scanner    - Run scanner tests
#   deploy      - Deploy to current environment
#   deploy-staging  - Deploy to staging
#   deploy-production - Deploy to production
# =============================================================================

.PHONY: start stop restart logs build rebuild clean status test test-backend test-frontend test-scanner deploy deploy-staging deploy-production openapi verify

# Default target
all: start

# Start all services
start:
	@echo "Starting ARC-Hawk services..."
	docker-compose up -d

# Stop all services
stop:
	@echo "Stopping ARC-Hawk services..."
	docker-compose down

# Restart all services
restart: stop start

# View logs
logs:
	docker-compose logs -f

# View logs for specific service
logs-%:
	docker-compose logs -f $*

# Build all images
build:
	@echo "Building Docker images..."
	docker-compose build

# Rebuild all images (no cache)
rebuild:
	@echo "Rebuilding Docker images (no cache)..."
	docker-compose build --no-cache

# Clean up (stop + remove volumes)
clean:
	@echo "Stopping services and removing volumes..."
	@read -p "This will delete all data. Continue? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		echo "Done!"; \
	else \
		echo "Aborted!"; \
	fi

# Check service status
status:
	@echo "Checking service status..."
	docker-compose ps
	@echo ""
	@echo "Health checks:"
	@curl -sf http://localhost:5432/postgres/health 2>/dev/null && echo "  ✓ PostgreSQL" || echo "  ✗ PostgreSQL"
	@curl -sf http://localhost:7474 2>/dev/null && echo "  ✓ Neo4j" || echo "  ✗ Neo4j"
	@curl -sf http://localhost:5001/health 2>/dev/null && echo "  ✓ Presidio" || echo "  ✗ Presidio"
	@curl -sf http://localhost:7233 2>/dev/null && echo "  ✓ Temporal" || echo "  ✗ Temporal"
	@curl -sf http://localhost:8080/api/v1/health 2>/dev/null && echo "  ✓ Backend" || echo "  ✗ Backend"
	@curl -sf http://localhost:3000 2>/dev/null && echo "  ✓ Frontend" || echo "  ✗ Frontend"

# Run all tests
test: test-backend test-frontend test-scanner

# Run backend tests
test-backend:
	@echo "Running backend tests..."
	cd apps/backend && go test ./... -short

# Run frontend tests
test-frontend:
	@echo "Running frontend tests..."
	cd apps/frontend && npm test -- --passWithNoTests

# Run scanner tests
test-scanner:
	@echo "Running Go scanner tests..."
	cd apps/goScanner && go test ./... -v

# Deploy to current environment
deploy:
	@bash scripts/deploy.sh

# Deploy to staging
deploy-staging:
	@bash scripts/deploy.sh staging

# Deploy to production
deploy-production:
	@echo "Deploying to PRODUCTION..."
	@read -p "Are you sure you want to deploy to production? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		bash scripts/deploy.sh production; \
	else \
		echo "Aborted!"; \
	fi

# Generate OpenAPI spec from swaggo annotations
openapi:
	@echo "Generating OpenAPI spec..."
	@cd apps/backend && ~/go/bin/swag init -g cmd/server/main.go -o docs/openapi --parseDependency --parseInternal
	@echo "✅ OpenAPI spec generated at apps/backend/docs/openapi/"

# Run full verification suite (build + tests + lint + helm)
verify:
	@echo "=== ARC-Hawk Verification Suite ==="
	@echo "--- Backend build ---"
	cd apps/backend && go build ./...
	@echo "--- Backend vet ---"
	cd apps/backend && go vet ./...
	@echo "--- Backend tests ---"
	cd apps/backend && go test ./... -short
	@echo "--- Scanner build ---"
	cd apps/goScanner && go build ./...
	@echo "--- Scanner vet ---"
	cd apps/goScanner && go vet ./...
	@echo "--- Helm lint ---"
	helm lint helm/arc-hawk-dd/
	@echo "=== ✅ Verification complete ==="

# Help
help:
	@echo "ARC-Hawk Development Commands"
	@echo ""
	@echo "  make start          - Start all services"
	@echo "  make stop           - Stop all services"
	@echo "  make restart        - Restart all services"
	@echo "  make logs           - View all logs"
	@echo "  make logs-backend   - View backend logs"
	@echo "  make logs-frontend  - View frontend logs"
	@echo "  make build          - Build all Docker images"
	@echo "  make rebuild        - Rebuild with no cache"
	@echo "  make clean          - Stop and remove volumes"
	@echo "  make status         - Check service health"
	@echo "  make test           - Run all tests"
	@echo "  make deploy         - Deploy to current env"
	@echo "  make help           - Show this help"
