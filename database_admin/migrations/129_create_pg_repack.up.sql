DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_repack') THEN
        CREATE EXTENSION IF NOT EXISTS pg_repack;
    END IF;
END
$$;
