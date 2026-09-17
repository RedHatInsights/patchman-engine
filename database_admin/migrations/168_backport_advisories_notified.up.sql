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

CREATE OR REPLACE FUNCTION sync_account_advisory_notified_on_insert()
    RETURNS TRIGGER AS
$sync_notified_insert$
BEGIN
    IF NEW.notified IS NULL THEN
        SELECT notified INTO NEW.notified
        FROM account_advisory
        WHERE rh_account_id = NEW.rh_account_id
          AND advisory_id = NEW.advisory_id
          AND notified IS NOT NULL
        LIMIT 1;
    END IF;
    RETURN NEW;
END;
$sync_notified_insert$ LANGUAGE plpgsql;

SELECT create_table_partition_triggers('account_advisory_sync_notified_insert',
                                       $$BEFORE INSERT$$,
                                       'account_advisory',
                                       $$FOR EACH ROW EXECUTE PROCEDURE sync_account_advisory_notified_on_insert()$$);
