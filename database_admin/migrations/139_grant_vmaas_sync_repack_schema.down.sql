DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'repack') THEN
        REVOKE USAGE ON SCHEMA repack FROM vmaas_sync;
    END IF;
END
$$;
