# Development Guide

Complete guide for setting up and developing the ARC-Hawk platform locally.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Setup](#quick-setup)
- [Detailed Setup](#detailed-setup)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Debugging](#debugging)
- [Best Practices](#best-practices)

---

## Prerequisites

### Required Software

- **Docker Desktop** 4.0+ (for infrastructure services)
- **Git** 2.30+
- **Go** 1.25+ (backend)
- **Node.js** 18+ (frontend)
- **Python** 3.9+ (scanner)

### Optional but Recommended

- **VS Code** with extensions:
  - Go
  - TypeScript and JavaScript
  - Python
  - Docker
  - YAML
- **Make** (for using Makefile commands)
- **Postman** or **Insomnia** (for API testing)

### System Requirements

- **RAM**: 8GB minimum, 16GB recommended
- **CPU**: 4 cores minimum
- **Disk**: 10GB free space
- **OS**: macOS, Linux, or Windows with WSL2

---

## Quick Setup

### 5-Minute Setup (Docker Everything)

```bash
# 1. Clone repository
git clone https://github.com/your-org/arc-hawk.git
cd arc-hawk

# 2. Start all services
docker-compose up -d

# 3. Access services
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
# Neo4j: http://localhost:7474
# Temporal: http://localhost:8088
```

### Development Setup (Recommended)

For active development, run infrastructure in Docker and applications locally.

---

## Detailed Setup

### Step 1: Infrastructure Services

Start databases and infrastructure:

```bash
# Start infrastructure only
docker-compose up -d postgres neo4j temporal

# Verify services are running
docker-compose ps
```

**Services started:**
- PostgreSQL: `localhost:5432`
- Neo4j: `localhost:7687` (bolt), `localhost:7474` (browser)
- Temporal: `localhost:7233` (gRPC), `localhost:8088` (UI)

### Step 2: Backend Setup

```bash
cd apps/backend

# Copy environment file
cp .env.example .env

# Edit .env with your settings
# Change passwords if needed

# Install Go dependencies
go mod tidy

# Run migrations (if needed)
go run cmd/neo4j_migrate/main.go

# Start server
go run cmd/server/main.go

# Server runs on http://localhost:8080
```

**Verify backend:**
```bash
curl http://localhost:8080/health
# Should return: {"status":"healthy"}
```

### Step 3: Frontend Setup

```bash
cd apps/frontend

# Create environment file
cat > .env.local << EOF
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
EOF

# Install dependencies
npm install

# Start development server
npm run dev

# Dashboard available at http://localhost:3000
```

### Step 4: Scanner Setup

```bash
cd apps/scanner

# Create virtual environment
python -m venv venv

# Activate virtual environment
source venv/bin/activate  # macOS/Linux
# venv\Scripts\activate  # Windows

# Install dependencies
pip install -r requirements.txt

# Download spaCy model
python -m spacy download en_core_web_sm

# Verify installation
python -m hawk_scanner.main --help
```

**Test scanner:**
```bash
# Create test data
mkdir -p /tmp/test-data
echo "Test email: user@example.com" > /tmp/test-data/test.txt

# Run scan
python -m hawk_scanner.main fs --path /tmp/test-data --json output.json

# View results
cat output.json
```

---

## Development Workflow

### Daily Development Workflow

```bash
# 1. Start infrastructure (if not running)
docker-compose up -d postgres neo4j temporal

# 2. Terminal 1: Backend
cd apps/backend
go run cmd/server/main.go

# 3. Terminal 2: Frontend
cd apps/frontend
npm run dev

# 4. Terminal 3: Scanner (when needed)
cd apps/scanner
source venv/bin/activate
```

### Running Tests

```bash
# Backend tests
cd apps/backend
go test ./... -v

# Frontend tests
cd apps/frontend
npm test

# Scanner tests
cd apps/scanner
python -m pytest tests/ -v
```

### Making Changes

#### Backend Changes

1. Edit Go files
2. Run `go fmt ./...` to format
3. Run tests: `go test ./...`
4. Test manually: `go run cmd/server/main.go`

#### Frontend Changes

1. Edit TypeScript/React files
2. Run `npm run lint` to check
3. Fix any linting errors
4. Test in browser at `http://localhost:3000`

#### Scanner Changes

1. Edit Python files
2. Run `flake8` to check style
3. Run tests: `python -m pytest tests/`
4. Test manually: `python -m hawk_scanner.main ...`

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/my-feature

# Make changes and commit
git add .
git commit -m "feat: add new feature"

# Push and create PR
git push origin feature/my-feature
# Create PR on GitHub
```

---

## Testing

### Unit Tests

**Backend:**
```bash
cd apps/backend

# Run all tests
go test ./... -v

# Run specific package
go test ./modules/scanning -v

# Run with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Frontend:**
```bash
cd apps/frontend

# Run tests
npm test

# Run with coverage
npm test -- --coverage

# Run specific test
npm test -- Button.test.tsx
```

**Scanner:**
```bash
cd apps/scanner

# Run all tests
python -m pytest tests/ -v

# Run specific test file
python -m pytest tests/test_validation.py -v

# Run with coverage
python -m pytest tests/ --cov=hawk_scanner
```

### Integration Tests

```bash
# Start all services
docker-compose up -d

# Run integration tests
cd apps/backend
go test ./tests/integration/... -v
```

### End-to-End Tests

```bash
# Using Playwright (requires setup)
cd apps/frontend
npm run test:e2e
```

---

## Debugging

### Backend Debugging

**Using VS Code:**

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Backend",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/apps/backend/cmd/server/main.go",
      "env": {
        "ENV": "development"
      }
    }
  ]
}
```

**Using Delve (command line):**

```bash
cd apps/backend

# Start with debugger
dlv debug cmd/server/main.go

# Set breakpoint
(dlv) break main.main
(dlv) continue
```

### Frontend Debugging

**Browser DevTools:**
- Open Chrome DevTools (F12)
- Use React DevTools extension
- Check Console for errors
- Use Network tab for API debugging

**VS Code Debugging:**

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Next.js: debug server-side",
      "type": "node-terminal",
      "request": "launch",
      "command": "npm run dev",
      "cwd": "${workspaceFolder}/apps/frontend"
    }
  ]
}
```

### Scanner Debugging

**Using Python Debugger:**

```python
# Add to your code
import pdb; pdb.set_trace()

# Or use breakpoint() in Python 3.7+
breakpoint()
```

**VS Code Debugging:**

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Python: Scanner",
      "type": "python",
      "request": "launch",
      "module": "hawk_scanner.main",
      "args": ["fs", "--path", "/tmp/test-data"],
      "cwd": "${workspaceFolder}/apps/scanner"
    }
  ]
}
```

---

## Common Issues

### Backend Issues

**Port already in use:**
```bash
# Find process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>

# Or use different port
PORT=8081 go run cmd/server/main.go
```

**Database connection refused:**
```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# Check logs
docker-compose logs postgres

# Restart PostgreSQL
docker-compose restart postgres
```

**Neo4j connection failed:**
```bash
# Check Neo4j status
docker-compose ps neo4j

# Reset Neo4j (WARNING: deletes all data)
docker-compose down neo4j
docker volume rm arc-hawk_neo4j_data
docker-compose up -d neo4j
```

### Frontend Issues

**Module not found:**
```bash
# Clear node_modules and reinstall
rm -rf node_modules package-lock.json
npm install
```

**Build errors:**
```bash
# Clear Next.js cache
rm -rf .next
npm run dev
```

**TypeScript errors:**
```bash
# Check TypeScript
npx tsc --noEmit

# Fix formatting
npx prettier --write .
```

### Scanner Issues

**spaCy model not found:**
```bash
# Download model
python -m spacy download en_core_web_sm
```

**Permission denied:**
```bash
# Fix permissions
chmod -R +x apps/scanner

# Or run with python explicitly
python -m hawk_scanner.main ...
```

**Import errors:**
```bash
# Ensure virtual environment is activated
source venv/bin/activate

# Reinstall dependencies
pip install -r requirements.txt
```

---

## Best Practices

### Code Style

**Go:**
- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use meaningful variable names
- Document exported functions

**TypeScript:**
- Use strict mode
- Prefer `const` over `let`
- Use descriptive variable names
- Add types to all functions

**Python:**
- Follow PEP 8
- Use type hints
- Document public functions
- Use list/dict comprehensions appropriately

### Git Commits

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(backend): add new scan endpoint

fix(scanner): correct Aadhaar validation
docs(api): update endpoint documentation
refactor(frontend): simplify dashboard component
test(backend): add scan workflow tests
chore(deps): update dependencies
```

### Pull Requests

1. Create descriptive PR titles
2. Include screenshots for UI changes
3. Add tests for new features
4. Update documentation
5. Link related issues
6. Request review from team members

### Environment Variables

- Never commit `.env` files
- Use `.env.example` for templates
- Document all environment variables
- Use strong secrets in production

### Testing

- Write tests before fixing bugs
- Aim for >80% coverage
- Test edge cases
- Use table-driven tests in Go
- Mock external dependencies

---

## Performance Optimization

### Backend Optimization

**Enable Go profiler:**
```bash
# Add to main.go
import _ "net/http/pprof"

# Access profiler
go tool pprof http://localhost:8080/debug/pprof/profile
```

### Frontend Optimization

**Analyze bundle size:**
```bash
cd apps/frontend
npm run analyze
```

**Use React DevTools Profiler:**
- Install React DevTools browser extension
- Use Profiler tab to identify slow components

### Scanner Optimization

**Profile Python code:**
```bash
# Use cProfile
python -m cProfile -o output.prof -m hawk_scanner.main fs --path /data

# View with snakeviz
pip install snakeviz
snakeviz output.prof
```

---

## Resources

### Documentation
- [Architecture Overview](../architecture/ARCHITECTURE.md)
- [API Reference](../API.md)
- [Contributing Guide](../../CONTRIBUTING.md)
- [Security Policy](../../SECURITY.md)

### Tools
- [Go Documentation](https://golang.org/doc)
- [Next.js Documentation](https://nextjs.org/docs)
- [React Documentation](https://react.dev)
- [Python Documentation](https://docs.python.org/3/)

### Community
- GitHub Issues: Bug reports and feature requests
- GitHub Discussions: Questions and ideas
- Email: dev@arc-hawk.io

---

**Last Updated**: February 10, 2026  
**Version**: 2.1.0
