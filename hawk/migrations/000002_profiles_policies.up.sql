-- 000002_profiles_policies.up.sql
-- Profile and policy tables for RBAC enforcement.

BEGIN;

CREATE TABLE profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  keycloak_user_id TEXT UNIQUE NOT NULL,
  display_name TEXT NOT NULL,
  email TEXT,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE policies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  profile_id UUID NOT NULL REFERENCES profiles(id),
  name TEXT NOT NULL,
  description TEXT,
  allowed_sources UUID[] DEFAULT '{}',  -- empty = all
  allowed_risk_tiers TEXT[] DEFAULT '{}',  -- empty = all
  suppressed_connectors TEXT[] DEFAULT '{}',
  attribute_level TEXT NOT NULL DEFAULT 'login',  -- org, person, login
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_policies_profile ON policies(profile_id);

CREATE TABLE profile_policy_assignments (
  profile_id UUID NOT NULL REFERENCES profiles(id),
  policy_id UUID NOT NULL REFERENCES policies(id),
  PRIMARY KEY (profile_id, policy_id)
);

INSERT INTO schema_migrations (version) VALUES ('000002') ON CONFLICT DO NOTHING;

COMMIT;
