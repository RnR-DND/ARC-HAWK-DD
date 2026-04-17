# Agent toolchain (`/.agent`) — runbook

ARC-HAWK-DD ships 4 agent subsystems in `.agent/`. Each has a narrow, distinct
use case. None of them is loaded by default — invoke the right one for the job.

`.agent/` is **gitignored**. Updates are pulled from upstream repos manually
(see "Refreshing" at the bottom).

## Decision matrix — which tool for which job

| Situation | Tool | Why |
|---|---|---|
| "How do I write idiomatic Go concurrency?" / any specialist question | `.agent/skills/` | 1414 `@name`-loadable context cards. Load with `@golang-pro`, `@nextjs-best-practices`, etc. in your prompt. Cheapest. |
| "Implement this feature end-to-end over an autonomous loop" | `.agent/ralph/` | Bash loop that reads `prd.json`, picks a failing user story, implements + commits. Best for well-scoped batch work. |
| "Build and orchestrate a self-improving multi-agent graph for <goal>" | `.agent/hive/` | Y Combinator-backed runtime. 102 MCP tools. Overkill for one-shot tasks; shines when the solution is multi-step and parallelizable. |
| "I want a disciplined orchestration framework with plain-text output and strict PR discipline" | `.agent/agentsys/` | Modular runtime that cooperates with Claude Code. Stricter rules than vanilla. Use when you want more guardrails. |
| "Building a game" | `.agent/game-studios/` | Godot/Unity/Unreal subagent framework. **Not relevant to ARC-HAWK-DD** — kept because user requested. Ignore unless pivoting. |

## Skills (`.agent/skills/`)

1,414 expert markdown cards — one file per skill. Load them explicitly:

```
@golang-pro @api-security-best-practices  "design a rate limiter for /api/v1/scans"
```

Relevant to this stack:

- Go backend: `@golang-pro`, `@go-concurrency-patterns`, `@api-design-principles`, `@backend-security-coder`
- Frontend: `@nextjs-best-practices`, `@react-best-practices`, `@typescript-expert`
- Infra: `@docker-expert`, `@postgres-best-practices`, `@postgresql`
- Security: `@api-security-best-practices`, `@cc-skill-security-review`
- Testing: `@test-driven-development`, `@e2e-testing-patterns`

**No installation needed.** They're markdown files — Claude Code reads them on demand.

## Ralph (`.agent/ralph/`)

Bash-native autonomous loop. Reads a `prd.json` (user stories with `passes: false`),
picks the highest-priority failing story, implements + tests + commits, then
re-evaluates.

```bash
# 1. Prepare a PRD (already seeded for this project's deferred work)
cp .agent/ralph/prd.arc-hawk-dd.json .agent/ralph/prd.json

# 2. Run the loop
bash .agent/ralph/ralph.sh
```

The pre-seeded `prd.arc-hawk-dd.json` contains 4 user stories pulled from Session
14's deferred list: tenant-id middleware, missing compliance/auth routes, lint debt,
test coverage.

## Hive (`.agent/hive/`)

Goal-driven multi-agent runtime. Installs Python (uv) + Node toolchain before use.

```bash
cd .agent/hive
# First-run setup
make install-hooks      # git hooks
uv sync                 # python deps in core/
bun install             # if running the frontend dashboard

# Invoke
make help               # list targets
# or run the frontend
make frontend-dev
```

Best when the problem is large and parallelizable (e.g., "add DPDPA compliance
report export in PDF, XLSX, and CSV, with unit tests for each formatter").

## Agentsys (`.agent/agentsys/`)

Modular orchestration system. Plain-text output enforcement, no emoji/ASCII art,
strict PR discipline. Its `CLAUDE.md` encodes stricter rules than vanilla Claude
Code.

```bash
cd .agent/agentsys
node bin/cli.js --help
```

Use when you want the orchestration layer to enforce: every PR requires tests,
no direct pushes to main, no silent failures.

## Game Studios (`.agent/game-studios/`)

Indie-game-dev subagent architecture (48 agents). **Not relevant to this project.**
Kept in tree because the user requested the full set — ignore unless pivoting to
game development.

## Refreshing from upstream

`.agent/` is gitignored. When upstream repos update, refresh manually:

```bash
# Antigravity skills (biggest, ~200MB)
rm -rf /tmp/antigravity && git clone --depth=1 \
  https://github.com/sickn33/antigravity-awesome-skills.git /tmp/antigravity
rm -rf /tmp/antigravity/.git
rm -rf .agent/skills && cp -R /tmp/antigravity .agent/skills

# Hive
rm -rf /tmp/hive && git clone --depth=1 \
  https://github.com/aden-hive/hive.git /tmp/hive
rm -rf /tmp/hive/.git
rm -rf .agent/hive && cp -R /tmp/hive .agent/hive

# Agentsys
rm -rf /tmp/agentsys && git clone --depth=1 \
  https://github.com/agent-sh/agentsys.git /tmp/agentsys
rm -rf /tmp/agentsys/.git
rm -rf .agent/agentsys && cp -R /tmp/agentsys .agent/agentsys

# Game-studios
rm -rf /tmp/game-studios && git clone --depth=1 \
  https://github.com/Donchitos/Claude-Code-Game-Studios.git /tmp/game-studios
rm -rf /tmp/game-studios/.git
rm -rf .agent/game-studios && cp -R /tmp/game-studios .agent/game-studios
```

Each subsystem's own upstream repo owns updates — don't edit `.agent/*` in place.

## GSD — Get Shit Done (`npm: get-shit-done-cc`)

Spec-driven dev system by TÂCHES. Installs 73 `/gsd-*` commands + agents + hooks
into `.claude/` (local to this repo). Already installed for this project.

```bash
# Refresh to latest
npx -y get-shit-done-cc@latest --claude --local
```

Key commands:
- `/gsd-new-project` — scaffold a fresh spec-driven project
- `/gsd-quick` — one-shot task in GSD's context engineering framework
- `/gsd-plan-phase` / `/gsd-execute-phase` / `/gsd-verify-work` — phased spec loop

Hooks added to `.claude/settings.json`: update check, context monitor, prompt
injection guard, read-before-edit guard. Opt-in hooks (workflow guard, commit
validation, session state, phase boundary) are OFF by default.

## YOLO mode — `claude --dangerously-skip-permissions`

Convenience wrapper at `scripts/dev/claude-yolo.sh`. Prints a banner + 10s
cancel window, then execs Claude Code with all permission prompts disabled.

```bash
./scripts/dev/claude-yolo.sh
```

**Never run against main or a checkout with production credentials.**
Intended for: disposable worktrees, throwaway branches, sandboxed containers.
