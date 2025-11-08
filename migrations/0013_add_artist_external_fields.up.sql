-- Add external provider fields to artists table
ALTER TABLE artists ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS provider TEXT;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS genres JSONB DEFAULT '[]'::jsonb;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS popularity INTEGER;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS external_url TEXT;

-- Create index for external_id lookups
CREATE INDEX IF NOT EXISTS idx_artists_external_id ON artists(external_id, provider);
