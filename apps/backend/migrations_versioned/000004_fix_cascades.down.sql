-- Rollback cascade fixes (restore to initial state)

ALTER TABLE review_states DROP CONSTRAINT IF EXISTS review_states_finding_id_fkey;

-- Re-add without explicit CASCADE (will use default which was likely RESTRICT)
ALTER TABLE review_states 
ADD CONSTRAINT review_states_finding_id_fkey 
FOREIGN KEY (finding_id) REFERENCES findings(id);
