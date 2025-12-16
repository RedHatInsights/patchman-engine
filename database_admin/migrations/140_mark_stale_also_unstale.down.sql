CREATE OR REPLACE FUNCTION mark_stale_systems(mark_limit integer)
    RETURNS INTEGER
AS
$fun$
DECLARE
    marked integer;
BEGIN
    WITH ids AS (
        SELECT rh_account_id, id
        FROM system_platform
        WHERE stale_warning_timestamp < now()
          AND stale = false
        ORDER BY rh_account_id, id FOR UPDATE OF system_platform
        LIMIT mark_limit
    )
    UPDATE system_platform sp
    SET stale = true
    FROM ids
    WHERE sp.rh_account_id = ids.rh_account_id
      AND sp.id = ids.id;
    GET DIAGNOSTICS marked = ROW_COUNT;
    RETURN marked;
END;
$fun$ LANGUAGE plpgsql;

