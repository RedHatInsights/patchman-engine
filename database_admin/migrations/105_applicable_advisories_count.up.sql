REVOKE UPDATE (advisory_count_cache,
              advisory_enh_count_cache,
              advisory_bug_count_cache,
              advisory_sec_count_cache), DELETE ON system_platform FROM manager;

ALTER TABLE system_platform RENAME COLUMN advisory_count_cache TO installable_advisory_count_cache;
ALTER TABLE system_platform RENAME COLUMN advisory_enh_count_cache TO installable_advisory_enh_count_cache;
ALTER TABLE system_platform RENAME COLUMN advisory_bug_count_cache TO installable_advisory_bug_count_cache;
ALTER TABLE system_platform RENAME COLUMN advisory_sec_count_cache TO installable_advisory_sec_count_cache;
ALTER TABLE system_platform ADD COLUMN applicable_advisory_count_cache     INT NOT NULL DEFAULT 0,
                            ADD COLUMN applicable_advisory_enh_count_cache INT NOT NULL DEFAULT 0,
                            ADD COLUMN applicable_advisory_bug_count_cache INT NOT NULL DEFAULT 0,
                            ADD COLUMN applicable_advisory_sec_count_cache INT NOT NULL DEFAULT 0;

GRANT UPDATE (installable_advisory_count_cache,
              installable_advisory_enh_count_cache,
              installable_advisory_bug_count_cache,
              installable_advisory_sec_count_cache), DELETE ON system_platform TO manager;
GRANT UPDATE (applicable_advisory_count_cache,
              applicable_advisory_enh_count_cache,
              applicable_advisory_bug_count_cache,
              applicable_advisory_sec_count_cache), DELETE ON system_platform TO manager;

CREATE OR REPLACE FUNCTION refresh_system_caches(system_id_in BIGINT DEFAULT NULL,
                                                 rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS INTEGER AS
$refresh_system$
DECLARE
    COUNT INTEGER;
BEGIN
    WITH system_advisories_count AS (
        SELECT asp.rh_account_id, asp.id,
               COUNT(advisory_id) FILTER (WHERE sa.status_id = 0) as installable_total,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 1 AND sa.status_id = 0) AS installable_enhancement,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 2 AND sa.status_id = 0) AS installable_bugfix,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 3 AND sa.status_id = 0) as installable_security,
               COUNT(advisory_id) FILTER (WHERE sa.status_id = 1) as applicable_total,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 1 AND sa.status_id = 1) AS applicable_enhancement,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 2 AND sa.status_id = 1) AS applicable_bugfix,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 3 AND sa.status_id = 1) as applicable_security
          FROM system_platform asp  -- this table ensures even systems without any system_advisories are in results
          LEFT JOIN system_advisories sa
            ON asp.rh_account_id = sa.rh_account_id AND asp.id = sa.system_id
          LEFT JOIN advisory_metadata am
            ON sa.advisory_id = am.id
         WHERE (asp.id = system_id_in OR system_id_in IS NULL)
           AND (asp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
         GROUP BY asp.rh_account_id, asp.id
         ORDER BY asp.rh_account_id, asp.id
    )
        UPDATE system_platform sp
           SET installable_advisory_count_cache = sc.installable_total,
               installable_advisory_enh_count_cache = sc.installable_enhancement,
               installable_advisory_bug_count_cache = sc.installable_bugfix,
               installable_advisory_sec_count_cache = sc.installable_security,
               applicable_advisory_count_cache = sc.applicable_total,
               applicable_advisory_enh_count_cache = sc.applicable_enhancement,
               applicable_advisory_bug_count_cache = sc.applicable_bugfix,
               applicable_advisory_sec_count_cache = sc.applicable_security
          FROM system_advisories_count sc
         WHERE sp.rh_account_id = sc.rh_account_id AND sp.id = sc.id
           AND (sp.id = system_id_in OR system_id_in IS NULL)
           AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);

    GET DIAGNOSTICS COUNT = ROW_COUNT;
    RETURN COUNT;
END;
$refresh_system$ LANGUAGE plpgsql;
