-- Phase 11: DPDPA Sec 5 — purpose limitation requires declared_purpose on each asset.
-- Phase 5:  Orphan detection — assets with no scan findings for > 90 days get tagged.
-- Phase 7:  DPO assignment tracking for high-risk assets.

ALTER TABLE assets
    ADD COLUMN IF NOT EXISTS declared_purpose TEXT,
    ADD COLUMN IF NOT EXISTS dpo_assigned      VARCHAR(256),
    ADD COLUMN IF NOT EXISTS is_orphan         BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS orphan_since      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS orphan_risk_bonus INTEGER NOT NULL DEFAULT 0;

-- Index for DPDPA obligation queries
CREATE INDEX IF NOT EXISTS idx_assets_no_purpose ON assets(declared_purpose) WHERE declared_purpose IS NULL;
CREATE INDEX IF NOT EXISTS idx_assets_dpo        ON assets(dpo_assigned)     WHERE dpo_assigned IS NULL;
CREATE INDEX IF NOT EXISTS idx_assets_orphan     ON assets(is_orphan)        WHERE is_orphan = TRUE;
