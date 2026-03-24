-- 1. Delete findings from test files (Aggressive)
DELETE FROM findings 
WHERE asset_id IN (
    SELECT id FROM assets 
    WHERE 
        path ILIKE '%test%' OR 
        path ILIKE '%mock%' OR 
        path ILIKE '%spec%' OR
        path ILIKE '%fixture%' OR
        path ILIKE '/tmp/%' OR
        path ILIKE '%example%'
);

-- 2. Delete findings with suspicious patterns
DELETE FROM findings 
WHERE 
    sample_text ILIKE '%John Doe%' OR
    sample_text ILIKE '%Example%' OR
    sample_text ILIKE '%test%' OR
    sample_text = '1234567890';


-- 3. Delete orphaned classifications
DELETE FROM classifications 
WHERE finding_id NOT IN (SELECT id FROM findings);

-- 4. Delete orphaned review states
DELETE FROM review_states 
WHERE finding_id NOT IN (SELECT id FROM findings);

-- 5. Delete orphaned assets (optional, but good for cleanup)
DELETE FROM assets 
WHERE id NOT IN (SELECT asset_id FROM findings);
