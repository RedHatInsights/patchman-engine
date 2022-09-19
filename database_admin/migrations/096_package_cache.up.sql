ALTER TABLE rh_account ADD valid_package_cache BOOLEAN NOT NULL DEFAULT FALSE;
GRANT UPDATE ON rh_account TO vmaas_sync;

CREATE TABLE IF NOT EXISTS package_account_data
(
    package_name_id          BIGINT NOT NULL,
    rh_account_id            INT NOT NULL,
    systems_installed        INT NOT NULL DEFAULT 0,
    systems_updatable        INT NOT NULL DEFAULT 0,
    CONSTRAINT package_name_id
        FOREIGN KEY (package_name_id)
            REFERENCES package_name (id),
    CONSTRAINT rh_account_id
        FOREIGN KEY (rh_account_id)
            REFERENCES rh_account (id),
    UNIQUE (package_name_id, rh_account_id),
    PRIMARY KEY (rh_account_id, package_name_id)
) WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
  TABLESPACE pg_default;

-- vmaas_sync user is used for admin api and for cronjobs, it needs to update counts
GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO vmaas_sync;
GRANT SELECT ON package_account_data TO evaluator;
GRANT SELECT ON package_account_data TO listener;
GRANT SELECT ON package_account_data TO manager;

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

CREATE OR REPLACE FUNCTION refresh_all_cached_counts()
    RETURNS void AS
$refresh_all_cached_counts$
BEGIN
    PERFORM refresh_system_caches(NULL, NULL);
    PERFORM refresh_advisory_caches(NULL, NULL);
    PERFORM refresh_packages_caches(NULL);
END;
$refresh_all_cached_counts$
    LANGUAGE 'plpgsql';
