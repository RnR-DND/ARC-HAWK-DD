-- Migration 000039: Backfill confidence scores for stale findings and classifications.
--
-- Previous code hardcoded confidence_score=0.5 for all findings and 0.66/1.0 for
-- classifications. This migration applies pattern-based best-effort scores to existing
-- rows so the UI reflects meaningful differentiation before a re-scan is run.
--
-- Logic mirrors classification_service.go classifyWithRules:
--   Aadhaar / PAN / Passport  → 0.99
--   Credit / Debit Card        → 0.95
--   Email                      → 0.95
--   Phone / Mobile             → 0.90
--   GST / Tax                  → 0.88
--   Generic / unrecognised     → 0.75
--
-- Only rows with the known flat defaults (0.5 or 0.66) are updated so any legitimately
-- low-confidence findings are left untouched.

UPDATE findings
SET confidence_score = CASE
    WHEN pattern_name ILIKE '%aadhaar%' OR pattern_name ILIKE '%aadhar%'
      OR pattern_name ILIKE '%pan%'     OR pattern_name ILIKE '%passport%'
      OR pattern_name ILIKE '%ssn%'     OR pattern_name ILIKE '%social_security%' THEN 0.99
    WHEN pattern_name ILIKE '%credit_card%' OR pattern_name ILIKE '%debit_card%'
      OR pattern_name ILIKE '%card_number%' OR pattern_name ILIKE '%cvv%'         THEN 0.95
    WHEN pattern_name ILIKE '%email%'   OR pattern_name ILIKE '%e-mail%'           THEN 0.95
    WHEN pattern_name ILIKE '%phone%'   OR pattern_name ILIKE '%mobile%'
      OR pattern_name ILIKE '%cellphone%'                                           THEN 0.90
    WHEN pattern_name ILIKE '%gst%'     OR pattern_name ILIKE '%gstin%'
      OR pattern_name ILIKE '%tax_id%'  OR pattern_name ILIKE '%vat%'              THEN 0.88
    WHEN pattern_name ILIKE '%aws_key%' OR pattern_name ILIKE '%api_key%'
      OR pattern_name ILIKE '%secret%'  OR pattern_name ILIKE '%password%'
      OR pattern_name ILIKE '%token%'                                               THEN 0.93
    ELSE 0.75
END
WHERE confidence_score IS NULL OR confidence_score = 0.5;

-- Backfill classifications that still carry the legacy flat value (0.66 or 1.0).
UPDATE classifications c
SET confidence_score = f.confidence_score
FROM findings f
WHERE c.finding_id = f.id
  AND c.confidence_score IN (0.5, 0.66, 1.0);
