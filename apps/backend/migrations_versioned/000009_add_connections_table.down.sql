-- Rollback migration for connections table

DROP TRIGGER IF EXISTS trigger_connections_updated_at ON connections;
DROP FUNCTION IF EXISTS update_connections_updated_at();
DROP TABLE IF EXISTS connections CASCADE;
