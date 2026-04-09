-- Migration 000030: Standalone consent_records table for DPDPA consent lifecycle tracking.
-- Note: migration 000008 also created a consent_records table; this migration is additive
-- and uses IF NOT EXISTS so it is safe to run even if that earlier table exists.
-- If the older table already exists with a different schema, this migration is a no-op on the table.
CREATE TABLE IF NOT EXISTS consent_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    data_subject_id TEXT NOT NULL,     -- anonymised identifier
    asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,
    purpose TEXT NOT NULL,
    consent_given_at TIMESTAMPTZ NOT NULL,
    consent_expires_at TIMESTAMPTZ,
    withdrawal_timestamp TIMESTAMPTZ,
    consent_mechanism TEXT NOT NULL,   -- 'explicit', 'implicit', 'legitimate_interest'
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_consent_tenant ON consent_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consent_asset ON consent_records(asset_id);
CREATE INDEX IF NOT EXISTS idx_consent_subject ON consent_records(data_subject_id);
