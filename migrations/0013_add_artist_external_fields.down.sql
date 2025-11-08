-- Remove external provider fields from artists table
DROP INDEX IF EXISTS idx_artists_external_id;
ALTER TABLE artists DROP COLUMN IF EXISTS external_url;
ALTER TABLE artists DROP COLUMN IF EXISTS popularity;
ALTER TABLE artists DROP COLUMN IF EXISTS genres;
ALTER TABLE artists DROP COLUMN IF EXISTS provider;
ALTER TABLE artists DROP COLUMN IF EXISTS external_id;
