CREATE OR REPLACE FUNCTION backfill_account_advisory(rh_account_id_in INTEGER)
    RETURNS VOID AS
$backfill$
BEGIN
    PERFORM refresh_account_advisory_caches_multi(NULL, rh_account_id_in);
END;
$backfill$ LANGUAGE plpgsql;