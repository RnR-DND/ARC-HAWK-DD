# Runbook: Schema Drift Detection and Classification

## Trigger

The scanner detects that a database table's schema has changed since the last successful scan — columns have been added, removed, or renamed. This is called **schema drift**. Because column names are a primary signal for PII classification (e.g., a column named `email_address` is a strong indicator of PII), schema changes may mean existing findings are stale or new PII has been introduced without a re-scan.

Activates when the scanner logs `event: schema_drift_detected` or when an asset shows `scan_warning: schema_changed`.

---

## Symptoms

**Logs (scanner worker):**
```
WARN   hawk_scanner  source=postgresql  table=users  event=schema_drift_detected
         added_columns=["date_of_birth","national_id"]
         removed_columns=["legacy_id"]
         changed_columns=["phone" -> "phone_number"]
```

**Dashboard:** The asset (table) shows a `SCHEMA CHANGED` badge. The asset detail page lists the drift: added columns, removed columns, renamed columns.

**Compliance:** If added columns match PII patterns (e.g., `national_id`, `date_of_birth`), new DPDPA obligations may be triggered immediately.

**Metrics:** `hawk_scanner_schema_drift_total{source="postgresql"}` counter increments.

---

## Automated Response

The scanner compares the current table schema against the stored schema snapshot in the `asset_schema_snapshots` table. On drift detection it:

1. Logs the diff (added/removed/changed columns).
2. Updates the stored schema snapshot to the current state.
3. Sets `scan_warning = 'schema_changed'` on the asset record.
4. Flags the asset for priority re-classification in the next scan run.
5. If any new column name matches a high-confidence PII indicator (e.g., `ssn`, `aadhaar`, `dob`, `credit_card`), the asset's `risk_score` is pre-emptively bumped by 10 points and an alert is generated.

The scanner does **not** automatically delete findings for removed columns — those findings are marked `stale_column_dropped` and retained for audit purposes.

---

## Manual Steps

1. **Review the drift** in the asset detail page or via the API:
   ```bash
   curl http://localhost:8080/api/v1/assets/{asset_id}/schema-diff
   ```

2. **Assess new columns for PII content:**
   - Are the new column names indicative of personal data (e.g., `national_id`, `home_address`, `dob`)?
   - If yes, trigger an immediate re-scan (do not wait for the scheduled scan).

3. **Trigger a targeted re-scan:**
   ```bash
   curl -X POST http://localhost:8080/api/v1/scan/trigger \
     -H "Content-Type: application/json" \
     -d '{"connection_id": "{connection_id}", "asset_id": "{asset_id}"}'
   ```

4. **Review new findings** after the re-scan completes. Classify them and create compliance records as required.

5. **Handle stale findings** from removed columns:
   - Review findings tagged `stale_column_dropped`.
   - If the column was removed as part of a PII remediation effort, mark the findings as `remediated`.
   - If the column was removed erroneously and data may still exist, investigate the data source and the change history.
   ```bash
   # Mark stale findings as remediated
   curl -X POST http://localhost:8080/api/v1/remediation/bulk \
     -H "Content-Type: application/json" \
     -d '{"finding_ids": ["..."], "action": "remediated", "reason": "column_dropped"}'
   ```

6. **Update DPDPA consent records** if the new columns introduce data that requires consent under DPDPA Sec 4/6. Use the consent API to create or update consent records for the affected assets.

7. **Clear the schema drift warning** once all new columns have been classified and all stale findings resolved:
   ```bash
   curl -X PUT http://localhost:8080/api/v1/assets/{asset_id} \
     -H "Content-Type: application/json" \
     -d '{"scan_warning": null}'
   ```

---

## Resolution Criteria

- A re-scan has completed for the affected asset after the schema change.
- All new columns have been classified (PII or not PII) in the findings.
- Stale findings from removed columns have been resolved (remediated or excluded).
- The asset's `scan_warning` is cleared.
- If new PII columns were found, consent and purpose records are up to date.

---

## Prevention

- **Schema change notifications:** Integrate with your database migration tool (Flyway, Liquibase, custom scripts) to send a webhook to ARC-Hawk when a migration is applied. This triggers an immediate re-scan rather than waiting for drift to be detected.
- **Pre-migration PII review:** Require a data privacy review for any database migration that adds columns containing personal data. The `POST /api/v1/patterns` dry-run mode can be used to check new column names against known PII patterns before the migration is applied.
- **Column naming conventions:** Establish a naming convention that makes PII columns identifiable (e.g., prefix `pii_`). The scanner can be configured to treat all such columns as high-priority.
- **Audit trail for migrations:** Ensure all schema changes are captured in migration files under version control so the origin of any new column can be traced.
