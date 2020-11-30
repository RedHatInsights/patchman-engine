DO
$$
DECLARE
    marked integer;
BEGIN
    WITH ids AS (
        SELECT rh_account_id, sp.id AS id FROM system_platform sp
        LEFT JOIN inventory.hosts ih ON sp.inventory_id = ih.id::text
        WHERE ih.id IS NULL
        ORDER BY rh_account_id, sp.id
        )
        UPDATE system_platform sp
        SET culled_timestamp = now()
        FROM ids
        WHERE sp.id = ids.id AND sp.rh_account_id = ids.rh_account_id;
EXCEPTION
     -- inventory.hosts is not in our schema, skip if not presented
     WHEN others THEN
        RAISE NOTICE 'Unable mark culled_timestamp in systems data!';
END
$$ LANGUAGE plpgsql;
