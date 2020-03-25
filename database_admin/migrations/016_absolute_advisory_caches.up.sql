CREATE OR REPLACE FUNCTION refresh_advisory_caches_multi(advisory_ids_in INTEGER[] DEFAULT NULL,
                                                         rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_advisory$
BEGIN
    -- Lock rows
    PERFORM aad.rh_account_id, aad.advisory_id
    FROM advisory_account_data aad
    WHERE (aad.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
      AND (aad.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        FOR UPDATE OF aad;

    WITH current_counts AS (
        SELECT sa.advisory_id, sp.rh_account_id, count(sa.system_id) as systems_affected
        FROM system_advisories sa
                 INNER JOIN
             system_platform sp ON sa.system_id = sp.id
        WHERE sp.last_evaluation IS NOT NULL
          AND sp.opt_out = FALSE
          AND sp.stale = FALSE
          AND sa.when_patched IS NULL
          AND (sa.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
          AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        GROUP BY sa.advisory_id, sp.rh_account_id
    ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_id, rh_account_id, systems_affected
                 FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE (advisory_id, rh_account_id) NOT IN (SELECT advisory_id, rh_account_id FROM current_counts)
      AND (advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
      AND (rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);
END;
$refresh_advisory$ language plpgsql;

CREATE OR REPLACE FUNCTION refresh_advisory_caches(advisory_id_in INTEGER DEFAULT NULL,
                                                   rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_advisory$
BEGIN
    IF advisory_id_in IS NOT NULL THEN
        PERFORM refresh_advisory_caches_multi(ARRAY [advisory_id_in], rh_account_id_in);
    ELSE
        PERFORM refresh_advisory_caches_multi(NULL, rh_account_id_in);
    END IF;
END;
$refresh_advisory$ language plpgsql;

GRANT SELECT, UPDATE ON rh_account TO evaluator;