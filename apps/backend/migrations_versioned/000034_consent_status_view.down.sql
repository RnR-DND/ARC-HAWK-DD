-- Rollback migration 000034
DROP VIEW IF EXISTS consent_status_view;

ALTER TABLE consent_records
    DROP COLUMN IF EXISTS asset_id,
    DROP COLUMN IF EXISTS consent_obtained_at,
    DROP COLUMN IF EXISTS consent_basis,
    DROP COLUMN IF EXISTS purpose,
    DROP COLUMN IF EXISTS obtained_by,
    DROP COLUMN IF EXISTS withdrawal_requested_by,
    DROP COLUMN IF EXISTS withdrawal_reason,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS tenant_id;
