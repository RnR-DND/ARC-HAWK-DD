# Contributing to ARC-Hawk

Thank you for your interest in contributing to ARC-Hawk! This document provides guidelines and instructions for contributing to the project.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Style Guidelines](#style-guidelines)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Commit Messages](#commit-messages)
- [Release Process](#release-process)

---

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code:

- Be respectful and inclusive
- Welcome newcomers and help them learn
- Focus on constructive feedback
- Respect different viewpoints and experiences

---

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Git**
- **Docker** & **Docker Compose**
- **Go 1.21+** (for backend)
- **Node.js 18+** (for frontend)
- **Python 3.9+** (for scanner)

### Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/arc-hawk.git
cd arc-hawk

# Add upstream remote
git remote add upstream https://github.com/original-org/arc-hawk.git

# Create a new branch for your feature/fix
git checkout -b feature/your-feature-name
```

---

## Development Setup

### Option 1: Full Docker Setup (Recommended for Quick Start)

```bash
# Start all services
docker-compose up -d

# Access services:
# - Frontend: http://localhost:3000
# - Backend API: http://localhost:8080
# - Neo4j: http://localhost:7474
# - Temporal: http://localhost:8088
```

### Option 2: Local Development (Recommended for Active Development)

#### 1. Start Infrastructure Services

```bash
# Start only databases and infrastructure
docker-compose up -d postgres neo4j temporal
```

#### 2. Backend Development

```bash
cd apps/backend

# Copy environment file
cp .env.example .env
# Edit .env with your configuration

# Install dependencies
go mod tidy

# Run the server
go run cmd/server/main.go

# Server will start on http://localhost:8080
```

#### 3. Frontend Development

```bash
cd apps/frontend

# Install dependencies
npm install

# Create environment file
echo "NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1" > .env.local
echo "NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws" >> .env.local

# Run development server
npm run dev

# Dashboard available at http://localhost:3000
```

#### 4. Scanner Development

```bash
cd apps/scanner

# Create virtual environment (recommended)
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
python -m spacy download en_core_web_sm

# Run tests to verify setup
python -m pytest tests/ -v
```

---

## How to Contribute

### Reporting Bugs

Before creating a bug report, please:

1. Check [existing issues](../../issues) to avoid duplicates
2. Use the latest version to verify the bug still exists
3. Collect relevant information (logs, error messages, steps to reproduce)

**Bug Report Template:**

```markdown
**Description:**
Clear description of the bug

**Steps to Reproduce:**
1. Go to '...'
2. Click on '...'
3. See error

**Expected Behavior:**
What you expected to happen

**Actual Behavior:**
What actually happened

**Environment:**
- OS: [e.g., macOS 14.2]
- Browser: [e.g., Chrome 120]
- Version: [e.g., 2.1.0]
- Component: [e.g., Backend, Frontend, Scanner]

**Logs:**
```
Paste relevant logs here
```
```

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:

1. Check [existing discussions](../../discussions) for similar ideas
2. Provide a clear use case
3. Explain why it would be useful to most users

**Enhancement Template:**

```markdown
**Is your feature request related to a problem?**
A clear description of the problem

**Describe the solution you'd like**
What you want to happen

**Describe alternatives you've considered**
Other solutions you've thought about

**Additional context**
Any other context, mockups, or examples
```

### Contributing Code

1. **Find an Issue**: Look for issues labeled `good first issue` or `help wanted`
2. **Comment**: Comment on the issue to let others know you're working on it
3. **Develop**: Make your changes following our style guidelines
4. **Test**: Ensure all tests pass and add new tests for new features
5. **Document**: Update documentation as needed
6. **Submit**: Create a pull request with a clear description

---

## Style Guidelines

### Go (Backend)

We follow standard Go conventions:

- Use `gofmt` for formatting
- Run `go vet` and `golint` before committing
- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use meaningful variable names
- Document exported functions with comments

**Example:**

```go
// ScanAsset initiates a scan for the given asset ID.
// It returns the scan ID and any error encountered.
func ScanAsset(ctx context.Context, assetID string) (string, error) {
    // Implementation
}
```

### TypeScript/React (Frontend)

- Use TypeScript strict mode
- Follow the existing component structure
- Use functional components with hooks
- Use Tailwind CSS for styling
- Name files using PascalCase for components, camelCase for utilities

**Example:**

```typescript
interface ScanCardProps {
  scanId: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
}

export const ScanCard: React.FC<ScanCardProps> = ({ scanId, status }) => {
  return (
    <div className="p-4 border rounded-lg">
      <h3 className="text-lg font-semibold">{scanId}</h3>
      <StatusBadge status={status} />
    </div>
  );
};
```

### Python (Scanner)

Follow PEP 8 guidelines:

- Use 4 spaces for indentation
- Maximum line length: 100 characters
- Use docstrings for all public functions
- Type hints are encouraged
- Run `flake8` and `black` before committing

**Example:**

```python
def validate_aadhaar(number: str) -> bool:
    """
    Validate an Aadhaar number using the Verhoeff algorithm.
    
    Args:
        number: The Aadhaar number to validate
        
    Returns:
        True if valid, False otherwise
    """
    # Implementation
    pass
```

---

## Testing

### Running Tests

**All Tests:**
```bash
./scripts/testing/run-tests.sh
```

**Backend Tests:**
```bash
cd apps/backend
go test ./... -v
```

**Frontend Tests:**
```bash
cd apps/frontend
npm test -- --passWithNoTests
```

**Scanner Tests:**
```bash
cd apps/scanner
python -m pytest tests/ -v
```

### Writing Tests

- **Backend**: Use Go's built-in testing package
- **Frontend**: Use React Testing Library
- **Scanner**: Use pytest

**Test Coverage Goals:**
- Core validation logic: >90%
- API handlers: >80%
- Integration points: >70%

---

## Pull Request Process

1. **Update Documentation**: Update README.md, API docs, or other documentation as needed
2. **Add Tests**: Ensure new code has tests and all tests pass
3. **Update CHANGELOG.md**: Add your changes under the `[Unreleased]` section
4. **Link Issues**: Reference any related issues in your PR description
5. **Request Review**: Request review from maintainers

**PR Template:**

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass locally
- [ ] Added tests for new functionality
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
```

---

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, semicolons, etc.)
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build process, dependencies, etc.

**Examples:**

```
feat(scanner): add Redis connector support

Implement Redis scanning capability with support for:
- Key pattern matching
- Value type detection
- Connection pooling

Closes #123
```

```
fix(backend): correct lineage query for null assets

- Updated Cypher query to handle null asset nodes
- Added null checks in service layer
- Added test case for edge condition

Fixes #456
```

---

## Release Process

1. **Version Bump**: Update version in relevant files
2. **CHANGELOG**: Move changes from `[Unreleased]` to version section
3. **Tag**: Create git tag: `git tag -a v2.1.0 -m "Release version 2.1.0"`
4. **Push**: Push tag: `git push origin v2.1.0`
5. **GitHub Release**: Create release on GitHub with release notes

---

## Questions?

- **General Questions**: [GitHub Discussions](../../discussions)
- **Bug Reports**: [GitHub Issues](../../issues)
- **Security Issues**: See [SECURITY.md](./SECURITY.md)

---

Thank you for contributing to ARC-Hawk! 🎉
