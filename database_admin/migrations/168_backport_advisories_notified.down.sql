CREATE OR REPLACE FUNCTION backfill_account_advisory(rh_account_id_in INTEGER)
    RETURNS VOID AS
$backfill$
BEGIN
    PERFORM refresh_account_advisory_caches_multi(NULL, rh_account_id_in);
END;
$backfill$ LANGUAGE plpgsql;

SELECT drop_table_partition_triggers('account_advisory_sync_notified_insert',
                                     $$BEFORE INSERT$$,
                                     'account_advisory',
                                     $$FOR EACH ROW EXECUTE PROCEDURE sync_account_advisory_notified_on_insert()$$);
                                     
DROP FUNCTION IF EXISTS sync_account_advisory_notified_on_insert();
