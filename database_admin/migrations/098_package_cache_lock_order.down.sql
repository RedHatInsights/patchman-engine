CREATE OR REPLACE FUNCTION refresh_packages_caches(rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_packages$
BEGIN
    -- lock rows
    PERFORM pad.rh_account_id, acc.id
        FROM package_account_data pad
        JOIN rh_account acc
          ON acc.id = pad.rh_account_id
        WHERE (pad.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
            FOR UPDATE OF pad, acc;

    WITH pkg_system_counts AS (
        SELECT sp.rh_account_id, spkg.name_id package_name_id,
               count(spkg.system_id) as systems_installed,
               count(spkg.system_id) filter (where spkg.latest_evra IS NOT NULL) as systems_updatable
          FROM system_platform sp
          JOIN system_package spkg
            ON sp.id = spkg.system_id AND sp.rh_account_id = spkg.rh_account_id
          JOIN rh_account acc
            ON sp.rh_account_id = acc.id
          JOIN inventory.hosts ih
            ON sp.inventory_id = ih.id
        WHERE sp.packages_installed > 0 AND sp.stale = FALSE
          AND (sp.rh_account_id = rh_account_id_in OR (rh_account_id_in IS NULL AND acc.valid_package_cache = FALSE))
        GROUP BY sp.rh_account_id, spkg.name_id
        ORDER BY sp.rh_account_id, spkg.name_id
    ),
        upserted AS (
            INSERT INTO package_account_data (package_name_id, rh_account_id, systems_installed, systems_updatable)
                 SELECT package_name_id, rh_account_id, systems_installed, systems_updatable
                   FROM pkg_system_counts
                     ON CONFLICT (package_name_id, rh_account_id) DO UPDATE SET
                        systems_installed = EXCLUDED.systems_installed,
                        systems_updatable = EXCLUDED.systems_updatable
         )
    DELETE
      FROM package_account_data
     WHERE (package_name_id, rh_account_id) NOT IN (SELECT package_name_id, rh_account_id FROM pkg_system_counts)
       AND (rh_account_id = rh_account_id_in OR rh_account_id IN (SELECT rh_account_id FROM pkg_system_counts));
    UPDATE rh_account acc
       SET valid_package_cache = TRUE
     WHERE (acc.id = rh_account_id_in OR rh_account_id_in IS NULL);

END;
$refresh_packages$ LANGUAGE plpgsql;
