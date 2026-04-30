-- just to trigger the migration
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'inventory' AND table_name = 'hosts') THEN
        PERFORM 1 FROM inventory.hosts LIMIT 1;
    END IF;
END
$$;
