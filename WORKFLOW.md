# WORKFLOW.md — Unified Agentic Execution Protocol

> **This document defines how all four productivity tools chain together for EVERY task in ARC-Hawk.**

---

## 🧰 Tool Stack

| Tool | Location | Purpose |
|------|----------|---------|
| **Antigravity Skills** | `.agent/skills/` | 860+ expert markdown skills loaded via `@skill-name` |
| **GSD (Get Shit Done)** | `.claude/commands/gsd/`, `.gemini/`, `.codex/` | Spec-driven development: discuss → plan → execute → verify |
| **Ralph** | `scripts/ralph/` | Autonomous PRD iteration loop with `prd.json` + `progress.txt` |
| **Hive** | `.agent/hive/` | Goal-driven multi-agent framework with MCP tools |
| **Supermemory** | `MCP Server` | Persistent cross-session memory layer |

---

## 🔄 Unified Protocol

### For ANY Task:

```
│  0. MEMORY → Load context from Supermemory                     │
│     use @supermemory-pro to retrieve past decisions & findings │
├─────────────────────────────────────────────────────────────────┤
│  1. SKILLS → Load relevant @skills for the domain              │
│     @golang-pro, @python-patterns, @nextjs-best-practices      │
│     @docker-expert, @api-security, @test-driven-development    │
├─────────────────────────────────────────────────────────────────┤
│  2. GSD → Structure the work                                   │
│     Small:  /gsd:quick "description"                           │
│     Medium: /gsd:plan-phase N                                  │
│     Large:  /gsd:new-project → discuss → plan → execute        │
├─────────────────────────────────────────────────────────────────┤
│  3. RALPH → Autonomous iteration                               │
│     Convert plan to prd.json → run ralph.sh                    │
│     Each iteration: fresh context, one story, commit, repeat   │
├─────────────────────────────────────────────────────────────────┤
│  4. HIVE → Multi-agent coordination (when needed)              │
│     /hive for goal-driven agent graph                          │
│     hive tui for interactive dashboard                         │
├─────────────────────────────────────────────────────────────────┤
│  5. VERIFY → Quality assurance                                 │
│     /gsd:verify-work N                                         │
│     Ralph progress.txt learnings                               │
│     Hive observability metrics                                 │
├─────────────────────────────────────────────────────────────────┤
│  6. COMPLETE → Archive and document                            │
│     /gsd:complete-milestone                                    │
│     Ralph auto-archive                                         │
│     **Save critical learnings to Supermemory**                 │
│     Update AGENTS.md with learnings                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📏 Task Size Guide

### ⚡ Small (< 5 min) — Bug fix, config change, small feature
```
1. Load @skill → 2. /gsd:quick → Done
```

### 🔧 Medium (5-30 min) — Feature, refactor, integration
```
1. Load @skills
2. /gsd:plan-phase N
3. /gsd:execute-phase N
4. /gsd:verify-work N
```

### 🏗️ Large (> 30 min) — New module, major feature, cross-component
```
1. Load @skills
2. /gsd:new-project (or /gsd:new-milestone)
3. /gsd:discuss-phase → /gsd:plan-phase → /gsd:execute-phase
4. Convert plans to prd.json for Ralph autonomous loop
5. ./scripts/ralph/ralph.sh [max_iterations]
6. /gsd:verify-work
7. /gsd:complete-milestone
```

### 🤖 Multi-Agent (complex orchestration)
```
1. /hive → Define goal in natural language
2. Hive auto-generates agent graph
3. hive tui → Monitor execution
4. Hive self-heals on failures
```

---

## 🎯 Skill Auto-Selection Map

| Working On | Load These Skills |
|-----------|------------------|
| Go backend | `@golang-pro`, `@api-design-principles`, `@test-driven-development` |
| Next.js frontend | `@nextjs-best-practices`, `@react-patterns`, `@typescript-expert` |
| Python scanner | `@python-patterns`, `@test-driven-development` |
| Docker/infra | `@docker-expert`, `@aws-serverless` |
| Security | `@api-security-best-practices`, `@vulnerability-scanner` |
| Architecture | `@senior-architect`, `@brainstorming`, `@c4-context` |
| Debugging | `@systematic-debugging`, `@test-fixing` |
| Memory/Context | `@supermemory-pro` |
| Documentation | `@doc-coauthoring`, `@writing-plans` |

---

## 🗂️ GSD Commands Reference

### Core Workflow
| Command | Purpose |
|---------|---------|
| `/gsd:new-project` | Initialize full spec (PROJECT.md, REQUIREMENTS.md, ROADMAP.md) |
| `/gsd:discuss-phase N` | Shape implementation details for phase N |
| `/gsd:plan-phase N` | Research + create atomic task plans |
| `/gsd:execute-phase N` | Run plans in waves (parallel where possible) |
| `/gsd:verify-work N` | Confirm feature works + spawn debug agents if needed |
| `/gsd:complete-milestone` | Archive milestone, tag release |
| `/gsd:quick` | Fast mode for ad-hoc tasks |

### Utilities
| Command | Purpose |
|---------|---------|
| `/gsd:map-codebase` | Analyze existing codebase (use for brownfield) |
| `/gsd:progress` | Show current progress |
| `/gsd:debug` | Debug an issue with specialized agents |
| `/gsd:settings` | Configure GSD behavior |
| `/gsd:health` | Check `.planning/` integrity |

---

## 🔁 Ralph Workflow

```bash
# 1. Create PRD (use /prd skill or manually create prd.json)
# 2. Run autonomous loop
./scripts/ralph/ralph.sh [max_iterations]

# Monitor progress
cat scripts/ralph/progress.txt
cat scripts/ralph/prd.json | jq '.userStories[] | {id, title, passes}'
```

### prd.json Structure
```json
{
  "projectName": "ARC-Hawk Feature",
  "branchName": "feature/my-feature",
  "userStories": [
    {
      "id": "STORY-001",
      "title": "Story title",
      "description": "What to implement",
      "priority": 1,
      "acceptanceCriteria": ["Criterion 1", "Criterion 2"],
      "passes": false
    }
  ]
}
```

---

## 🐝 Hive Quick Reference

```bash
# Build an agent (from any coding tool)
/hive

# Debug an agent
/hive-debugger

# Run interactively
cd .agent/hive && hive tui

# Run a specific agent
cd .agent/hive && hive run exports/agent_name --input '{"key": "value"}'
```

**Windows**: Use `quickstart.ps1` instead of `quickstart.sh`:
```powershell
cd .agent/hive
.\quickstart.ps1
```

---

## 📝 Key Files

| File | Purpose |
|------|---------|
| `WORKFLOW.md` | This file — unified protocol |
| `AGENTS.md` | AI agent guide + learnings |
| `gemini.md` | Project constitution |
| `task_plan.md` | Phase-based project plan |
| `.planning/` | GSD planning artifacts |
| `scripts/ralph/prd.json` | Ralph active PRD |
| `scripts/ralph/progress.txt` | Ralph iteration learnings |
| `.agent/skills/` | 860+ agentic skills |
| `.agent/hive/` | Hive framework |
