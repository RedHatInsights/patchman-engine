DO $$
BEGIN
    DROP EXTENSION IF EXISTS pg_repack;
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_repack') THEN
        CREATE EXTENSION pg_repack;
    END IF;
END
$$;
