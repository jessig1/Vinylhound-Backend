-- Drop the album_collections table and related objects
DROP TRIGGER IF EXISTS album_collections_updated_at_trigger ON album_collections;
DROP FUNCTION IF EXISTS update_album_collections_updated_at();
DROP TABLE IF EXISTS album_collections;
