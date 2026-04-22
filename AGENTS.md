# AGENTS.md — AI Automation Guide

Reference for all AI agents and automation tools active in this repo.

---

## 1. Claude Code (Primary)

**Tool:** [Claude Code](https://claude.ai/code) CLI  
**Config:** `.claude/settings.json`, `CLAUDE.md`  
**Scope:** All development tasks — feature implementation, debugging, code review, docs, refactoring

### gstack Skills (invoke via Skill tool or `/skill-name`)

| Skill | When to use |
|---|---|
| `/review` | Code review after implementing features |
| `/qa` | Quality assurance — test user flows |
| `/ship` | Create PR + run review + prepare for merge |
| `/investigate` | Debug errors, analyze logs |
| `/plan-eng-review` | Architecture review before major changes |
| `/browse` | Web research (never use MCP browser tools directly) |
| `/autoplan` | Feature planning and task breakdown |
| `/health` | Code quality dashboard |

### Hooks (`.claude/hooks/`)

- `check-gstack.sh` — enforces gstack skill usage for deployments; blocks direct CI bypasses

---

## 2. Antigravity Skills (`.agent/skills/`)

860+ expert markdown skills, loaded with `@skill-name` in prompts.

**Key skills for this stack:**

| Domain | Skills |
|---|---|
| Go backend | `@golang-pro`, `@go-concurrency-patterns`, `@api-design-principles`, `@backend-security-coder` |
| Frontend | `@nextjs-best-practices`, `@nextjs-app-router-patterns`, `@react-best-practices` |
| Infra | `@docker-expert`, `@postgres-best-practices` |
| Security | `@api-security-best-practices`, `@cc-skill-security-review` |
| Testing | `@test-driven-development`, `@e2e-testing-patterns` |

---

## 3. Ralph — Autonomous PRD Loop (`.agent/ralph/`)

Spec-driven autonomous iteration for batch feature work.

```bash
# 1. Create prd.json from prd.json.example
# 2. Run:
bash .agent/ralph/ralph.sh
# Ralph picks the highest-priority failing story, implements it, tests, commits, repeats
```

**Use for:** batch feature implementation, migrations, test coverage improvements.

---

## 4. Hive — Goal-Driven Agent Framework (`.agent/hive/`)

Define a goal in natural language; Hive auto-generates and runs an agent graph.

```bash
cd .agent/hive && node hive.js "add DPDPA compliance report export to PDF"
```

**Use for:** complex multi-step features needing parallel sub-agents.

---

## 5. CI / GitHub Actions (`.github/workflows/`)

| Workflow | Trigger | Purpose |
|---|---|---|
| `ci.yml` | PR | Build + test (`go test ./...`, `npx jest`, `helm lint`) |
| `claude.yml` | Issue/PR comment | Claude Code reviews triggered by `@claude` mention |
| `claude-code-review.yml` | PR | Automated code review on every PR |

### CI guardrails

- PRs require all checks to pass before merge
- `go vet ./...` runs on every push — no vet failures allowed
- Frontend: `npx jest` required; no `--passWithNoTests` flag

---

## 6. Graphify (knowledge graph)

Rebuild the repo knowledge graph after major changes:

```bash
/graphify .
# or: /graphify . --update
```

Query the graph:
```bash
/graphify query "how does the scan pipeline connect to neo4j?"
/graphify path "IngestionService" "Neo4jRepository"
```

Output at `graphify-out/` (gitignored).

---

## Guardrails

- **Never commit secrets** — `apps/backend/.env.production` was untracked in WS1; `SCANNER_SERVICE_TOKEN` must be rotated (see `TODO.md`)
- **Never skip hooks** (`--no-verify`) — hooks catch security policy violations
- **Never deploy without `/ship` skill** — it runs review + QA before landing
- **OpenAPI spec must stay in sync** — run `make openapi` after any handler change
