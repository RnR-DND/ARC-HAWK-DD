# Ralph Agent Instructions — ARC-Hawk

You are an autonomous coding agent working on the **ARC-Hawk** PII discovery, classification, and lineage tracking platform.

## Project Context

- **Backend**: Go 1.21+ (Gin framework), PostgreSQL 15, Neo4j 5.15, Temporal
- **Frontend**: Next.js 14, TypeScript, ReactFlow, Cytoscape, Tailwind CSS
- **Scanner**: Python 3.9+, spaCy NLP, custom validators (Verhoeff, Luhn)
- **Infrastructure**: Docker, Docker Compose, Kubernetes

## Your Task

1. Read the PRD at `prd.json` (in the same directory as this file)
2. Read the progress log at `progress.txt` (check Codebase Patterns section first)
3. Check you're on the correct branch from PRD `branchName`. If not, check it out or create from main.
4. Pick the **highest priority** user story where `passes: false`
5. Implement that single user story
6. Run quality checks (see below)
7. Update AGENTS.md files if you discover reusable patterns
8. If checks pass, commit ALL changes with message: `feat: [Story ID] - [Story Title]`
9. Update the PRD to set `passes: true` for the completed story
10. Append your progress to `progress.txt`

## ARC-Hawk Quality Checks

```bash
# Backend
cd apps/backend && go vet ./... && go test ./...

# Frontend
cd apps/frontend && npm run lint && npm run build

# Scanner
cd apps/scanner && python -m pytest tests/

# Infrastructure
docker-compose config --quiet
```

## ARC-Hawk Codebase Conventions

- **CRITICAL**: Always reference `apps/scanner/config/connection.yml.sample` for connection schemas. Each data source type requires UNIQUE parameters — they are NOT identical.
- **Intelligence-at-Edge**: Scanner SDK is the sole authority for PII validation. Backend NEVER re-validates.
- **Module Structure**: Backend has 8 core modules: scanning, assets, lineage, compliance, masking, analytics, connections, remediation.
- **Import Style**: Go uses CamelCase for exported, camelCase for unexported. Frontend uses `@/*` alias imports.
- **Error Handling**: Go returns explicit errors with `fmt.Errorf`. TypeScript uses try-catch.
- **DB Access**: PostgreSQL via GORM, Neo4j via official driver.
- **API Format**: RESTful with `{ success: bool, data: any, error: string|null }` response format.

## Progress Report Format

APPEND to progress.txt (never replace, always append):
```
## [Date/Time] - [Story ID]
- What was implemented
- Files changed
- **Learnings for future iterations:**
  - Patterns discovered
  - Gotchas encountered
  - Useful context
---
```

## Consolidate Patterns

If you discover a **reusable pattern**, add it to the `## Codebase Patterns` section at the TOP of progress.txt:

```
## Codebase Patterns
- Always use connection.yml.sample (NOT connection.yml) for reference
- Backend modules follow: api/ → service/ → repository/ layering
- Scanner connectors follow the BaseConnector pattern
- Frontend services use api-client.ts for all backend calls
```

## Stop Condition

After completing a user story, check if ALL stories have `passes: true`.
If ALL complete: reply with `<promise>COMPLETE</promise>`
If stories remain: end normally (another iteration will pick up the next story).

## Important

- Work on ONE story per iteration
- Commit frequently
- Keep CI green
- Read the Codebase Patterns section in progress.txt before starting
- Load relevant `@skills` for your task domain (e.g., `@golang-pro`, `@nextjs-best-practices`, `@python-patterns`)
