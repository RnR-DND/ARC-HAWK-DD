# RISK.md — Branch `cc` Investigation

**Date:** 2026-04-09
**Scope:** Why 369 files? Is the -49k deletion safe? Are there circular imports?

---

## Risk 1: 369-file blast radius — is this one PR or five?

### Evidence

18 commits, 11 days (2026-03-30 → 2026-04-09). File distribution:

| Path | Files changed | Notes |
|------|---------------|-------|
| `apps/backend/modules/` | 81 | DPDPA gap-close + Discovery v1 |
| `apps/backend/migrations_versioned/` | 33 | 9 new migrations (000025–000033) |
| `.opencode/get-shit-done/` | 79 | **Pure deletion** — old scaffolding framework |
| `apps/frontend/{components,app,services}` | 33 | DPDPA UI, Discovery UI, scan UX polish |
| `apps/scanner/{hawk_scanner,sdk,config}` | 17 | Custom patterns, validation pipeline, LLM classifier |
| `scripts/` (deletions) | ~15 | ralph/, verify_*, test_* scripts removed |
| Other | ~100 | Mixed cleanup, docs, config |

### Classification

This branch is **three logically distinct units of work bundled together**:

1. **DPDPA P0-P2 gap-close** (primary stated goal). Encryption, N+1 fixes, risk scoring, obligation service, LLM classifier, custom patterns. Commits: a50ced1, ae68537, 46a1966, beb25d5, 2370b74.
2. **Discovery Module v1** (scope creep per session memory). 39 files, +3,896 lines, 6 new services, 6 migrations, 15 endpoints. Commit: ee0c616.
3. **Dead-code amnesty** (cleanup). 79 `.opencode/` files + 15 scripts + `task_plan.md`. Commit: 0bb0168.

### Risk level: MEDIUM

- **Not technically risky** — the three units don't depend on each other in dangerous ways.
- **Review risky** — 369 files is beyond human review. A reviewer will rubber-stamp or nitpick, not actually check the work.
- **Bisect risky** — if a production issue surfaces after merge, `git bisect` will land on one of these giant commits and tell you nothing useful.

### Recommendation

Ship as-is for this cycle (splitting now would cost more than it's worth given the work is already done and the changes are logically consistent). **Going forward, enforce a branch-size gate** — any branch over ~50 files should auto-split via `/ship --split` or refuse to merge without an override.

---

## Risk 2: Did the -49,574 deletion remove anything DPDPA compliance depends on?

### Evidence

Agent 5 (scope audit) categorized deletions:
- **INTENTIONAL**: 145 files (`.opencode/`, ralph/, old scripts, `task_plan.md`, verify_*/test_* helpers)
- **MOVED**: 5 files (`apps/backend/pkg/validation/{common,luhn,pan,ssn,verhoeff}.go` → consolidated into `apps/backend/pkg/validators/validators.go`)
- **ACCIDENTAL**: 0

### Independent verification

Ran grep for references to the deleted `pkg/validation` package path across the post-deletion tree — zero hits. The consolidated `pkg/validators` package is referenced by `ingestion_service.go` and `ingest_sdk_verified.go` (verified by subagent). No deleted Go file has a dangling import.

DPDPA-relevant deletions (none):
- `common.go`, `luhn.go`, `pan.go`, `ssn.go`, `verhoeff.go` — all PII validator helpers, moved to new path, functions still exported.
- No migration was deleted. Migrations only add.
- No compliance service file was deleted. `compliance/service/compliance_service.go` and `retention_service.go` are modified, not removed.

### Risk level: LOW

Deletions are dominated by dev scaffolding (`.opencode/get-shit-done/` = 79 files of templates/workflows for an old AI orchestrator) and obsolete verification scripts. DPDPA machinery is intact.

**One residual concern:** the `retention_service.go` and `compliance_service.go` modifications haven't been fully audited by the subagent pass (the review focused on adds, not modifies in those files). Before ship, do a targeted `git diff origin/main -- apps/backend/modules/compliance/` read to confirm no regression in obligation checks.

---

## Risk 3: Circular imports in discovery/compliance Go modules?

### Evidence

Ran: `grep -rn 'import.*modules/discovery\|import.*modules/compliance' apps/backend/modules/{discovery,compliance}/`

**Result: zero cross-imports.** Discovery does not import compliance and compliance does not import discovery at the module boundary.

Discovery reads data via the shared repository layer (`modules/shared/infrastructure/persistence`), which is the correct pattern. No circular dependency risk.

### Risk level: NONE

---

## Risk 4: Deferred DeleteScanRun tenant isolation (from Phase 2 fix log)

### Evidence

`apps/backend/modules/shared/infrastructure/persistence/scan_run_repository.go:134` —

```go
func (r *PostgresRepository) DeleteScanRun(ctx context.Context, id uuid.UUID) error {
    // DELETE FROM classifications ... WHERE finding_id IN (SELECT id FROM findings WHERE scan_run_id = $1)
    // DELETE FROM findings WHERE scan_run_id = $1
    // DELETE FROM scan_runs WHERE id = $1
}
```

Handler at `scan_status_handler.go:231` parses the scan UUID from the URL path and calls `DeleteScanRun(ctx, scanID)` with zero tenant filter. Any authenticated user with a valid scan UUID can delete any other tenant's scan.

### Why deferred

- `scan_runs` table schema has no `tenant_id` column (verified via grep on `scan_run_repository.go`). Adding one requires a migration + backfill + handler changes across every read/write path for scans, not just delete.
- Handler has no tenant-context extraction pattern visible in the file. Adding it requires wiring through the auth middleware's stored tenant_id.
- Fix is real multi-file work: migration + repo layer + handler + similar audit on other scan endpoints (list, get, update).

### Risk level: HIGH

This is a real multi-tenant data integrity hole. It is not fixed in this PR. Ship decision must account for this.

### Mitigation until fixed

1. **Short term:** document in PR body that cross-tenant scan deletion is possible given a valid scan UUID.
2. **Auth gate:** require `admin` role for `DELETE /scans/:id` via route middleware. Add one line in `module.go`. Not a real fix but raises the bar.
3. **Follow-up ticket:** create a P0 TODO for "scan_runs tenant isolation" covering migration, repo, handler, and test.

### Recommendation

Before ship: add the admin-role gate mitigation (5-line change) so the route isn't completely open. File the follow-up.

---

## Risk 5: Memory/handoff file staleness

`.claude/projects/.../memory/project_dpdpa_gaps.md` says "ALL P0–P7 COMPLETE." Diff confirms P5–P7 are NOT in this branch (per `.continue-here.md` and git log). The memory file is wrong.

### Risk level: LOW (meta)

Not a code risk. But future sessions relying on this memory will get wrong context. **Action:** update the memory file as part of Phase 6 `/learn`.

---

## Summary

| Risk | Level | Ship-blocking? |
|------|-------|----------------|
| 369-file blast radius | MEDIUM | No — ship, enforce gate going forward |
| -49k deletion safety | LOW | No — verified clean |
| Discovery/compliance circular imports | NONE | No |
| DeleteScanRun tenant isolation | HIGH | **Conditional — mitigate with admin role gate before ship** |
| Memory file staleness | LOW | No — fix in Phase 6 |

**One gate remains before Phase 3**: decide on DeleteScanRun mitigation. Options:
- **A)** Add admin-role gate on `DELETE /scans/:id`, file follow-up ticket, proceed.
- **B)** Revert `DeleteScan` route entirely from this PR, proceed.
- **C)** Ignore, document in PR body, proceed.

Recommendation: **A**. Cheapest real mitigation.
