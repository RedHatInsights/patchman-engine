CREATE OR REPLACE FUNCTION backfill_account_advisory(rh_account_id_in INTEGER)
    RETURNS VOID AS
$backfill$
BEGIN
    PERFORM refresh_account_advisory_caches_multi(NULL, rh_account_id_in);

    -- copy `notified` for all `workspace_id`s per account
    UPDATE account_advisory aa
    SET notified = aad.notified
        FROM advisory_account_data aad
    WHERE aa.advisory_id = aad.advisory_id
        AND aa.rh_account_id = aad.rh_account_id
        AND aa.rh_account_id = rh_account_id_in
        AND aad.notified IS NOT NULL;
END;
$backfill$ LANGUAGE plpgsql;
