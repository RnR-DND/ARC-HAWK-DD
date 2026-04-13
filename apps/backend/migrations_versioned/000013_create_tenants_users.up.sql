-- Migration: 000013_create_tenants_users
-- Description: Create tenants and users tables, seed a default system tenant for dev mode

-- Create Tenants Table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT true,
    settings TEXT DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create Users Table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL DEFAULT '',
    last_name VARCHAR(100) NOT NULL DEFAULT '',
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Seed default system tenant (DevSystemTenantID = 00000000-0000-0000-0000-000000000001)
-- Must NOT use uuid.Nil (000...000) — EnsureTenantID rejects nil UUIDs as a security guard.
-- All scan-ingested assets/findings use this ID; discovery worker also resolves to this ID.
INSERT INTO tenants (id, name, slug, description, is_active, settings, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'System Default',
    'system-default',
    'Default system tenant for development and anonymous access',
    true,
    '{"theme":"light","notifications":true,"scanFrequency":"daily","dataRetention":"90days"}',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;
