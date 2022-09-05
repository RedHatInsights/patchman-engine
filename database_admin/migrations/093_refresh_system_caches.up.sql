CREATE OR REPLACE FUNCTION refresh_system_caches(system_id_in BIGINT DEFAULT NULL,
                                                 rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS INTEGER AS
$refresh_system$
DECLARE
    COUNT INTEGER;
BEGIN
    WITH system_advisories_count AS (
        SELECT asp.rh_account_id, asp.id,
               COUNT(advisory_id) as total,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 1) AS enhancement,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 2) AS bugfix,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 3) as security
          FROM system_platform asp  -- this table ensures even systems without any system_advisories are in results
          LEFT JOIN system_advisories sa
            ON asp.rh_account_id = sa.rh_account_id AND asp.id = sa.system_id and sa.when_patched IS NULL
          LEFT JOIN advisory_metadata am
            ON sa.advisory_id = am.id
         WHERE (asp.id = system_id_in OR system_id_in IS NULL)
           AND (asp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
         GROUP BY asp.rh_account_id, asp.id
         ORDER BY asp.rh_account_id, asp.id
    )
        UPDATE system_platform sp
           SET advisory_count_cache = sc.total,
               advisory_enh_count_cache = sc.enhancement,
               advisory_bug_count_cache = sc.bugfix,
               advisory_sec_count_cache = sc.security
          FROM system_advisories_count sc
         WHERE sp.rh_account_id = sc.rh_account_id AND sp.id = sc.id
           AND (sp.id = system_id_in OR system_id_in IS NULL)
           AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);

    GET DIAGNOSTICS COUNT = ROW_COUNT;
    RETURN COUNT;
END;
$refresh_system$ LANGUAGE plpgsql;
