-- set_last_updated
CREATE OR REPLACE FUNCTION on_system_timestamp_update()
    RETURNS TRIGGER AS
$set_last_updated$
BEGIN
    IF (TG_OP = 'UPDATE') OR (TG_OP = 'INSERT') THEN
        NEW.stale := COALESCE(NEW.stale_warning_timestamp < CURRENT_TIMESTAMP, FALSE);
    END IF;
    RETURN NEW;
END;
$set_last_updated$
    LANGUAGE 'plpgsql';


CREATE TRIGGER system_platform_on_update_timestamp
    BEFORE UPDATE OF stale_timestamp, stale_warning_timestamp, culled_timestamp
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE on_system_timestamp_update();



CREATE OR REPLACE FUNCTION delete_culled_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    culled integer;
BEGIN
    WITH ids AS (SELECT inventory_id
                 FROM system_platform
                 WHERE culled_timestamp < now()
                 ORDER BY id FOR UPDATE OF system_platform
    ),
         deleted AS (SELECT delete_system(inventory_id) from ids)
    SELECT count(*)
    FROM deleted
    INTO culled;
    RETURN culled;
END;
$fun$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION mark_stale_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    marked integer;
BEGIN
    WITH ids AS (SELECT id
                 FROM system_platform
                 WHERE stale_warning_timestamp < now()
                   AND stale = false
                 ORDER BY id FOR UPDATE OF system_platform
    ),
         updated as (
             UPDATE system_platform
                 SET stale = true
                 FROM ids
                 RETURNING ids.id
         )
    SELECT count(*)
    FROM updated
    INTO marked;
    RETURN marked;
END;
$fun$ LANGUAGE plpgsql;