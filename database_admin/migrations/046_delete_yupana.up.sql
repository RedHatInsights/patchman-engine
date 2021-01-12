DO
$$
    DECLARE
        ids     int[];
        deleted int = 1;
    BEGIN
        WHILE deleted > 0
            LOOP
                ids := ARRAY(SELECT id
                             FROM system_platform
                             WHERE reporter_id = (select id from reporter where name = 'yupana')
                             LIMIT 1000
                    );
                deleted := delete_systems(ids);
                RAISE NOTICE 'Deleted yupana systems: %', text(deleted);
            END LOOP;
    END;
$$ LANGUAGE plpgsql;