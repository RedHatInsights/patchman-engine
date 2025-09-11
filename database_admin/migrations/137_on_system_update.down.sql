CREATE OR REPLACE FUNCTION on_system_update()
-- this trigger updates advisory_account_data when server changes its stale flag
    RETURNS TRIGGER
AS
$system_update$
DECLARE
    was_counted  BOOLEAN;
    should_count BOOLEAN;
    change       INT;
BEGIN
    -- Ignore not yet evaluated systems
    IF TG_OP != 'UPDATE' OR NEW.last_evaluation IS NULL THEN
        RETURN NEW;
    END IF;

    was_counted := OLD.stale = FALSE;
    should_count := NEW.stale = FALSE;

    -- Determine what change we are performing
    IF was_counted and NOT should_count THEN
        change := -1;
    ELSIF NOT was_counted AND should_count THEN
        change := 1;
    ELSE
        -- No change
        RETURN NEW;
    END IF;

    -- insert/update advisories linked to the server
    INSERT
      INTO advisory_account_data (advisory_id, rh_account_id, systems_installable, systems_applicable)
    SELECT sa.advisory_id, NEW.rh_account_id,
           case when sa.status_id = 0 then change else 0 end as systems_installable,
           change as systems_applicable
      FROM system_advisories sa
     WHERE sa.system_id = NEW.id AND sa.rh_account_id = NEW.rh_account_id
        ON CONFLICT (advisory_id, rh_account_id) DO UPDATE
           SET systems_installable = advisory_account_data.systems_installable + EXCLUDED.systems_installable,
               systems_applicable = advisory_account_data.systems_applicable + EXCLUDED.systems_applicable;
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;

SELECT create_table_partition_triggers('system_platform_on_update',
                                       $$AFTER UPDATE$$,
                                       'system_platform',
                                       $$FOR EACH ROW EXECUTE PROCEDURE on_system_update()$$);
