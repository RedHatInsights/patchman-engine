CREATE OR REPLACE FUNCTION delete_culled_systems(delete_limit INTEGER)
    RETURNS INTEGER
AS
$fun$
DECLARE
    n_deleted INTEGER;
BEGIN
    SELECT count(*) INTO n_deleted FROM (
                                            SELECT delete_system(inventory_id)
                                            FROM system_platform sp
                                            WHERE culled_timestamp < now()
                                            ORDER BY id
                                            LIMIT delete_limit) x;
    RETURN n_deleted;
END;
$fun$ LANGUAGE plpgsql;

DROP FUNCTION IF EXISTS delete_systems(ids INT[]);

