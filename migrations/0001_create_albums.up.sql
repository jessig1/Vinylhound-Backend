CREATE TABLE IF NOT EXISTS albums (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    artist TEXT NOT NULL,
    title TEXT NOT NULL,
    release_year INTEGER NOT NULL CHECK (release_year > 0),
    tracks JSONB NOT NULL DEFAULT '[]'::jsonb,
    genres JSONB NOT NULL DEFAULT '[]'::jsonb,
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_albums_user_id ON albums(user_id);
