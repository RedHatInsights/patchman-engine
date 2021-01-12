CREATE OR REPLACE FUNCTION delete_systems(ids INT[])
    RETURNS INTEGER
AS
$$
DECLARE
    system_ids INT[];
    tmp_cnt    INTEGER;
BEGIN
    system_ids := ARRAY(
            SELECT id
            FROM system_platform
            WHERE id = ANY (ids)
                FOR UPDATE OF system_platform
        );

    UPDATE system_platform sp
    SET opt_out = true
    WHERE id = ANY (ids);

    GET DIAGNOSTICS tmp_cnt = row_count;
    RAISE NOTICE 'Marked systems %', text(tmp_cnt);


    DELETE
    FROM system_advisories
    WHERE system_id = ANY (system_ids);

    GET DIAGNOSTICS tmp_cnt = row_count;
    RAISE NOTICE 'Deleted system_advisories %', text(tmp_cnt);


    DELETE
    FROM system_repo
    WHERE system_id = ANY (system_ids);

    GET DIAGNOSTICS tmp_cnt = row_count;
    RAISE NOTICE 'Deleted system_repos %', text(tmp_cnt);


    DELETE
    FROM system_package
    WHERE system_id = ANY (system_ids);

    GET DIAGNOSTICS tmp_cnt = row_count;
    RAISE NOTICE 'Deleted system_packages %', text(tmp_cnt);


    DELETE
    FROM system_platform
    WHERE id = ANY (system_ids);

    GET DIAGNOSTICS tmp_cnt = row_count;
    RAISE NOTICE 'Deleted system_platform %', text(tmp_cnt);


    RETURN tmp_cnt;
END
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION delete_culled_systems(delete_limit INTEGER)
    RETURNS INTEGER
AS
$fun$
DECLARE
    ids INTEGER[];
BEGIN
    ids := ARRAY(
            SELECT id
            FROM system_platform
            WHERE culled_timestamp < now()
            ORDER BY id
            LIMIT delete_limit
        );
    return delete_systems(ids);
END;
$fun$ LANGUAGE plpgsql;

