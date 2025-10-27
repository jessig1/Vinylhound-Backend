-- Performance indexes for album search and filtering

-- B-tree indexes for exact match and range queries
CREATE INDEX IF NOT EXISTS idx_albums_release_year ON albums(release_year);
CREATE INDEX IF NOT EXISTS idx_albums_rating ON albums(rating);

-- Text search indexes using trigram for ILIKE queries
-- This requires pg_trgm extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Trigram indexes for case-insensitive text search
CREATE INDEX IF NOT EXISTS idx_albums_artist_trgm ON albums USING gin (artist gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_albums_title_trgm ON albums USING gin (title gin_trgm_ops);

-- GIN index for JSONB genre containment queries
CREATE INDEX IF NOT EXISTS idx_albums_genres ON albums USING gin (genres);

-- Composite index for user_id + release_year ordering (used in AlbumsByToken)
CREATE INDEX IF NOT EXISTS idx_albums_user_release ON albums(user_id, release_year DESC);

-- Indexes for user_album_preferences table
-- Primary key already covers (user_id, album_id)
-- But we need indexes for aggregation queries

-- Index for album_id lookups in fetchAlbumRatingStats
CREATE INDEX IF NOT EXISTS idx_user_album_prefs_album_id ON user_album_preferences(album_id)
    WHERE rating IS NOT NULL;

-- Index for favorited albums
CREATE INDEX IF NOT EXISTS idx_user_album_prefs_favorited ON user_album_preferences(user_id, favorited)
    WHERE favorited = TRUE;

-- Index for updated_at ordering (used in AlbumPreferencesByToken)
CREATE INDEX IF NOT EXISTS idx_user_album_prefs_updated ON user_album_preferences(user_id, updated_at DESC);

-- Composite index for rating aggregation (covers album_id + rating columns)
CREATE INDEX IF NOT EXISTS idx_user_album_prefs_rating_stats ON user_album_preferences(album_id, rating, user_id)
    WHERE rating IS NOT NULL;
