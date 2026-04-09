-- 000027: Mark sample_text values as encrypted (DPDPA P0-1 — AES-256-GCM at rest)
-- New rows will have sample_text_encrypted = TRUE after this migration.
-- Existing rows stay FALSE; a one-time backfill job (see scripts/backfill_encrypt_samples.go)
-- should be run against the live database to encrypt historical rows.

ALTER TABLE findings
    ADD COLUMN IF NOT EXISTS sample_text_encrypted BOOLEAN NOT NULL DEFAULT FALSE;

-- Mark all future inserts as encrypted by default via application logic.
-- The column is read by the API layer to know whether to decrypt before returning.

COMMENT ON COLUMN findings.sample_text_encrypted IS
    'TRUE = sample_text is AES-256-GCM encrypted (base64). FALSE = plaintext (pre-migration or dev).';
