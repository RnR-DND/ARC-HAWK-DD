-- Fix foreign key cascades
-- Ensures proper cascade deletion from scan_runs -> findings -> review_states

-- Drop existing constraint on review_states
ALTER TABLE review_states DROP CONSTRAINT IF EXISTS review_states_finding_id_fkey;

-- Re-add with proper CASCADE
ALTER TABLE review_states 
ADD CONSTRAINT review_states_finding_id_fkey 
FOREIGN KEY (finding_id) REFERENCES findings(id) ON DELETE CASCADE;

-- Verify cascade paths (these should already exist from initial schema):
-- scan_runs -> findings: ON DELETE CASCADE ✓
-- findings -> classifications: ON DELETE CASCADE ✓
-- findings -> review_states: ON DELETE CASCADE ✓ (just fixed)
-- assets -> findings: ON DELETE CASCADE ✓
-- assets -> asset_relationships: ON DELETE CASCADE ✓
