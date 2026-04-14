ALTER TABLE findings ADD COLUMN IF NOT EXISTS detector_type TEXT DEFAULT 'regex';
