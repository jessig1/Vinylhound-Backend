-- Add tags and is_public columns to playlists table
ALTER TABLE playlists
ADD COLUMN tags TEXT[] DEFAULT '{}',
ADD COLUMN is_public BOOLEAN DEFAULT false;

-- Add index for searching by tags
CREATE INDEX idx_playlists_tags ON playlists USING GIN(tags);

-- Add index for filtering by is_public
CREATE INDEX idx_playlists_is_public ON playlists(is_public);
