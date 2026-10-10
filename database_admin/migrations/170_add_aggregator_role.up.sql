CREATE OR REPLACE FUNCTION revoke_table_partitions(perms text, tbl regclass, grantie text)
    RETURNS VOID AS
$$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(tbl)
    LOOP
        EXECUTE 'REVOKE ' || perms || ' ON TABLE ' || r.relname || ' FROM ' || grantie;
    END LOOP;
    EXECUTE 'REVOKE ' || perms || ' ON TABLE ' || text(tbl) || ' FROM ' || grantie;
END;
$$ LANGUAGE plpgsql;

SELECT revoke_table_partitions('INSERT, UPDATE, DELETE', 'account_advisory', 'manager');
SELECT revoke_table_partitions('INSERT, UPDATE, DELETE', 'account_advisory', 'evaluator');
SELECT revoke_table_partitions('INSERT, UPDATE, DELETE', 'account_advisory', 'listener');
SELECT revoke_table_partitions('INSERT, UPDATE', 'account_advisory', 'vmaas_sync');

SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'aggregator');

GRANT SELECT ON ALL TABLES IN SCHEMA public TO aggregator;

GRANT EXECUTE ON FUNCTION refresh_account_advisory_caches_multi(INTEGER[], INTEGER) TO aggregator;
GRANT EXECUTE ON FUNCTION refresh_account_advisory_caches(INTEGER, INTEGER) TO aggregator;
