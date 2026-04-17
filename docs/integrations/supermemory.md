# Supermemory.ai integration

ARC-HAWK-DD uses [supermemory.ai](https://supermemory.ai) as a memory & hybrid-search
layer. Scan completions, high-severity findings, and ad-hoc queries go through it so
the dashboard (and AI agents) can answer things like *"what PII did we find in S3 last
week?"* without re-scanning.

## Tier

Free tier is sufficient for most self-hosted deployments:

| Quota | Free |
|---|---|
| Content ingested per month | 1,000,000 tokens (~750k words) |
| Search queries per month | 10,000 |
| Storage | unlimited |
| Multi-modal extraction | included |
| Connectors (Gmail, S3, Drive) | NOT included — Scale tier only |
| Claude Code / Cursor plugin | NOT included — Pro tier; self-host the MCP instead |

## Setup

### 1. Get an API key

Sign up at <https://app.supermemory.ai>, create a key (`sm_...` format).

### 2. Local (dev) — `.env`

In `apps/backend/.env`:

```bash
SUPERMEMORY_ENABLED=true
SUPERMEMORY_API_KEY=sm_your_key_here
SUPERMEMORY_API_URL=https://api.supermemory.ai
```

`.env` is gitignored. The recorder is a NoOp when any of these are missing —
memory calls never break a scan.

### 3. Docker-compose — host env

Same three vars pass through from `.env` into the backend container (see
`docker-compose.yml` → `backend.environment`). No code changes needed.

### 4. Kubernetes (Helm) — secret

```bash
kubectl create secret generic supermemory \
  --from-literal=api-key=sm_your_key_here

helm upgrade arc-hawk-dd ./helm/arc-hawk-dd \
  --set supermemory.enabled=true
```

The chart already maps the secret into the backend deployment via `valueFrom`.

### 5. Verify

```bash
# 1. Backend logs: look for this line on startup
#    🧠 Memory recorder: supermemory.ai ready

# 2. Status endpoint
curl -s http://localhost:8080/api/v1/memory/status
# → {"enabled":true,"provider":"supermemory.ai"}

# 3. After a scan completes, search for it
curl -s -X POST http://localhost:8080/api/v1/memory/search \
  -H 'Content-Type: application/json' \
  -d '{"q":"scan","limit":5}'
```

## What gets stored

**On every scan completion (status→"completed"):**

One narrow summary per scan (see `modules/memory/service/service.go`
`RecordScanCompletion`). Roughly 80 tokens of content + metadata. At 100 scans/day
that's ~240k tokens/month — well under the free-tier 1M cap.

```text
Scan "nightly-s3" (id=..., tenant=...) finished at 2026-04-17T19:00Z.
42 assets, 1,337 findings (12 critical, 89 high).
Sources: postgresql,s3. PII types: PAN,AADHAAR,EMAIL. Duration: 14382ms.
```

**Explicitly NOT stored:**

- Raw finding values (PII values never leave the cluster)
- Per-finding details by default (can be added via `RecordFinding`, but budget-aware)

## API surface

`apps/backend/modules/memory/`:

| Route | Method | Purpose |
|---|---|---|
| `/api/v1/memory/status` | GET | Returns `{enabled, provider}` — for the UI to show the memory badge |
| `/api/v1/memory/search` | POST | `{q, limit}` → forwards to `/v3/search` — hybrid RAG + memory |

Internal (Go) API via `interfaces.MemoryRecorder`:

```go
type MemoryRecorder interface {
    RecordScanCompletion(ctx, ScanSummarySnapshot) error
    Enabled() bool
}
```

Other modules call it via `deps.MemoryRecorder`. A `NoOpMemoryRecorder` is injected
when the backend is disabled, so call sites don't need nil checks.

## MCP server (for Claude Code / other AI agents)

Supermemory runs a hosted MCP endpoint at `https://mcp.supermemory.ai/mcp`. Wired
into this project at `.claude/settings.local.json` (gitignored). Every Claude Code
session in this repo has memory tools available: `addDocument`, `search`, etc.

## Rotation

When rotating the API key:

1. Create a new key at <https://app.supermemory.ai>
2. Update `apps/backend/.env`, restart the backend container
3. Update `.claude/settings.local.json` `Authorization` header
4. Update the Kubernetes secret (`kubectl create secret generic supermemory --from-literal=api-key=NEW --dry-run=client -o yaml | kubectl apply -f -`)
5. Revoke the old key at <https://app.supermemory.ai>

## Why not self-host the engine?

Supermemory's memory engine is closed-source (hosted at `api.supermemory.ai`). Only
the SDKs, dashboard, and MCP server are open-source. If full data sovereignty is
required, swap `MemoryRecorder` for a [Zep](https://github.com/getzep/zep) or
[Mem0](https://github.com/mem0ai/mem0) implementation — the interface at
`modules/shared/interfaces/memory_recorder.go` is provider-agnostic.
