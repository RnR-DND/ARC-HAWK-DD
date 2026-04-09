# Session Retrospective — Branch `cc`

Date: 2026-04-09

## What Shipped

- Enterprise Data Discovery Module v1: risk scoring, spike detector, discovery heatmap
- Custom regex patterns engine with ReDoS protection and per-scan hot-reload
- DPDPA compliance expansion: obligation service, gap reports, retention policies, consent records, audit log chain
- 14 new scanner data sources: BigQuery, Redshift, Snowflake, Azure Blob, Kinesis, Salesforce, HubSpot, Jira, MS Teams, Avro, Parquet, PPTX, HTML, Email
- 6 security fixes (2x P0, 4x P1): ReDoS protection, Gin route ordering, admin gate on scan deletion, JWT token_blacklist cleanup, API key last_used_at tracking, WS URL normalization
- Helm chart, Grafana dashboards, 8+ Postgres migrations
- 6 bisectable commits pushed to origin/cc
- Documentation updated: README (v3.0.0), API.md (patterns and discovery endpoints), MIGRATION_GUIDE.md (v2.1 to v3.0 steps)

## What Broke (and Was Fixed)

- **JSX comment in ternary expression** caused a TypeScript error. Removed the comment.
- **Missing `net/http` import** after inline middleware used `http.StatusForbidden`. Caught at build time, added the import.
- **Migration 000027** was flagged as a P0 full-table rewrite lock. Turned out to be a false positive — Postgres 11+ stores constant defaults in the catalog with no lock required.
- **Frontend test runner mismatch**: ran `npx vitest run` on a project that uses Jest. Always check `package.json` scripts first.

## What We'd Do Differently

1. Never apply subagent-reported line numbers without reading the file first. Hallucination rate on line numbers in large diffs was around 50%. Every finding needed manual verification before editing.
2. Always use absolute paths in bash commands. CWD drifted to `apps/frontend` after a `cd` and several subsequent commands hit the wrong directory before catching it.
3. Check `package.json` "test" script before running any test command in a frontend project.
4. When a migration is flagged as dangerous, verify against the actual Postgres version behavior before treating it as a blocker.

## What Worked Well

- Parallel 6-subagent analysis in Phase 1 was efficient for the initial security sweep.
- Verify-before-edit discipline caught all hallucinated line numbers before any bad edits landed.
- Bisectable commit structure (5 logical commits) makes future blame and revert straightforward.
- ReDoS protection pattern is self-contained and reusable for any future endpoint that accepts user-supplied regex.
