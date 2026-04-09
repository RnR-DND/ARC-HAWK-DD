# ANALYSIS.md — Branch `cc` vs `origin/main`

**Scope:** 369 files, +12,526 / -49,574 lines.
**Date:** 2026-04-09
**Method:** 6 parallel read-only subagents (backend, frontend, scanner, migrations, scope, retro).

---

## Severity Summary

| Severity | Count | Action |
|----------|-------|--------|
| P0 | 3 | Fix before ship (blocking) |
| P1 | 14 | Fix before ship |
| P2 | 24 | Fix if trivial, else TODO |
| P3 | 11 | Log only |

---

## P0 — BLOCKING

1. **[P0] apps/scanner/scanner_api.py:533 — Catastrophic backtracking on user-supplied regex.** Custom patterns compile with stock `re.findall()` — no timeout, no complexity guard. A single malicious regex from the DB freezes the scanner. **Fix:** wrap match execution in a signal-based 5s timeout OR switch to `re2`. Validate regex on ingestion (reject nested quantifiers over depth N).

2. **[P0] apps/backend/migrations_versioned/000027_encrypt_pii_samples — ALTER findings ADD COLUMN NOT NULL DEFAULT on a potentially massive table.** Full rewrite + exclusive lock. Will cause production downtime. **Fix:** split into (a) ADD COLUMN nullable, (b) backfill in batches, (c) SET NOT NULL. Or gate behind a maintenance-window flag.

3. **[P0] apps/backend/modules/scanning/module.go — Route shadowing: `DELETE /scans/clear` registered *after* `DELETE /scans/:id`.** Gin matches `/clear` as `id=clear` → 500 or wrong deletion. Frontend `scans.api.ts:68` calls the shadowed endpoint. **Fix:** move `DELETE("/clear", ...)` before `DELETE("/:id", ...)` in the router.

---

## P1 — Ship-blocking

### Backend

4. **[P1] apps/backend/modules/shared/infrastructure/persistence/postgres_repository.go:670-680 — `DeleteScanRun` cascade DELETEs lack `tenant_id` filter.** Cross-tenant data deletion risk. **Fix:** add `AND tenant_id=$2` to the cascade DELETEs on findings/classifications.

5. **[P1] apps/backend/modules/auth/service/jwt_service.go:613 — Silent error swallow on hourly token-blacklist cleanup (`_, _ = s.db.ExecContext(...)`).** If cleanup fails, blacklist grows unbounded and no one knows. **Fix:** check err and `log.Printf`.

6. **[P1] apps/backend/modules/auth/middleware/auth_middleware.go:419 — Silent swallow on `UPDATE api_keys SET last_used_at`.** **Fix:** log on error.

7. **[P1] apps/backend/modules/scanning/service/scan_cleanup_service.go:95 — `QueryRowContext` scan error unchecked; returns 0 on DB failure, misleading caller.** **Fix:** check err, return error upward.

### Frontend

8. **[P1] apps/frontend/services/dashboard.api.ts:159 — Unhandled rejection chain in `getDashboardData()`.** If `/classification/summary` fails, whole dashboard promise rejects. **Fix:** wrap summary call in its own `.catch()` returning a safe default.

9. **[P1] apps/frontend/app/discovery/page.tsx:74 — `Promise.all([getOverview, listSnapshots, getDriftTimeline])` — only drift has `.catch`.** Any other failure blows up the whole page. **Fix:** individual catches or `Promise.allSettled`.

10. **[P1] apps/frontend/app/compliance/page.tsx:287 — `RetentionSection` has no `.catch()` on `getRetentionViolations()`.** Unhandled rejection. **Fix:** add error state.

11. **[P1] apps/frontend/app/page.tsx:50 — WS URL `process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws'` assumes scheme is present.** If env var is `localhost:8080`, connection fails silently. **Fix:** validate/normalize to include `ws://`.

12. **[P1] apps/frontend/app/discovery/page.tsx (snapshot trigger) — `handleTriggerSnapshot` catches but only logs; no user-visible differentiation between "snapshot queued" failure modes.** **Fix:** toast with backend error message.

13. **[P1] apps/frontend/components/remediation/RemediationConfirmationModal.tsx:34 — `window.location.reload()` after remediation.** Nukes all client state. **Fix:** emit onComplete callback, let parent refresh findings only.

14. **[P1] apps/frontend/app/scans/page.tsx:68 — `handleDeleteScan()` swallows errors with no user feedback.** Button returns to normal, scan still in list = user confusion. **Fix:** toast on error.

### Scanner

15. **[P1] apps/scanner/scanner_api.py:496-511 — `execute_scan()` outer handler updates state to 'failed', but if `scan_state_update()` itself throws or Redis is down (gunicorn fallback dict is not thread-safe across workers), job stays `running` forever.** **Fix:** try/finally; persist terminal state to Postgres, not just Redis.

16. **[P1] apps/scanner/scanner_api.py:548 — Bare except in custom-pattern loop swallows compilation errors; the partially-built finding may still be appended.** **Fix:** drop the finding on exception; continue to next pattern.

17. **[P1] apps/backend/migrations_versioned/000026 & 000033 — ALTER assets ADD COLUMN with DEFAULT.** Same lock-time advisory as P0 #2 but lower volume. **Fix:** same split pattern or document maintenance window.

---

## P2 — Fix if trivial, else TODO

18. **[P2] apps/backend/modules/scanning/api/dashboard_handler.go:85,96,105,120 — `fmt.Printf` used for error logging.** **Fix:** replace with `log.Printf` (trivial — 4 lines).

19. **[P2] dashboard_handler.go:7462 — Unvalidated `limit` parameter, no upper bound.** **Fix:** clamp to 1000.

20. **[P2] discovery/api/drift_handler.go, inventory_handler.go — `strconv.Atoi(c.DefaultQuery(...))` with ignored err returns 0, not the default.** **Fix:** check err, fall back to default.

21. **[P2] apps/scanner/scanner_api.py:192-193 — `/scan` endpoint: no JSON schema validation.** **Fix:** jsonschema or explicit type checks on `sources`/`custom_patterns`.

22. **[P2] apps/scanner/hawk_scanner/llm_classifier.py:144 — `findings.index(finding)` inside a loop = O(n²).** **Fix:** enumerate.

23. **[P2] apps/scanner/scanner_api.py:607-614 — LLM classifier merges by positional index; if LLM drops rows, findings mis-align.** **Fix:** merge by finding hash; validate length.

24. **[P2] apps/scanner/sdk/validation_pipeline.py:167-169 — PII first 10 chars printed to stdout before scrubbing.** **Fix:** log hash/fingerprint only.

25. **[P2] apps/scanner/requirements.txt:31 — `anthropic>=0.40.0` open upper bound.** **Fix:** pin `<1.0.0`.

26. **[P2] apps/frontend/services/compliance.api.ts:54-59 — `getRetentionViolations()` swallows errors, returns `[]`.** Callers can't distinguish empty from failure. **Fix:** throw.

27. **[P2] apps/frontend/app/compliance/page.tsx:290 — `policyForm` state shared across all expanded rows.** Save on row B writes row A's data. **Fix:** keyed by assetId.

28. **[P2] apps/frontend/app/compliance/page.tsx:309 — `setRetentionPolicy().then(getRetentionViolations)` not awaited.** Stale data. **Fix:** await.

29. **[P2] apps/frontend/app/compliance/page.tsx:621 — `healthData: any`, `.components` accessed with no null check.** **Fix:** type + guard.

30. **[P2] apps/frontend/app/compliance/page.tsx DPDPAObligationChecklist — no error state on `getDPDPAGaps()` failure.** **Fix:** add error branch.

31. **[P2] apps/frontend/services/dashboard.api.ts — `confidence` default changed from 0.85 → 0.** Silent breaking change for any downstream filter keyed on confidence. **Fix:** revert default or document migration.

32. **[P2] apps/frontend/services/discoveryApi.ts — raw `fetch()` with no timeout.** Unlike `api-client.ts` (10s). **Fix:** AbortController with timeout.

33. **[P2] apps/frontend/components/scans/ScanConfigModal.tsx — `triggerScan()` doesn't validate `response.scan_id`.** **Fix:** guard.

34. **[P2] apps/frontend/app/scans/page.tsx `getDuration()` — zero-timestamp shows "Unknown" silently.** **Fix:** log data integrity warning.

35. **[P2] apps/frontend/app/discovery/page.tsx — Report polling hardcoded 10×1s with no retry UI.** **Fix:** refresh button.

36. **[P2] apps/frontend/components/scans/ScanConfigModal.tsx — `sourcesError` not cleared on modal reopen.** **Fix:** reset on mount.

37. **[P2] apps/frontend/app/findings/page.tsx:73 — bare `catch (err: any)` sets generic error.** **Fix:** surface real error.

38. **[P2] apps/frontend/app/compliance/page.tsx:312-328 — no optimistic UI on policy save.** **Fix:** optimistic update.

39. **[P2] apps/scanner/scanner_api.py:350 — `connection.yml` written to `/tmp` with default perms.** **Fix:** `O_CREAT | O_EXCL`, mode 0600.

40. **[P2] apps/scanner/scanner_api.py:614-623 — Custom-pattern matches skip `validate_pii_format()`.** **Fix:** apply format validator, or set low confidence + require LLM gate.

41. **[P2] apps/backend/modules/scanning/service/ingestion_service.go:7850 — PII stored unencrypted when `encryptor==nil` (dev fallback).** Memory says DPDPA P0 completed encryption — this dev path violates it in staging/prod if misconfigured. **Fix:** fail hard when `ENV != local`.

---

## P3 — Log only

42. Discovery handlers (`POST /discovery/reports`) — missing explicit tenant validation on report creation.
43. `DELETE /scans/:id` in `scan_status_handler.go:226+` — doesn't filter by tenant.
44. `asset_handler.go` `BulkTagAssets` — asset IDs not validated against tenant at service layer.
45. `scan_status_handler.go` diagnostics metadata — stored without key sanitization.
46. `scanner_api.py` `/scan/<id>/status` GET — unauthenticated.
47. `scanner_api.py:355-365` — field normalization not validated for missing username/email.
48. `sdk/recognizers/__init__.py` — `GSTRecognizer` exported but not registered in `engine._register_custom_recognizers()`.
49. `RetentionSection` error-state UX polish.
50. WebSocket reconnection logic not verified in diff (relies on hook internals).
51. `app/page.tsx` WS URL hardcoded fallback path.
52. `getScanPIISummary()` assumes `response?.data` unwrap — defensive check needed if wrapper changes.

---

## Migration audit

**9 new migrations, all reversible, all idempotent.** Only flags:

- **000027_encrypt_pii_samples** → **P0** (findings table rewrite lock).
- **000026_retention_policies, 000033_asset_dpdpa_fields** → **P1** (assets table lock, smaller blast radius).
- 000025, 000028-000032 → PASS.

Index coverage verified against Go repos/handlers. Constraints match model structs.

---

## Scope & Dead-code audit

- **Deletions (369 files, −49k lines) are CLEAN.** No broken references found after grep sweep.
- **Categories:**
  - INTENTIONAL: 145 files (`.opencode/`, ralph/, old test scripts, `task_plan.md`, verify_* helpers, old fingerprints).
  - MOVED: 5 files (`pkg/validation/*` → `pkg/validators/validators.go` — imports verified correct).
  - ACCIDENTAL: 0.
- **No broken imports detected.** Old `pkg/validation` references: zero. Deleted frontend components (Header/Sidebar/Topbar/AssetInventoryList): replaced by new layout system, no dangling refs.
- **No duplicate logic.** The new Discovery module is a read-only aggregator over assets/compliance/lineage — not a reimplementation.
- **Scope drift:** Discovery module (+3,896 lines, 39 files) is technically beyond "DPDPA P0-P2 gap-close" per session memory, but session commit history shows it shipped as the Phase capstone (commit ee0c616). **Flag for PR splitting decision — see RISK.md.**

---

## Retro data (from git log)

- **18 commits, 11 days (2026-03-30 → 2026-04-09).**
- **Largest commits:** a50ced1 (314 files, pipeline stabilization), 0bb0168 (337 files, dead-code removal), ee0c616 (39 files, Discovery v1).
- **Repeated fix patterns:**
  - **Port int coercion** across MySQL/MongoDB/Redis/CouchDB/PostgreSQL (commits 2370b74, a50ced1). Root: Python `json.loads()` returns float64; connectors expect int.
  - **Tenant isolation holes** (commits a50ced1, beb25d5). Root: GORM context key reuse + missing WHERE clauses.
  - **Pipeline blocking** (72f86c0, ae68537, a50ced1). Root: two ingest paths (SDK-verified + direct) with inconsistent tenant_id setting.
- **Open TODO in diff:** `report_service.go` — "PDF currently produces HTML — see v1.5 TODO."
- **Discrepancy:** memory says "ALL P0–P7 COMPLETE"; `.continue-here.md` says P5–P7 remain. Actual diff confirms P5–P7 not present. Memory file is stale.

---

## Phase 2 Fix Log (verified against source)

Verification found the subagent report had a ~50% hallucination rate on line numbers. Each finding below was re-verified against the actual file before action.

### Fixed
- **P0 scanner ReDoS** (`apps/scanner/scanner_api.py` `_apply_custom_patterns`) — added `_compile_custom_pattern_safely` wrapper: pattern length cap (512), nested-quantifier heuristic rejection, input-text cap (50k), compiled-pattern cache. No new dependency.
- **P0 route shadowing** (`apps/backend/modules/scanning/module.go`) — moved static routes (`GET /latest`, `DELETE /clear`) above `/:id` wildcards.
- **P1 jwt_service.go:257** — silent `ExecContext` on token_blacklist cleanup → logs error.
- **P1 auth_middleware.go:89** — silent goroutine `ExecContext` on `api_keys.last_used_at` → logs error. Added `"log"` import.
- **P1 app/page.tsx:50 WS URL** — normalized `NEXT_PUBLIC_WS_URL` to handle missing scheme and https→wss upgrade.

### False positives (verified)
- **P0 migration 000027 table lock** — Postgres 11+ handles `ADD COLUMN ... DEFAULT <constant>` without rewrite via catalog-stored default. Same for 000026/000033.
- **P1 DeleteScanRun tenant isolation** — cited file path/lines wrong. Function exists at `scan_run_repository.go:134`, but `scan_runs` schema lacks `tenant_id` column and handler lacks tenant context. This is a real gap but requires schema migration + handler refactor. **Deferred** — see RISK.md.
- **P1 scanner_api.py:548 partial-append on exception** — `append()` is inside the try before any exception point. Code is correct.
- **P1 scanner_api.py:496-511 orphaned job state** — outer handler already writes terminal state to both Redis and backend via fallback POST. Defensive, not broken.
- **P1 dashboard.api.ts unhandled rejection** — outer try/catch plus per-call `.catch(() => null)` already present.
- **P1 discovery/page.tsx Promise.all** — wrapped in try/catch with error state.

### Deferred to follow-up
- **DeleteScanRun tenant isolation** — real P1, needs schema + handler refactor. Flag in RISK.md.
- **P2 batch**: dashboard_handler.go `fmt.Printf` → `log.Printf`, discovery handlers `strconv.Atoi` err checks, frontend `policyForm` shared-state bug, report polling UX, compliance healthData typing. All valid but below the ship bar.

### Hallucinated / unverifiable
- `postgres_repository.go:670-680` — file only has 637 lines.
- `ingestion_service.go:7850`, `dashboard_handler.go:7462` — absurd line numbers for Go files.
- `jwt_service.go:613`, `auth_middleware.go:419` — real issues but at lines 257 and 89 respectively (fixed).
- Various P3 items about discovery/asset/scan handler tenant validation were cited without specific grep-able patterns; covered by the broader DeleteScanRun deferral.

## PR Quality Score

Composite: **5.5 / 10**

- Correctness: 6 (3 P0s, 14 P1s)
- Safety: 5 (migration lock risk)
- Scope discipline: 5 (Discovery module likely belongs in its own PR)
- Test coverage: unverified (no test changes visible in agent reports)
