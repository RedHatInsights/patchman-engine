SELECT revoke_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'aggregator');
REVOKE EXECUTE ON FUNCTION refresh_account_advisory_caches_multi(INTEGER[], INTEGER) FROM aggregator;
REVOKE EXECUTE ON FUNCTION refresh_account_advisory_caches(INTEGER, INTEGER) FROM aggregator;
REVOKE SELECT ON ALL TABLES IN SCHEMA public FROM aggregator;

SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'manager');
SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'evaluator');
SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'listener');
SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'vmaas_sync');

DROP FUNCTION IF EXISTS revoke_table_partitions(text, regclass, text);
