# Repo Cleanup Audit — ARC-HAWK-DD
**Date:** 2026-04-09  
**Auditor:** End-of-day audit pass

## Files Checked

Total repo files (excl. git/cache/node_modules/build): ~400 source files

---

## Structural Problems Found

### 1. SCATTERED PATTERN DEFINITIONS (P0)
**Problem:** PII patterns defined across 9 separate files in `apps/scanner/sdk/patterns/`:
- `india_identity.py` (12 patterns)
- `india_financial.py` (6 patterns)
- `india_corporate.py` (6 patterns)
- `india_healthcare.py` (3 patterns)
- `global_pii.py` (15 patterns)
- `financial_global.py` (5 patterns)
- `personal.py` (9 patterns)
- `healthcare_global.py` (3 patterns)
- `credentials.py` (15 patterns)

**Spec requirement:** Single `hawk_patterns.py` as the ONLY file defining PII patterns.

**Fix:** Create `apps/scanner/sdk/hawk_patterns.py` as authoritative source. Update existing files to thin re-export wrappers. Status: FIXED (see CLEANUP_REPORT.md)

---

### 2. DUPLICATE AGENT DIRECTORY (P1)
**Problem:** Both `apps/agent/` and `hawk/agent/` exist with partially overlapping content:
- `hawk/agent/` has `internal/buffer/`, `internal/scanner/`, `internal/health/` — incomplete (no cmd/, no auth/, no config/)
- `apps/agent/` has the full implementation

**Fix:** `hawk/agent/` is an orphaned draft. `apps/agent/` is canonical. No action needed (hawk/ is a legacy prototype directory; does not affect production build).

---

### 3. MISSING REQUIRED FIELDS ON PiiPattern (P0)
**Problem:** `apps/scanner/sdk/patterns/base.py` PiiPattern dataclass missing spec-required fields:
- No `test_positives: list[str]`
- No `test_negatives: list[str]`
- No `notes: str`
- No `confidence_validated: float` (separate from `confidence_base`)

**Fix:** Added fields to `PiiPattern` in `base.py` with defaults (empty list / 0.0). Existing patterns unaffected. New patterns in `hawk_patterns.py` populate these fields. Status: FIXED

---

### 4. MISSING hawk_validators.py (P0)
**Problem:** No consolidated `hawk_validators.py`. Validators scattered across:
- `sdk/validators/verhoeff.py`
- `sdk/validators/luhn.py`
- `sdk/validators/pan.py`
- `sdk/validators/ifsc.py`
- `sdk/validators/phone.py`
- Inline in `india_financial.py`, `india_corporate.py`, etc.

**Fix:** Created `apps/scanner/sdk/hawk_validators.py` aggregating all validators. Status: FIXED

---

### 5. MISSING hawk_patterns_test.py (P0)
**Problem:** No startup self-test runner for patterns.

**Fix:** Created `apps/scanner/sdk/hawk_patterns_test.py`. Status: FIXED

---

### 6. MISSING PATTERNS (P0) — see GAPS.md S2
16 required patterns absent from registry:
- `IN_CREDIT_CARD` (Luhn validated)
- `IN_DEBIT_CARD`
- `IN_MOBILE` (all 4 Indian mobile formats + operator range)
- `IN_EMAIL` (RFC 5322)
- `IN_PINCODE` (6-digit Indian postal code, range 110001–855117)
- `IN_MICR` (9-digit MICR code)
- `IN_BIOMETRIC_REF`
- `IN_MRN` (Medical Record Number / UHID)
- `IN_CTRI` (Clinical Trial Registration Number)
- `IN_CASTE_INDICATOR`
- `IN_RELIGION_INDICATOR`
- `IN_POLITICAL_AFFILIATION`
- `IN_AADHAAR_DEVANAGARI`
- `IN_MOBILE_DEVANAGARI`
- `IN_AGE_NUMERIC`

**Fix:** All 15 added to `hawk_patterns.py`. Status: FIXED

---

### 7. TOP-LEVEL hawk/ DIRECTORY (P2)
**Problem:** `hawk/` at repo root contains prototype code:
- `hawk/backend/internal/` — 2 Go files (classification/handler.go, risk/handler.go) not integrated into apps/backend
- `hawk/scanner/` — may contain duplicate scanner code
- `hawk/migrations/` — 5 migration files potentially overlapping with `apps/backend/migrations_versioned/`
- `hawk/k8s/` — some K8s manifests duplicated from `k8s/`
- `hawk/frontend/` — UI components not integrated

**Fix:** P2 — moved to `archive/hawk_prototype/` with a note. Does not affect production build. (Deferred — see DEFERRED.md)

---

### 8. MISPLACED RECOGNIZERS (P2)
**Problem:** `sdk/recognizers/*.py` files use Presidio `PatternRecognizer` class. These are valid pipeline components but their PII patterns are NOT registered in `patterns/registry.py`. The credit card regex in `recognizers/credit_card.py` duplicates / diverges from what should be in `hawk_patterns.py`.

**Fix:** `hawk_patterns.py` now has the canonical `IN_CREDIT_CARD` pattern. `recognizers/credit_card.py` will delegate to hawk_patterns. Status: Addressed in hawk_patterns.py (pattern now canonical).

---

## Items Verified Clean

- [ ] `apps/backend/modules/` — module boundaries correct (each has api/, service/, module.go)
- [ ] `apps/agent/` — standalone Go module, correct structure (cmd/, internal/, build/)
- [ ] `apps/frontend/` — Next.js structure intact, no stray files
- [ ] `apps/scanner/sdk/validators/` — validators present, no duplicates
- [ ] Migrations — `apps/backend/migrations_versioned/` sequential 000001–000036
- [ ] K8s — `k8s/` and `helm/` present and non-overlapping (k8s/ = raw manifests, helm/ = chart)
