DO $$
BEGIN
    IF to_regclass('public.index_1753189169') IS NOT NULL THEN
        RAISE NOTICE 'Reindexing index_1753189169';
        REINDEX INDEX index_1753189169;
    END IF;
    IF to_regclass('public.index_272307221') IS NOT NULL THEN
        RAISE NOTICE 'Reindexing index_272307221';
        REINDEX INDEX index_272307221;
    END IF;
END $$;
