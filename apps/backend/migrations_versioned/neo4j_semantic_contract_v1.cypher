-- ARC-Hawk Neo4j Migration: 4-Level to 3-Level Hierarchy
-- CRITICAL: This migration clears ALL existing graph data
-- Run ONLY when ready to rebuild lineage from PostgreSQL
-- ===================================================================

-- Step 1: Delete all existing nodes and relationships
MATCH (n) DETACH DELETE n;

-- Step 2: Create constraints for new 3-level schema
-- Ensures uniqueness and enables faster lookups

-- System nodes (identified by ID)
CREATE CONSTRAINT system_id_unique IF NOT EXISTS
FOR (s:System) REQUIRE s.id IS UNIQUE;

-- Asset nodes (identified by ID)
CREATE CONSTRAINT asset_id_unique IF NOT EXISTS
FOR (a:Asset) REQUIRE a.id IS UNIQUE;

-- PII_Category nodes (identified by type: IN_AADHAAR, CREDIT_CARD, etc.)
CREATE CONSTRAINT pii_category_type_unique IF NOT EXISTS
FOR (p:PII_Category) REQUIRE p.type IS UNIQUE;

-- Step 3: Create indexes for performance

-- Index on System.host for filtering
CREATE INDEX system_host_idx IF NOT EXISTS
FOR (s:System) ON (s.host);

-- Index on Asset.path for lookups
CREATE INDEX asset_path_idx IF NOT EXISTS
FOR (a:Asset) ON (a.path);

-- Index on PII_Category.risk_level for filtering
CREATE INDEX pii_risk_level_idx IF NOT EXISTS
FOR (p:PII_Category) ON (p.risk_level);

-- Step 4: Verify schema
-- Run these queries to confirm migration success:

-- MATCH (s:System) RETURN count(s) as systems;
-- MATCH (a:Asset) RETURN count(a) as assets;
-- MATCH (p:PII_Category) RETURN count(p) as pii_categories;
-- MATCH ()-[r:SYSTEM_OWNS_ASSET]->() RETURN count(r) as owns_edges;
-- MATCH ()-[r:ASSET_CONTAINS_PII]->() RETURN count(r) as contains_edges;

-- Step 5: Next steps
-- After running this migration:
-- 1. Restart backend service
-- 2. Run: POST /api/v1/lineage/sync to rebuild graph from PostgreSQL
-- 3. Verify frontend lineage view displays correct 3-level hierarchy
