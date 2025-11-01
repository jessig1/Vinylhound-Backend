-- Create concerts table
CREATE TABLE IF NOT EXISTS concerts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    venue_id BIGINT NOT NULL REFERENCES venues(id) ON DELETE RESTRICT,
    artist_name VARCHAR(255) NOT NULL,
    name VARCHAR(500) NOT NULL,
    date TIMESTAMP WITH TIME ZONE NOT NULL,
    ticket_price DECIMAL(10,2),
    notes TEXT,
    attended BOOLEAN DEFAULT FALSE,
    rating INT CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5)),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries
CREATE INDEX idx_concerts_user_id ON concerts(user_id);
CREATE INDEX idx_concerts_venue_id ON concerts(venue_id);
CREATE INDEX idx_concerts_artist_name ON concerts(artist_name);
CREATE INDEX idx_concerts_date ON concerts(date);
CREATE INDEX idx_concerts_user_date ON concerts(user_id, date DESC);

-- Index for upcoming concerts query
CREATE INDEX idx_concerts_user_upcoming ON concerts(user_id, date)
    WHERE attended = FALSE AND date >= CURRENT_TIMESTAMP;

-- Index for past concerts query
CREATE INDEX idx_concerts_user_past ON concerts(user_id, date DESC)
    WHERE attended = TRUE;

-- Optional: Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_concerts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER concerts_updated_at_trigger
    BEFORE UPDATE ON concerts
    FOR EACH ROW
    EXECUTE FUNCTION update_concerts_updated_at();
