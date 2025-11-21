DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'repack') THEN
        GRANT USAGE ON SCHEMA repack TO vmaas_sync;
    END IF;
END
$$;
