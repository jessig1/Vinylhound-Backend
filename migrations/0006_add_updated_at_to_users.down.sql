DROP TRIGGER IF EXISTS users_updated_at ON users;
DROP FUNCTION IF EXISTS update_users_updated_at();
ALTER TABLE users DROP COLUMN IF EXISTS updated_at;
