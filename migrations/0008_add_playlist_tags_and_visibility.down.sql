-- Remove indexes
DROP INDEX IF EXISTS idx_playlists_is_public;
DROP INDEX IF EXISTS idx_playlists_tags;

-- Remove columns from playlists table
ALTER TABLE playlists
DROP COLUMN IF EXISTS is_public,
DROP COLUMN IF EXISTS tags;
