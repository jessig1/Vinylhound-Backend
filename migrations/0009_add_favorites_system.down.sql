-- Remove favorites playlists (keep the playlist_songs for data integrity)
DELETE FROM playlists WHERE is_favorite = TRUE;

-- Drop indexes
DROP INDEX IF EXISTS idx_playlists_is_favorite;
DROP INDEX IF EXISTS idx_favorites_album_id;
DROP INDEX IF EXISTS idx_favorites_song_id;
DROP INDEX IF EXISTS idx_favorites_user_id;
DROP INDEX IF EXISTS idx_playlists_user_favorite;

-- Remove is_favorite column from playlists
ALTER TABLE playlists DROP COLUMN IF EXISTS is_favorite;

-- Drop favorites table
DROP TABLE IF EXISTS favorites;
