-- API keys for service-to-service and CI/CD authentication (P5-4)
-- Keys are stored as SHA-256 hashes; the plain-text key is shown once at creation.
CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    created_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,                           -- human label e.g. "CI/CD pipeline"
    key_hash     TEXT NOT NULL UNIQUE,                    -- SHA-256(plain_key), hex-encoded
    key_prefix   CHAR(8) NOT NULL,                        -- first 8 chars of plain key, for display
    scopes       TEXT[] NOT NULL DEFAULT '{}',            -- e.g. '{scan:read,scan:write}'
    expires_at   TIMESTAMPTZ,                             -- NULL = non-expiring
    last_used_at TIMESTAMPTZ,
    revoked      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);

COMMENT ON TABLE api_keys IS 'API keys for service-to-service and CI/CD auth. Plain key shown once; only SHA-256 hash stored.';
