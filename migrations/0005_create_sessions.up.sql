CREATE TABLE IF NOT EXISTS sessions (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(512) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Add comment documenting the table
COMMENT ON TABLE sessions IS 'User session tokens for authentication';
COMMENT ON COLUMN sessions.token IS 'Session token (base64 encoded random bytes)';
COMMENT ON COLUMN sessions.expires_at IS 'Expiration timestamp for the session';
