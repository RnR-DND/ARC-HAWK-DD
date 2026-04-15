ALTER TABLE custom_patterns
  DROP COLUMN IF EXISTS context_keywords,
  DROP COLUMN IF EXISTS negative_keywords;
