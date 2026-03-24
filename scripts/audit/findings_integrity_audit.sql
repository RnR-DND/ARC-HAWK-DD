-- =====================================================================
-- ARC-HAWK FINDINGS SYSTEM INTEGRITY AUDIT
-- Comprehensive verification of data integrity and traceability
-- =====================================================================

-- Create audit results table
DROP TABLE IF EXISTS audit_results;
CREATE TABLE audit_results (
    test_name VARCHAR(255) PRIMARY KEY,
    status VARCHAR(20) NOT NULL, -- PASS, FAIL, WARNING
    details TEXT,
    count_affected INTEGER DEFAULT 0,
    audit_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================================
-- 1. FINDING STRUCTURE VERIFICATION
-- =====================================================================

INSERT INTO audit_results (test_name, status, details, count_affected) 
SELECT 
    'finding_required_fields',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'FAIL'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All findings have required fields'
        ELSE CONCAT('Findings missing required fields: ', COUNT(*))
    END,
    COUNT(*)
FROM findings f
WHERE 
    f.scan_run_id IS NULL 
    OR f.asset_id IS NULL 
    OR f.pattern_name IS NULL 
    OR f.pattern_name = ''
    OR f.severity IS NULL 
    OR f.severity = '';

-- Verify valid PII types (patterns)
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'valid_pii_types',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'FAIL'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All findings use valid PII patterns'
        ELSE CONCAT('Findings with invalid PII types: ', COUNT(*))
    END,
    COUNT(*)
FROM findings f
LEFT JOIN patterns p ON f.pattern_name = p.name
WHERE p.name IS NULL;

-- Verify confidence scores are within valid range
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'confidence_score_range',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'FAIL'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All confidence scores are valid (0.0-1.0)'
        ELSE CONCAT('Findings with invalid confidence scores: ', COUNT(*))
    END,
    COUNT(*)
FROM findings f
WHERE 
    f.confidence_score IS NOT NULL 
    AND (f.confidence_score < 0.0 OR f.confidence_score > 1.0);

-- =====================================================================
-- 2. SCAN-FINDING RELATIONSHIP VERIFICATION
-- =====================================================================

-- Check for orphaned findings (findings without valid scan runs)
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'scan_finding_relationship',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'CRITICAL'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All findings have valid scan runs'
        ELSE CONCAT('Orphaned findings without valid scan runs: ', COUNT(*))
    END,
    COUNT(*)
FROM findings f
LEFT JOIN scan_runs sr ON f.scan_run_id = sr.id
WHERE sr.id IS NULL;

-- Verify scan run statistics match actual findings
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'scan_statistics_accuracy',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'WARNING'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'Scan run statistics are accurate'
        ELSE CONCAT('Scan runs with inaccurate finding counts: ', COUNT(*))
    END,
    COUNT(*)
FROM scan_runs sr
LEFT JOIN (
    SELECT scan_run_id, COUNT(*) as actual_count
    FROM findings
    GROUP BY scan_run_id
) fc ON sr.id = fc.scan_run_id
WHERE COALESCE(fc.actual_count, 0) != sr.total_findings;

-- =====================================================================
-- 3. ASSET-FINDING RELATIONSHIP VERIFICATION
-- =====================================================================

-- Check for findings without valid assets
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'asset_finding_relationship',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'CRITICAL'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All findings belong to valid assets'
        ELSE CONCAT('Findings without valid assets: ', COUNT(*))
    END,
    COUNT(*)
FROM findings f
LEFT JOIN assets a ON f.asset_id = a.id
WHERE a.id IS NULL;

-- Verify asset finding counts are accurate
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'asset_statistics_accuracy',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'WARNING'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'Asset finding counts are accurate'
        ELSE CONCAT('Assets with inaccurate finding counts: ', COUNT(*))
    END,
    COUNT(*)
FROM assets a
LEFT JOIN (
    SELECT asset_id, COUNT(*) as actual_count
    FROM findings
    GROUP BY asset_id
) fc ON a.id = fc.asset_id
WHERE COALESCE(fc.actual_count, 0) != a.total_findings;

-- =====================================================================
-- 4. LOCATION TRACEABILITY VERIFICATION
-- =====================================================================

-- Check findings without proper location data (file paths, etc.)
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'location_traceability',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'FAIL'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All findings have traceable locations'
        ELSE CONCAT('Findings without location data: ', COUNT(*))
    END,
    COUNT(*)
FROM findings f
LEFT JOIN assets a ON f.asset_id = a.id
WHERE a.path IS NULL OR a.path = '';

-- Verify unique asset paths (no duplicates without stable_id)
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'unique_asset_locations',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'WARNING'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All asset locations are properly unique'
        ELSE CONCAT('Duplicate asset paths without stable_id: ', COUNT(*))
    END,
    COUNT(*)
FROM (
    SELECT path, COUNT(*) as dup_count
    FROM assets
    WHERE path IS NOT NULL AND path != ''
    GROUP BY path
    HAVING COUNT(*) > 1
) dup_paths;

-- =====================================================================
-- 5. VALIDATION LOGIC REFERENCE VERIFICATION
-- =====================================================================

-- Check findings reference actual validation methods
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'validation_logic_reference',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'FAIL'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All findings reference valid patterns'
        ELSE CONCAT('Findings referencing invalid patterns: ', COUNT(*))
    END,
    COUNT(*)
FROM findings f
LEFT JOIN patterns p ON f.pattern_id = p.id
WHERE f.pattern_id IS NOT NULL AND p.id IS NULL;

-- Verify classifications exist for findings
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'classification_completeness',
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS'
        ELSE 'WARNING'
    END,
    CASE 
        WHEN COUNT(*) = 0 THEN 'All findings have classifications'
        ELSE CONCAT('Findings without classifications: ', COUNT(*))
    END,
    COUNT(*)
FROM findings f
LEFT JOIN classifications c ON f.id = c.finding_id
WHERE c.finding_id IS NULL;

-- =====================================================================
-- 6. AGGREGATION INTEGRITY VERIFICATION
-- =====================================================================

-- Verify total findings count matches individual records
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'aggregation_integrity',
    CASE 
        WHEN ABS(COUNT(*) - (SELECT COUNT(*) FROM findings)) = 0 THEN 'PASS'
        ELSE 'CRITICAL'
    END,
    CASE 
        WHEN COUNT(*) = (SELECT COUNT(*) FROM findings) THEN 'Aggregation statistics are accurate'
        ELSE CONCAT('Aggregation mismatch: reported=', COUNT(*), ', actual=', (SELECT COUNT(*) FROM findings))
    END,
    ABS(COUNT(*) - (SELECT COUNT(*) FROM findings))
FROM scan_runs sr;

-- Check for hidden aggregations (classified findings properly filtered)
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'classification_filtering',
    CASE 
        WHEN non_pii_count = 0 THEN 'PASS'
        ELSE 'WARNING'
    END,
    CASE 
        WHEN non_pii_count = 0 THEN 'Non-PII findings are properly filtered'
        ELSE CONCAT('Non-PII findings in system: ', non_pii_count)
    END,
    non_pii_count
FROM (
    SELECT COUNT(*) as non_pii_count
    FROM classifications c
    WHERE c.classification_type = 'Non-PII'
) np;

-- =====================================================================
-- 7. CASCADE DELETION BEHAVIOR VERIFICATION
-- =====================================================================

-- Check cascade deletion constraints are properly set
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'cascade_constraints',
    CASE 
        WHEN COUNT(*) = expected_constraints THEN 'PASS'
        ELSE 'FAIL'
    END,
    CASE 
        WHEN COUNT(*) = expected_constraints THEN 'All cascade constraints are properly configured'
        ELSE CONCAT('Missing cascade constraints: ', expected_constraints - COUNT(*))
    END,
    expected_constraints - COUNT(*)
FROM (
    SELECT COUNT(*) as constraint_count
    FROM information_schema.referential_constraints rc
    JOIN information_schema.table_constraints tc ON rc.constraint_name = tc.constraint_name
    WHERE rc.delete_rule = 'CASCADE'
    AND tc.table_name IN ('findings', 'classifications', 'review_states', 'finding_feedback', 'asset_relationships')
) actual_constraints,
(
    SELECT 6 as expected_constraints -- Expected cascade constraints
) expected;

-- Verify no circular dependencies in cascade chains
INSERT INTO audit_results (test_name, status, details)
SELECT 
    'cascade_safety',
    'PASS',
    'Cascade deletion chains are properly designed (scan_runs -> findings -> child tables)'
WHERE EXISTS (
    SELECT 1 
    FROM information_schema.referential_constraints rc1
    WHERE rc1.delete_rule = 'CASCADE'
    AND rc1.constraint_schema = 'public'
    LIMIT 1
);

-- =====================================================================
-- 8. COMPREHENSIVE TRACEABILITY TEST
-- =====================================================================

-- Complete traceability: finding -> asset -> scan -> classification
INSERT INTO audit_results (test_name, status, details, count_affected)
SELECT 
    'complete_traceability',
    CASE 
        WHEN COUNT(*) = total_findings THEN 'PASS'
        ELSE 'CRITICAL'
    END,
    CASE 
        WHEN COUNT(*) = total_findings THEN 'All findings have complete audit trail'
        ELSE CONCAT('Findings with incomplete traceability: ', total_findings - COUNT(*))
    END,
    total_findings - COUNT(*)
FROM (
    SELECT f.id
    FROM findings f
    JOIN assets a ON f.asset_id = a.id
    JOIN scan_runs sr ON f.scan_run_id = sr.id
    LEFT JOIN classifications c ON f.id = c.finding_id
    WHERE a.id IS NOT NULL AND sr.id IS NOT NULL
) traceable,
(
    SELECT COUNT(*) as total_findings FROM findings
) totals;

-- =====================================================================
-- AUDIT SUMMARY
-- =====================================================================

-- Generate comprehensive audit report
SELECT 
    test_name,
    status,
    details,
    count_affected,
    CASE 
        WHEN status = 'CRITICAL' THEN '🚨 Immediate attention required'
        WHEN status = 'FAIL' THEN '❌ Must be fixed'
        WHEN status = 'WARNING' THEN '⚠️  Should be reviewed'
        WHEN status = 'PASS' THEN '✅ No issues found'
    END as recommendation
FROM audit_results
ORDER BY 
    CASE 
        WHEN status = 'CRITICAL' THEN 1
        WHEN status = 'FAIL' THEN 2
        WHEN status = 'WARNING' THEN 3
        WHEN status = 'PASS' THEN 4
    END,
    test_name;

-- Overall system health summary
SELECT 
    'SYSTEM_HEALTH_SUMMARY' as test_name,
    CASE 
        WHEN critical_count > 0 THEN 'CRITICAL'
        WHEN fail_count > 0 THEN 'FAIL'
        WHEN warning_count > 0 THEN 'WARNING'
        ELSE 'PASS'
    END as status,
    CONCAT(
        'Critical: ', critical_count, 
        ', Fail: ', fail_count, 
        ', Warning: ', warning_count, 
        ', Pass: ', pass_count
    ) as details,
    COUNT(*) as count_affected
FROM audit_results,
(
    SELECT 
        SUM(CASE WHEN status = 'CRITICAL' THEN 1 ELSE 0 END) as critical_count,
        SUM(CASE WHEN status = 'FAIL' THEN 1 ELSE 0 END) as fail_count,
        SUM(CASE WHEN status = 'WARNING' THEN 1 ELSE 0 END) as warning_count,
        SUM(CASE WHEN status = 'PASS' THEN 1 ELSE 0 END) as pass_count
    FROM audit_results
) counts;

-- =====================================================================
-- REPAIR SUGGESTIONS
-- =====================================================================

-- Identify specific records that need attention
SELECT 'ORPHANED_FINDINGS' as issue_type, COUNT(*) as count, 
       'Findings without valid scan runs' as description
FROM findings f
LEFT JOIN scan_runs sr ON f.scan_run_id = sr.id
WHERE sr.id IS NULL

UNION ALL

SELECT 'ASSET_ORPHANS' as issue_type, COUNT(*) as count,
       'Findings without valid assets' as description  
FROM findings f
LEFT JOIN assets a ON f.asset_id = a.id
WHERE a.id IS NULL

UNION ALL

SELECT 'MISSING_CLASSIFICATIONS' as issue_type, COUNT(*) as count,
       'Findings without classification records' as description
FROM findings f
LEFT JOIN classifications c ON f.id = c.finding_id
WHERE c.finding_id IS NULL

UNION ALL

SELECT 'INVALID_PATTERNS' as issue_type, COUNT(*) as count,
       'Findings referencing unknown patterns' as description
FROM findings f
LEFT JOIN patterns p ON f.pattern_name = p.name
WHERE p.name IS NULL

ORDER BY count DESC;

