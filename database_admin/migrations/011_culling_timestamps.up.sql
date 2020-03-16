-- mark all systems as fresh, since we had the wrong implementation for function for marking
UPDATE system_platform
SET stale = false;

CREATE OR REPLACE FUNCTION mark_stale_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    marked integer;
BEGIN
    with updated as (UPDATE system_platform
        SET stale = true
        -- Systems AFTER stale_warning timestamp
        WHERE now() > stale_warning_timestamp
        RETURNING id
    )
    select count(*)
    from updated
    INTO marked;
    return marked;
END;
$fun$ LANGUAGE plpgsql;

-- Re-mark systems that have been marked as stale correctly
SELECT * from mark_stale_systems();