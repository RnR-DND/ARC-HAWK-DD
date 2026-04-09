-- Migration 000034: Add missing columns to consent_records and create consent_status_view.
--
-- Background: migration 000008 created consent_records with a minimal schema.
-- Migration 000030 attempted CREATE TABLE IF NOT EXISTS (no-op on existing table),
-- so columns needed by consent_service.go were never added.
-- This migration safely adds the missing columns and creates the view that
-- consent_service.go:156,198,276 queries.

-- ============================================================================
-- Add missing columns (IF NOT EXISTS so idempotent on re-run)
-- ============================================================================

ALTER TABLE consent_records
    ADD COLUMN IF NOT EXISTS asset_id         UUID REFERENCES assets(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS consent_obtained_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS consent_basis     TEXT,
    ADD COLUMN IF NOT EXISTS purpose           TEXT,
    ADD COLUMN IF NOT EXISTS obtained_by       TEXT,
    ADD COLUMN IF NOT EXISTS withdrawal_requested_by TEXT,
    ADD COLUMN IF NOT EXISTS withdrawal_reason TEXT,
    ADD COLUMN IF NOT EXISTS updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS tenant_id         UUID;

-- Back-fill consent_obtained_at from consent_given_at where it is NULL
UPDATE consent_records
   SET consent_obtained_at = consent_given_at
 WHERE consent_obtained_at IS NULL AND consent_given_at IS NOT NULL;

-- Back-fill purpose from consent_purpose where it is NULL
UPDATE consent_records
   SET purpose = consent_purpose
 WHERE purpose IS NULL AND consent_purpose IS NOT NULL;

-- Indexes for new columns
CREATE INDEX IF NOT EXISTS idx_consent_records_asset_id     ON consent_records(asset_id);
CREATE INDEX IF NOT EXISTS idx_consent_records_tenant_id    ON consent_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consent_records_obtained_at  ON consent_records(consent_obtained_at DESC);

-- ============================================================================
-- consent_status_view
-- Computes a derived `status` field so callers don't need CASE logic in SQL.
-- consent_service.go queries: id, asset_id, pii_type, consent_obtained_at,
--   consent_expires_at, consent_withdrawn_at, consent_basis, purpose,
--   obtained_by, withdrawal_requested_by, withdrawal_reason, metadata,
--   created_at, updated_at, status
-- ============================================================================

CREATE OR REPLACE VIEW consent_status_view AS
SELECT
    id,
    asset_id,
    pii_type,
    COALESCE(consent_obtained_at, consent_given_at)          AS consent_obtained_at,
    consent_expires_at,
    COALESCE(consent_withdrawn_at, NULL)                      AS consent_withdrawn_at,
    COALESCE(consent_basis, 'explicit')                       AS consent_basis,
    COALESCE(purpose, consent_purpose, '')                    AS purpose,
    COALESCE(obtained_by, consent_source, 'unknown')          AS obtained_by,
    withdrawal_requested_by,
    withdrawal_reason,
    metadata,
    created_at,
    COALESCE(updated_at, created_at)                          AS updated_at,
    CASE
        WHEN COALESCE(consent_withdrawn_at, NULL) IS NOT NULL
            THEN 'WITHDRAWN'
        WHEN consent_expires_at IS NOT NULL
             AND consent_expires_at < NOW()
            THEN 'EXPIRED'
        WHEN consent_expires_at IS NOT NULL
             AND consent_expires_at < NOW() + INTERVAL '30 days'
            THEN 'EXPIRING_SOON'
        ELSE 'VALID'
    END AS status
FROM consent_records;
