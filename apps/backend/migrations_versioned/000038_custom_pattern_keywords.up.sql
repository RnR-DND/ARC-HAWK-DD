ALTER TABLE custom_patterns
  ADD COLUMN IF NOT EXISTS context_keywords TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS negative_keywords TEXT[] NOT NULL DEFAULT '{}';

COMMENT ON COLUMN custom_patterns.context_keywords IS
  'Words found near a match that INCREASE confidence. Each word adds +5% per occurrence (max +25%).';
COMMENT ON COLUMN custom_patterns.negative_keywords IS
  'Words found near a match that DECREASE confidence. Each word subtracts 15% per occurrence.';
