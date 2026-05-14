DROP FUNCTION IF EXISTS backfill_account_advisory(INTEGER);

DROP FUNCTION IF EXISTS refresh_account_advisory_caches(INTEGER, INTEGER);

DROP FUNCTION IF EXISTS refresh_account_advisory_caches_multi(INTEGER[], INTEGER);

DO $$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child ON pg_inherits.inhrelid = child.oid
              WHERE parent.relname = 'system_inventory'
    LOOP
        EXECUTE 'DROP TRIGGER IF EXISTS system_inventory_on_update_account_advisory'
                || substr(r.relname, length('system_inventory') + 1)
                || ' ON ' || r.relname;
    END LOOP;
END $$;

DROP FUNCTION IF EXISTS on_system_update_account_advisory();

DROP INDEX IF EXISTS account_advisory_systems_applicable_idx;
DROP INDEX IF EXISTS account_advisory_systems_installable_idx;

TRUNCATE account_advisory;
