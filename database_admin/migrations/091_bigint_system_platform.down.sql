ALTER TABLE system_platform ALTER COLUMN id TYPE INT,
                            ALTER COLUMN baseline_id TYPE INT;

-- count system advisories according to advisory type
DROP FUNCTION IF EXISTS system_advisories_count(system_id_in BIGINT, advisory_type_id_in INT);
CREATE OR REPLACE FUNCTION system_advisories_count(system_id_in INT, advisory_type_id_in INT DEFAULT NULL)
    RETURNS INT AS
$system_advisories_count$
DECLARE
    result_cnt INT;
BEGIN
    SELECT COUNT(advisory_id)
    FROM system_advisories sa
             JOIN advisory_metadata am ON sa.advisory_id = am.id
    WHERE (am.advisory_type_id = advisory_type_id_in OR advisory_type_id_in IS NULL)
      AND sa.system_id = system_id_in
      AND sa.when_patched IS NULL
    INTO result_cnt;
    RETURN result_cnt;
END;
$system_advisories_count$ LANGUAGE 'plpgsql';

DROP FUNCTION IF EXISTS refresh_system_caches(system_id_in BIGINT, rh_account_id_in INTEGER);
CREATE OR REPLACE FUNCTION refresh_system_caches(system_id_in INTEGER DEFAULT NULL,
                                                 rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS INTEGER AS
$refresh_system$
DECLARE
    COUNT INTEGER;
BEGIN
    WITH to_update AS (
        SELECT sp.rh_account_id, sp.id
        FROM system_platform sp
        WHERE (sp.id = system_id_in OR system_id_in IS NULL)
          AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        ORDER BY sp.rh_account_id, sp.id
            FOR UPDATE OF sp
        )
        UPDATE system_platform sp
           SET advisory_count_cache = system_advisories_count(sp.id, NULL),
               advisory_enh_count_cache = system_advisories_count(sp.id, 1),
               advisory_bug_count_cache = system_advisories_count(sp.id, 2),
               advisory_sec_count_cache = system_advisories_count(sp.id, 3)
          FROM to_update to_up
         WHERE sp.rh_account_id = to_up.rh_account_id AND sp.id = to_up.id;
    GET DIAGNOSTICS COUNT = ROW_COUNT;
    RETURN COUNT;
END;
$refresh_system$ LANGUAGE plpgsql;

-- update system advisories counts (all and according types)
DROP FUNCTION IF EXISTS update_system_caches(system_id_in BIGINT);
CREATE OR REPLACE FUNCTION update_system_caches(system_id_in INT)
    RETURNS VOID AS
$update_system_caches$
BEGIN
    PERFORM refresh_system_caches(system_id_in, NULL);
END;
$update_system_caches$
    LANGUAGE 'plpgsql';
