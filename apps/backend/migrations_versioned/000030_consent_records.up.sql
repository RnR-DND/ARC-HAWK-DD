-- Migration 000030: Extend consent_records table for DPDPA consent lifecycle tracking.
-- Migration 000008 created consent_records without tenant_id, purpose, etc.
-- This migration adds the missing columns if they don't exist, and creates new indexes.

-- Create the table if it truly doesn't exist yet (fresh installs)
CREATE TABLE IF NOT EXISTS consent_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_subject_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add columns that 000008 may not have created
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS asset_id UUID REFERENCES assets(id) ON DELETE SET NULL;
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS purpose TEXT;
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS consent_given_at TIMESTAMPTZ;
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS consent_expires_at TIMESTAMPTZ;
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS withdrawal_timestamp TIMESTAMPTZ;
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS consent_mechanism TEXT;
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_consent_tenant ON consent_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consent_asset ON consent_records(asset_id);
CREATE INDEX IF NOT EXISTS idx_consent_subject ON consent_records(data_subject_id);
