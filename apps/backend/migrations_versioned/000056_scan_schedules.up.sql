CREATE TABLE IF NOT EXISTS scan_schedules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    profile_name TEXT NOT NULL,
    frequency   TEXT NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    hour        SMALLINT NOT NULL DEFAULT 2 CHECK (hour BETWEEN 0 AND 23),
    day_of_week SMALLINT CHECK (day_of_week BETWEEN 0 AND 6),  -- weekly: 0=Sun
    day_of_month SMALLINT CHECK (day_of_month BETWEEN 1 AND 31), -- monthly
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, profile_name)
);

CREATE INDEX IF NOT EXISTS idx_scan_schedules_tenant ON scan_schedules (tenant_id);
CREATE INDEX IF NOT EXISTS idx_scan_schedules_next_run ON scan_schedules (next_run_at) WHERE enabled = TRUE;
