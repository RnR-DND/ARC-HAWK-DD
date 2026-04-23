# ARC-HAWK-DD Development Configuration

## Project Overview

**ARC-HAWK-DD** is a Go/React-based application with:
- Backend: Go microservices (auth, scanning, remediation, connections)
- Database: PostgreSQL with versioned migrations
- Architecture: Modular design with separate services

## Tool Stack Architecture

```
Layer 0: TOKEN ECONOMICS (always on, automatic)
  Caveman → -75% output tokens, auto-activates each session
  Claude-Mem compress → -46% context tokens (run /caveman:compress CLAUDE.md)

Layer 1: SESSION MEMORY (automatic, cross-session)
  Claude-Mem → SQLite + Chroma vector, port 37777, /mem-search
  Built-in memory → ~/.claude/projects/.../memory/ (explicit notes)

Layer 1.5: STRUCTURAL MEMORY (rebuild after major changes)
  Graphify → codebase knowledge graph, /graphify query/path/explain
  Push to Neo4j: /graphify . --neo4j-push bolt://localhost:7687
  MCP server for agents: /graphify . --mcp

Layer 2: SKILLS LIBRARY (860 skills, invoke with /)
  Awesome-Claude-Skills → domain tasks, 500+ SaaS automations

Layer 3: WORKFLOW ENGINE (daily driver)
  Gstack → /browse /qa /ship /review /investigate /autoplan /checkpoint

Layer 4: AUTONOMOUS EXECUTION (situational)
  Ralph → known stories from prd.json (batch implementation)
  GSD → phase planning for large unknown features
```

## Workflow Rules

1. **Web browsing**: Always use `/browse` (Gstack), never MCP browser tools
2. **Daily work**: Route to Gstack skills first (see routing table below)
3. **Batch features**: Use Ralph with prd.json — run `bash .agent/ralph/ralph.sh`
4. **Big new feature planning**: Use `gsd:plan-phase` or `gsd:discuss-phase`
5. **Skills**: Use `/skill-name` for any of the 860+ domain skills
6. **Memory search**: Use `/mem-search` to query past session work
7. **Never**: Use Hive (retired — Ralph + AgentSys cover its use cases)

## Gstack — Daily Workflow Skills

| Skill | Purpose | When to Use |
|-------|---------|-------------|
| `/browse` | Headless Chromium browser | Research, docs, web tasks |
| `/qa` | QA testing with Playwright | Before releases, testing flows |
| `/review` | Code review | After implementing features |
| `/ship` | Deploy and release workflow | Before merging to main |
| `/land-and-deploy` | Production deployment | Production releases |
| `/investigate` | Systematic debugging | Errors, 500s, broken things |
| `/autoplan` | Automatic planning | Feature planning and breakdown |
| `/checkpoint` | Save/resume context | Before complex multi-step work |
| `/plan-eng-review` | Architecture review | Backend changes, architecture decisions |
| `/design-review` | Visual/UI audit | UI polish, design check |
| `/health` | Code quality dashboard | Before releases |
| `/document-release` | Update docs | After shipping |

**All Gstack skills:** `/office-hours`, `/plan-ceo-review`, `/plan-eng-review`, `/plan-design-review`, `/design-consultation`, `/design-shotgun`, `/design-html`, `/design-review`, `/review`, `/ship`, `/land-and-deploy`, `/canary`, `/benchmark`, `/qa`, `/qa-only`, `/investigate`, `/document-release`, `/retro`, `/codex`, `/cso`, `/autoplan`, `/careful`, `/freeze`, `/guard`, `/unfreeze`, `/gstack-upgrade`, `/learn`, `/checkpoint`, `/browse`, `/setup-browser-cookies`, `/setup-deploy`, `/connect-chrome`

## Awesome Claude Skills — Domain Skills

860 skills installed globally at `~/.claude/skills/`. Use any with `/skill-name`.

Key skills for this stack:
- **Dev**: `/changelog-generator`, `/artifacts-builder`, `/mcp-builder`, `/webapp-testing`
- **Content**: `/content-research-writer`, `/tailored-resume-writer`, `/internal-comms`
- **Productivity**: `/file-organizer`, `/meeting-insights-analyzer`, `/skill-creator`
- **SaaS automation**: `/slack-automation`, `/jira-automation`, `/notion-automation`, `/gmail-automation`
- **Media**: `/video-downloader`, `/canvas-design`, `/theme-factory`, `/image-enhancer`

## Claude-Mem — Persistent Memory

Automatic cross-session memory. No manual action needed — it captures everything.

```bash
npx claude-mem start          # start the worker (port 37777)
```

- Web UI: http://localhost:37777
- Search past work: `/mem-search` in Claude Code
- Compress context: `/caveman:compress CLAUDE.md` (run once per session)

## Caveman — Token Compression

Auto-activates every session (installed via hooks). Saves ~75% output tokens.

```
/caveman          # toggle on/off
/caveman lite     # light compression
/caveman ultra    # maximum compression
/caveman:compress CLAUDE.md   # compress this file to cut input tokens
```

## Ralph — Autonomous PRD Loop

Spec-driven batch implementation. Use when you have multiple clear stories.

```bash
# Edit .agent/ralph/prd.arc-hawk-dd.json — set passes: false for stories to implement
bash .agent/ralph/ralph.sh
# Ralph picks highest-priority story → implements → tests → commits → repeats
```

## GSD — Phase Planning (planning only, not execution)

Use for planning large features. Skip gsd:execute-phase — use Gstack for execution.

```
/gsd:plan-phase       # plan a new phase
/gsd:discuss-phase    # clarify requirements before planning
/gsd:code-review      # systematic code review
/gsd:secure-phase     # security audit of a phase
```

## AgentSys

Modular agent runtime at `.agent/agentsys/`. 19 plugins, 49 specialized agents.
Use for structured pipeline work where phase gates matter (security reviews, audits).

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
- `apps/agent/` — Edge scanner agent (Go)
- `apps/goScanner/` — Main scanner service (Go) — connectors, classifier, orchestrator, Presidio integration
- `apps/frontend/components/ui/` — Shared UI primitives (MetricCards, Loading indicators, etc.)
- `infra/k8s/monitoring/` — Kubernetes ServiceMonitors + NetworkPolicy
- `.agent/ralph/prd.arc-hawk-dd.json` — Active PRD for Ralph autonomous loop
- `graphify-out/` — Knowledge graph (run `/graphify .` to rebuild after major changes)

## Knowledge Graph (Graphify — Layer 1.5: Structural Memory)

Graphify maps how the codebase is wired — which modules depend on what, cross-module relationships,
architectural clusters. Complements Claude-Mem (which captures what happened in sessions).

ALWAYS query before: deep refactors, architectural decisions, touching unfamiliar modules.
ALWAYS rebuild after: major feature additions, module restructuring.

```bash
# Query (uses graphify CLI directly — fast, no LLM needed)
graphify query "how does the scan pipeline connect to neo4j?"
graphify path "IngestionService" "Neo4jRepository"
graphify explain "SharedAnalyzerEngine"

# Rebuild via skill (LLM-powered extraction — run in Claude Code)
# Type in prompt: /graphify .            ← full rebuild
# Type in prompt: /graphify . --update  ← incremental (changed files only)

# Push to Neo4j (ARC-HAWK-DD already runs Neo4j)
# Type in prompt: /graphify . --neo4j-push bolt://localhost:7687

# Start MCP server for agent access (Ralph/AgentSys query without reading files)
graphify watch .   # auto-rebuilds on code changes (no LLM)

# Git hooks are installed — graph updates automatically on commit/checkout
```

## Skill Routing

When the user's request matches an available skill, ALWAYS invoke it using the Skill
tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.
The skill has specialized workflows that produce better results than ad-hoc answers.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke office-hours
- Bugs, errors, "why is this broken", 500 errors → invoke investigate
- Ship, deploy, push, create PR → invoke ship
- QA, test the site, find bugs → invoke qa
- Code review, check my diff → invoke review
- Update docs after shipping → invoke document-release
- Weekly retro → invoke retro
- Design system, brand → invoke design-consultation
- Visual audit, design polish → invoke design-review
- Architecture review → invoke plan-eng-review
- Save progress, checkpoint, resume → invoke checkpoint
- Code quality, health check → invoke health
- Search past session work → use /mem-search
