-- Create album_collections table to track user album wishlists and owned collections
-- This extends the existing favorites system with collection types
CREATE TABLE IF NOT EXISTS album_collections (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    album_id BIGINT NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    collection_type VARCHAR(20) NOT NULL CHECK (collection_type IN ('wishlist', 'owned')),
    notes TEXT,
    date_added TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    date_acquired TIMESTAMP WITH TIME ZONE, -- When they acquired it (for owned albums)
    purchase_price DECIMAL(10,2), -- Optional purchase price
    condition VARCHAR(20) CHECK (condition IS NULL OR condition IN ('mint', 'near_mint', 'very_good', 'good', 'fair', 'poor')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Prevent duplicate entries for same user, album, and collection type
    CONSTRAINT unique_user_album_collection UNIQUE (user_id, album_id, collection_type)
);

-- Indexes for performance
CREATE INDEX idx_album_collections_user_id ON album_collections(user_id);
CREATE INDEX idx_album_collections_album_id ON album_collections(album_id);
CREATE INDEX idx_album_collections_user_type ON album_collections(user_id, collection_type);
CREATE INDEX idx_album_collections_date_added ON album_collections(date_added DESC);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_album_collections_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER album_collections_updated_at_trigger
    BEFORE UPDATE ON album_collections
    FOR EACH ROW
    EXECUTE FUNCTION update_album_collections_updated_at();
