DROP FUNCTION refresh_system_caches(system_id_in integer, rh_account_id_in integer);


CREATE OR REPLACE FUNCTION refresh_system_caches(system_id_in INTEGER DEFAULT NULL,
                                                 rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS INTEGER AS
$refresh_system$
DECLARE
    COUNT INTEGER;
BEGIN
    WITH to_update AS (
        SELECT sp.id
        FROM system_platform sp
        WHERE (sp.id = system_id_in OR system_id_in IS NULL)
          AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        ORDER BY sp.rh_account_id, sp.id
            FOR UPDATE OF sp
    ),
         updated as (
             UPDATE system_platform sp
                 SET advisory_count_cache = system_advisories_count(sp.id, NULL),
                     advisory_enh_count_cache = system_advisories_count(sp.id, 1),
                     advisory_bug_count_cache = system_advisories_count(sp.id, 2),
                     advisory_sec_count_cache = system_advisories_count(sp.id, 3)
                 FROM to_update to_up
                 WHERE sp.id = to_up.id
                 RETURNING sp.id)
    SELECT count(*)
    FROM updated
    INTO COUNT;
    RETURN COUNT;
END;
$refresh_system$ LANGUAGE plpgsql;