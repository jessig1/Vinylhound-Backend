-- Remove performance indexes

DROP INDEX IF EXISTS idx_user_album_prefs_rating_stats;
DROP INDEX IF EXISTS idx_user_album_prefs_updated;
DROP INDEX IF EXISTS idx_user_album_prefs_favorited;
DROP INDEX IF EXISTS idx_user_album_prefs_album_id;
DROP INDEX IF EXISTS idx_albums_user_release;
DROP INDEX IF EXISTS idx_albums_genres;
DROP INDEX IF EXISTS idx_albums_title_trgm;
DROP INDEX IF EXISTS idx_albums_artist_trgm;
DROP INDEX IF EXISTS idx_albums_rating;
DROP INDEX IF EXISTS idx_albums_release_year;

-- Note: We don't drop pg_trgm extension as other parts of the application might use it
