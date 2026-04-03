# ARC-HAWK-DD Development Configuration

## Project Overview

**ARC-HAWK-DD** is a Go/React-based application with:
- Backend: Go microservices (auth, scanning, remediation, connections)
- Database: PostgreSQL with versioned migrations
- Architecture: Modular design with separate services

## gstack Integration

This project uses **gstack** as the primary development workflow orchestrator. All team members should use gstack skills for development.

### Web Browsing
- Use **`/browse`** skill from gstack for all web browsing and research
- NEVER use `mcp__claude-ai-chrome__*` tools or MCP browser tools
- The /browse skill provides a more integrated browsing experience

### Key gstack Skills for This Project

| Skill | Purpose | When to Use |
|-------|---------|-------------|
| `/plan-eng-review` | Engineering design review | Planning backend changes, architecture decisions |
| `/review` | Code review | After implementing features or fixes |
| `/ship` | Deploy and release | Before merging to main |
| `/land-and-deploy` | Landing and deployment | Production deployments |
| `/qa` | Quality assurance testing | Before releases, testing user flows |
| `/investigate` | Investigation and debugging | Troubleshooting issues, analyzing logs |
| `/autoplan` | Automatic planning | Feature planning and breakdown |
| `/browse` | Web browsing | Research, documentation lookup |

**Full list of available skills:** `/office-hours`, `/plan-ceo-review`, `/plan-eng-review`, `/plan-design-review`, `/design-consultation`, `/design-shotgun`, `/design-html`, `/design-review`, `/review`, `/ship`, `/land-and-deploy`, `/canary`, `/benchmark`, `/qa`, `/qa-only`, `/investigate`, `/document-release`, `/retro`, `/codex`, `/cso`, `/autoplan`, `/careful`, `/freeze`, `/guard`, `/unfreeze`, `/gstack-upgrade`, `/learn`, `/checkpoint`, `/browse`, `/setup-browser-cookies`, `/setup-deploy`, `/connect-chrome`

## Project Workflow Rules

1. **Always use gstack skills** for development workflows
2. **Never use ecc (everything-claude-code) orchestrators** — use gstack instead
3. **Never use gsd orchestrators** — use gstack instead
4. **Web browsing**: Always use `/browse` skill, never mcp browser tools

## Tech Stack

- **Language**: Go (backend), React (frontend)
- **Database**: PostgreSQL with migrations (`migrations_versioned/`)
- **Modules**: auth, scanning, connections, remediation, shared
- **Build**: Standard Go build system
- **Testing**: Unit + integration tests required (80% coverage minimum)

## Code Review Standards

Before merging:
- [ ] Code passes all automated checks
- [ ] Tests cover 80%+ of changes
- [ ] Security review for auth/DB changes
- [ ] Use `/review` skill for detailed review
- [ ] No hardcoded secrets or credentials
- [ ] Error handling is explicit

## Commit Message Format

```
<type>: <description>

<optional body>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`, `ci`

## Important Files & Directories

- `apps/backend/` — Go backend services
- `apps/backend/migrations_versioned/` — Database migrations (000018+)
- `apps/backend/modules/` — Service modules (auth, scanning, etc.)
- `.continue-here.md` — Session handoff file for context preservation

## Session Continuity

If a session pauses, use the handoff file at `.continue-here.md` to preserve context and resume work efficiently.
