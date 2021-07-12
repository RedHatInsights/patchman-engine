DO
$$
    DECLARE
        stmt TEXT;
    BEGIN
        FOR stmt IN (SELECT 'DROP ' || case when prokind = 'f' then 'FUNCTION ' else 'PROCEDURE ' end
                            || ns.nspname || '.' || proname || '(' || oidvectortypes(proargtypes) || ') CASCADE;'
                     FROM pg_proc
                              INNER JOIN pg_namespace ns ON (pg_proc.pronamespace = ns.oid)
                     WHERE ns.nspname = 'public'
                       AND proname not like 'uuid_%'
                     ORDER BY proname)
            LOOP
                EXECUTE stmt;
            END LOOP;

        FOR stmt IN (SELECT 'DROP TABLE IF EXISTS "' || table_name || '" CASCADE;'
                     FROM information_schema.tables
                     WHERE table_schema = (SELECT current_schema())
                       AND table_type = 'BASE TABLE')
            LOOP
                EXECUTE stmt;
            END LOOP;

    END;
$$ language plpgsql;
