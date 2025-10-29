-- Add favorites table to track user favorites for both songs and albums
CREATE TABLE IF NOT EXISTS favorites (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    song_id BIGINT REFERENCES songs(id) ON DELETE CASCADE,
    album_id BIGINT REFERENCES albums(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    -- Constraint to ensure either song_id or album_id is set, but not both
    CONSTRAINT check_favorite_type CHECK (
        (song_id IS NOT NULL AND album_id IS NULL) OR
        (song_id IS NULL AND album_id IS NOT NULL)
    ),
    -- Unique constraint to prevent duplicate favorites
    CONSTRAINT unique_user_song_favorite UNIQUE (user_id, song_id),
    CONSTRAINT unique_user_album_favorite UNIQUE (user_id, album_id)
);

-- Add is_favorite flag to playlists to mark the special favorites playlist
ALTER TABLE playlists ADD COLUMN IF NOT EXISTS is_favorite BOOLEAN DEFAULT FALSE;

-- Add unique constraint to ensure only one favorites playlist per user
CREATE UNIQUE INDEX idx_playlists_user_favorite ON playlists(user_id) WHERE is_favorite = TRUE;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_favorites_user_id ON favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_favorites_song_id ON favorites(song_id);
CREATE INDEX IF NOT EXISTS idx_favorites_album_id ON favorites(album_id);
CREATE INDEX IF NOT EXISTS idx_playlists_is_favorite ON playlists(is_favorite);

-- Create favorites playlist for all existing users
INSERT INTO playlists (title, description, owner, user_id, is_favorite, is_public, created_at, updated_at)
SELECT
    'Favorites',
    'Your favorited songs and albums',
    u.username,
    u.id,
    TRUE,
    FALSE,
    NOW(),
    NOW()
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM playlists p WHERE p.user_id = u.id AND p.is_favorite = TRUE
);
