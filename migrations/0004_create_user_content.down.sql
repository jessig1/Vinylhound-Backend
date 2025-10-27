DROP TRIGGER IF EXISTS user_content_updated_at ON user_content;
DROP FUNCTION IF EXISTS update_user_content_updated_at();
DROP INDEX IF EXISTS idx_user_content_user_id;
DROP TABLE IF EXISTS user_content;
