CREATE TABLE IF NOT EXISTS schema_migrations
(
    version bigint  NOT NULL,
    dirty   boolean NOT NULL,
    PRIMARY KEY (version)
) TABLESPACE pg_default;


INSERT INTO schema_migrations
VALUES (38, false);

-- ---------------------------------------------------------------------------
-- Functions
-- ---------------------------------------------------------------------------

-- empty
CREATE OR REPLACE FUNCTION empty(t TEXT)
    RETURNS BOOLEAN as
$empty$
BEGIN
    RETURN t ~ '^[[:space:]]*$';
END;
$empty$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION ternary(cond BOOL, iftrue ANYELEMENT, iffalse ANYELEMENT)
    RETURNS ANYELEMENT
AS
$$
SELECT CASE WHEN cond = TRUE THEN iftrue else iffalse END;
$$ LANGUAGE SQL IMMUTABLE;

-- set_first_reported
CREATE OR REPLACE FUNCTION set_first_reported()
    RETURNS TRIGGER AS
$set_first_reported$
BEGIN
    IF NEW.first_reported IS NULL THEN
        NEW.first_reported := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$set_first_reported$
    LANGUAGE 'plpgsql';

-- set_last_updated
CREATE OR REPLACE FUNCTION set_last_updated()
    RETURNS TRIGGER AS
$set_last_updated$
BEGIN
    IF (TG_OP = 'UPDATE') OR
       NEW.last_updated IS NULL THEN
        NEW.last_updated := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$set_last_updated$
    LANGUAGE 'plpgsql';

-- check_unchanged
CREATE OR REPLACE FUNCTION check_unchanged()
    RETURNS TRIGGER AS
$check_unchanged$
BEGIN
    IF (TG_OP = 'INSERT') AND
       NEW.unchanged_since IS NULL THEN
        NEW.unchanged_since := CURRENT_TIMESTAMP;
    END IF;
    IF (TG_OP = 'UPDATE') AND
       NEW.json_checksum <> OLD.json_checksum THEN
        NEW.unchanged_since := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$check_unchanged$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION on_system_update()
    RETURNS TRIGGER
AS
$system_update$
DECLARE
    was_counted  BOOLEAN;
    should_count BOOLEAN;
    change       INT;
BEGIN
    -- Ignore not yet evaluated systems
    IF TG_OP != 'UPDATE' OR NEW.last_evaluation IS NULL THEN
        RETURN NEW;
    END IF;

    was_counted := OLD.opt_out = FALSE AND OLD.stale = FALSE;
    should_count := NEW.opt_out = FALSE AND NEW.stale = FALSE;

    -- Determine what change we are performing
    IF was_counted and NOT should_count THEN
        change := -1;
    ELSIF NOT was_counted AND should_count THEN
        change := 1;
    ELSE
        -- No change
        RETURN NEW;
    END IF;

    -- Select all changed rows, lock them
    WITH to_update_advisories AS (
        SELECT aad.advisory_id,
               aad.rh_account_id,
               -- Desired count depends on old count + change
               aad.systems_affected + change                                                    as systems_affected_dst,
               -- Divergent count is the same, only depends on advisory_account_data status being different
               aad.systems_status_divergent + ternary(aad.status_id != sa.status_id, change, 0) as divergent
        FROM advisory_account_data aad
                 INNER JOIN system_advisories sa ON aad.advisory_id = sa.advisory_id
             -- Filter advisory_account_data only for advisories affectign this system & belonging to system account
        WHERE aad.rh_account_id = NEW.rh_account_id
          AND sa.system_id = NEW.id
          AND sa.when_patched IS NULL
        ORDER BY aad.advisory_id FOR UPDATE OF aad),
         -- Where count > 0, update existing rows
         update AS (
             UPDATE advisory_account_data aad
                 SET systems_affected = ta.systems_affected_dst,
                     systems_status_divergent = ta.divergent
                 FROM to_update_advisories ta
                 WHERE aad.advisory_id = ta.advisory_id
                     AND aad.rh_account_id = NEW.rh_account_id
                     AND ta.systems_affected_dst > 0
         ),
         -- Where count = 0, delete existing rows
         delete AS (
             DELETE
                 FROM advisory_account_data aad
                     USING to_update_advisories ta
                     WHERE aad.rh_account_id = ta.rh_account_id
                         AND aad.advisory_id = ta.advisory_id
                         AND ta.systems_affected_dst <= 0
         )
         -- If we have system affected && no exisiting advisory_account_data entry, we insert new rows
    INSERT
    INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
    SELECT sa.advisory_id, NEW.rh_account_id, 1
    FROM system_advisories sa
    WHERE sa.system_id = NEW.id
      AND sa.when_patched IS NULL
      AND change > 0
      -- We system_advisory pairs which don't already have rows in to_update_advisories
      AND (NEW.rh_account_id, sa.advisory_id) NOT IN (
        SELECT ta.rh_account_id, ta.advisory_id
        FROM to_update_advisories ta
    )
    ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET systems_affected = advisory_account_data.systems_affected + EXCLUDED.systems_affected;
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;

-- count system advisories according to advisory type
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

-- update system advisories counts (all and according types)
CREATE OR REPLACE FUNCTION update_system_caches(system_id_in INT)
    RETURNS VOID AS
$update_system_caches$
BEGIN
    PERFORM refresh_system_caches(system_id_in, NULL);
END;
$update_system_caches$
    LANGUAGE 'plpgsql';

-- refresh_all_cached_counts
-- WARNING: executing this procedure takes long time,
--          use only when necessary, e.g. during upgrade to populate initial caches
CREATE OR REPLACE FUNCTION refresh_all_cached_counts()
    RETURNS void AS
$refresh_all_cached_counts$
BEGIN
    PERFORM refresh_system_caches(NULL, NULL);
    PERFORM refresh_advisory_caches(NULL, NULL);
END;
$refresh_all_cached_counts$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION refresh_account_cached_counts(rh_account_in varchar)
    RETURNS void AS
$refresh_account_cached_counts$
DECLARE
    rh_account_id_in INT;
BEGIN
    -- update advisory count for ordered systems
    SELECT id FROM rh_account WHERE name = rh_account_in INTO rh_account_id_in;

    PERFORM refresh_system_caches(NULL, rh_account_id_in);
    PERFORM refresh_advisory_caches(NULL, rh_account_id_in);
END;
$refresh_account_cached_counts$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION refresh_advisory_cached_counts(advisory_name varchar)
    RETURNS void AS
$refresh_advisory_cached_counts$
DECLARE
    advisory_id_id INT;
BEGIN
    -- update system count for advisory
    SELECT id FROM advisory_metadata WHERE name = advisory_name INTO advisory_id_id;

    PERFORM refresh_advisory_caches(advisory_id_id, NULL);
END;
$refresh_advisory_cached_counts$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION refresh_advisory_account_cached_counts(advisory_name varchar, rh_account_name varchar)
    RETURNS void AS
$refresh_advisory_account_cached_counts$
DECLARE
    advisory_md_id   INT;
    rh_account_id_in INT;
BEGIN
    -- update system count for ordered advisories
    SELECT id FROM advisory_metadata WHERE name = advisory_name INTO advisory_md_id;
    SELECT id FROM rh_account WHERE name = rh_account_name INTO rh_account_id_in;

    PERFORM refresh_advisory_caches(advisory_md_id, rh_account_id_in);
END;
$refresh_advisory_account_cached_counts$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION refresh_system_cached_counts(inventory_id_in varchar)
    RETURNS void AS
$refresh_system_cached_counts$
DECLARE
    system_id int;
BEGIN

    SELECT id FROM system_platform WHERE inventory_id = inventory_id_in INTO system_id;

    PERFORM refresh_system_caches(system_id, NULL);
END;
$refresh_system_cached_counts$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION delete_system(inventory_id_in varchar)
    RETURNS TABLE
            (
                deleted_inventory_id TEXT
            )
AS
$delete_system$
DECLARE
    v_system_id  INT;
    v_account_id INT;
BEGIN
    -- opt out to refresh cache and then delete
    SELECT id, rh_account_id
    FROM system_platform
    WHERE inventory_id = inventory_id_in
        FOR UPDATE OF system_platform
    INTO v_system_id, v_account_id;

    IF v_system_id IS NULL OR v_account_id IS NULL THEN
        RAISE NOTICE 'Not found';
        RETURN;
    END IF;

    UPDATE system_platform
    SET opt_out = true
    WHERE rh_account_id = v_account_id
      AND id = v_system_id;

    DELETE
    FROM system_advisories
    WHERE system_id = v_system_id;

    DELETE
    FROM system_repo
    WHERE system_id = v_system_id;

    DELETE
    FROM system_package
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    RETURN QUERY DELETE FROM system_platform
        WHERE rh_account_id = v_account_id AND
              id = v_system_id
        RETURNING inventory_id;
END;
$delete_system$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION delete_culled_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    culled integer;
BEGIN
    WITH ids AS (SELECT inventory_id
                 FROM system_platform
                 WHERE culled_timestamp < now()
                 ORDER BY id FOR UPDATE OF system_platform
    ),
         deleted AS (SELECT delete_system(inventory_id) from ids)
    SELECT count(*)
    FROM deleted
    INTO culled;
    RETURN culled;
END;
$fun$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION mark_stale_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    marked integer;
BEGIN
    WITH ids AS (
        SELECT id
        FROM system_platform
        WHERE stale_warning_timestamp < now()
          AND stale = false
        ORDER BY id FOR UPDATE OF system_platform
    ),
         updated as (
             UPDATE system_platform
                 SET stale = true
                 FROM ids
                 WHERE system_platform.id = ids.id
                 RETURNING ids.id
         )
    SELECT count(*)
    FROM updated
    INTO marked;
    RETURN marked;
END;
$fun$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_table_partitions(tbl regclass, parts INTEGER, rest text)
    RETURNS VOID AS
$$
DECLARE
    I INTEGER;
BEGIN
    I := 0;
    WHILE I < parts
        LOOP
            EXECUTE 'CREATE TABLE ' || text(tbl) || '_' || text(I) || ' PARTITION OF ' || text(tbl) ||
                    ' FOR VALUES WITH ' || ' ( MODULUS ' || text(parts) || ', REMAINDER ' || text(I) || ')' ||
                    rest || ';';
            I = I + 1;
        END LOOP;
END;
$$ LANGUAGE plpgsql;

-- rh_account
CREATE TABLE IF NOT EXISTS rh_account
(
    id   INT GENERATED BY DEFAULT AS IDENTITY,
    name TEXT NOT NULL UNIQUE,
    CHECK (NOT empty(name)),
    PRIMARY KEY (id)
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON rh_account TO listener;
GRANT SELECT, UPDATE ON rh_account TO evaluator;

CREATE TABLE reporter
(
    id   INT  NOT NULL,
    name TEXT NOT NULL UNIQUE CHECK ( not empty(name) ),
    PRIMARY KEY (id)
);

INSERT INTO reporter (id, name)
VALUES (1, 'puptoo'),
       (2, 'rhsm-conduit'),
       (3, 'yupana')
ON CONFLICT DO NOTHING;

-- system_platform
CREATE TABLE IF NOT EXISTS system_platform
(
    id                       INT GENERATED BY DEFAULT AS IDENTITY,
    inventory_id             TEXT                     NOT NULL,
    CHECK (NOT empty(inventory_id)),
    rh_account_id            INT                      NOT NULL,
    first_reported           TIMESTAMP WITH TIME ZONE NOT NULL,
    vmaas_json               TEXT,
    json_checksum            TEXT,
    last_updated             TIMESTAMP WITH TIME ZONE NOT NULL,
    unchanged_since          TIMESTAMP WITH TIME ZONE NOT NULL,
    last_evaluation          TIMESTAMP WITH TIME ZONE,
    opt_out                  BOOLEAN                  NOT NULL DEFAULT FALSE,
    advisory_count_cache     INT                      NOT NULL DEFAULT 0,
    advisory_enh_count_cache INT                      NOT NULL DEFAULT 0,
    advisory_bug_count_cache INT                      NOT NULL DEFAULT 0,
    advisory_sec_count_cache INT                      NOT NULL DEFAULT 0,

    last_upload              TIMESTAMP WITH TIME ZONE,
    stale_timestamp          TIMESTAMP WITH TIME ZONE,
    stale_warning_timestamp  TIMESTAMP WITH TIME ZONE,
    culled_timestamp         TIMESTAMP WITH TIME ZONE,
    stale                    BOOLEAN                  NOT NULL DEFAULT false,
    display_name             TEXT                     NOT NULL CHECK (NOT empty(display_name)),
    packages_installed       INT                      NOT NULL DEFAULT 0,
    packages_updatable       INT                      NOT NULL DEFAULT 0,
    reporter_id              INT,
    PRIMARY KEY (id),
    UNIQUE (inventory_id),
    CONSTRAINT rh_account_id
        FOREIGN KEY (rh_account_id)
            REFERENCES rh_account (id),
    CONSTRAINT reporter_id
        FOREIGN KEY (reporter_id)
            REFERENCES reporter (id)
) WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
  TABLESPACE pg_default;

CREATE INDEX ON system_platform (rh_account_id);
CREATE INDEX ON system_platform (last_upload DESC NULLS LAST);
CREATE INDEX ON system_platform (advisory_count_cache DESC NULLS LAST);
CREATE INDEX ON system_platform (advisory_enh_count_cache DESC NULLS LAST);
CREATE INDEX ON system_platform (advisory_bug_count_cache DESC NULLS LAST);
CREATE INDEX ON system_platform (advisory_sec_count_cache DESC NULLS LAST);



CREATE TRIGGER system_platform_set_first_reported
    BEFORE INSERT
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE set_first_reported();

CREATE TRIGGER system_platform_set_last_updated
    BEFORE INSERT OR UPDATE
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE set_last_updated();

CREATE TRIGGER system_platform_check_unchanged
    BEFORE INSERT OR UPDATE
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE check_unchanged();

CREATE TRIGGER system_platform_on_update
    AFTER UPDATE OF opt_out, stale
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE on_system_update();

GRANT SELECT, INSERT, UPDATE, DELETE ON system_platform TO listener;
-- evaluator needs to update last_evaluation
GRANT UPDATE ON system_platform TO evaluator;
-- manager needs to update cache and delete systems
GRANT UPDATE (advisory_count_cache,
              advisory_enh_count_cache,
              advisory_bug_count_cache,
              advisory_sec_count_cache), DELETE ON system_platform TO manager;

-- VMaaS sync needs to be able to perform system culling tasks
GRANT SELECT, UPDATE, DELETE ON system_platform to vmaas_sync;

CREATE TABLE IF NOT EXISTS deleted_system
(
    inventory_id TEXT                     NOT NULL,
    CHECK (NOT empty(inventory_id)),
    when_deleted TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE (inventory_id)
) TABLESPACE pg_default;

CREATE INDEX ON deleted_system (when_deleted);

GRANT SELECT, INSERT, UPDATE, DELETE ON deleted_system TO listener;
-- advisory_type
CREATE TABLE IF NOT EXISTS advisory_type
(
    id   INT  NOT NULL,
    name TEXT NOT NULL UNIQUE,
    CHECK (NOT empty(name)),
    PRIMARY KEY (id)
) TABLESPACE pg_default;

INSERT INTO advisory_type (id, name)
VALUES (0, 'unknown'),
       (1, 'enhancement'),
       (2, 'bugfix'),
       (3, 'security')
ON CONFLICT DO NOTHING;

CREATE TABLE advisory_severity
(
    id   INT  NOT NULL,
    name TEXT NOT NULL UNIQUE CHECK ( not empty(name) ),
    PRIMARY KEY (id)
);

INSERT INTO advisory_severity (id, name)
VALUES (1, 'Low'),
       (2, 'Moderate'),
       (3, 'Important'),
       (4, 'Critical')
ON CONFLICT DO NOTHING;

-- advisory_metadata
CREATE TABLE IF NOT EXISTS advisory_metadata
(
    id               INT GENERATED BY DEFAULT AS IDENTITY,
    name             TEXT                     NOT NULL,
    CHECK (NOT empty(name)),
    description      TEXT                     NOT NULL,
    CHECK (NOT empty(description)),
    synopsis         TEXT                     NOT NULL,
    CHECK (NOT empty(synopsis)),
    summary          TEXT                     NOT NULL,
    CHECK (NOT empty(summary)),
    solution         TEXT                     NOT NULL,
    CHECK (NOT empty(solution)),
    advisory_type_id INT                      NOT NULL,
    public_date      TIMESTAMP WITH TIME ZONE NULL,
    modified_date    TIMESTAMP WITH TIME ZONE NULL,
    url              TEXT,
    severity_id      INT,
    package_data     JSONB,
    cve_list         JSONB,
    UNIQUE (name),
    PRIMARY KEY (id),
    CONSTRAINT advisory_type_id
        FOREIGN KEY (advisory_type_id)
            REFERENCES advisory_type (id),
    CONSTRAINT advisory_severity_id
        FOREIGN KEY (severity_id)
            REFERENCES advisory_severity (id)
) TABLESPACE pg_default;

CREATE INDEX ON advisory_metadata (advisory_type_id);

CREATE INDEX IF NOT EXISTS
    advisory_metadata_pkgdata_idx ON advisory_metadata
    USING GIN ((advisory_metadata.package_data));

GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO vmaas_sync;
-- TODO: Remove
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO listener;
GRANT SELECT ON advisory_metadata TO manager;

-- status table
CREATE TABLE IF NOT EXISTS status
(
    id   INT  NOT NULL,
    name TEXT NOT NULL UNIQUE,
    CHECK (NOT empty(name)),
    PRIMARY KEY (id)
) TABLESPACE pg_default;

INSERT INTO status (id, name)
VALUES (0, 'Not Reviewed'),
       (1, 'In-Review'),
       (2, 'On-Hold'),
       (3, 'Scheduled for Patch'),
       (4, 'Resolved'),
       (5, 'No Action')
ON CONFLICT DO NOTHING;


-- system_advisories
CREATE TABLE IF NOT EXISTS system_advisories
(
    system_id      INT                      NOT NULL,
    advisory_id    INT                      NOT NULL,
    first_reported TIMESTAMP WITH TIME ZONE NOT NULL,
    when_patched   TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    status_id      INT                      DEFAULT 0,
    PRIMARY KEY (system_id, advisory_id),
    CONSTRAINT system_platform_id
        FOREIGN KEY (system_id)
            REFERENCES system_platform (id),
    CONSTRAINT advisory_metadata_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id),
    CONSTRAINT status_id
        FOREIGN KEY (status_id)
            REFERENCES status (id)
) WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
  TABLESPACE pg_default;

CREATE TRIGGER system_advisories_set_first_reported
    BEFORE INSERT
    ON system_advisories
    FOR EACH ROW
EXECUTE PROCEDURE set_first_reported();

GRANT SELECT, INSERT, UPDATE, DELETE ON system_advisories TO evaluator;
-- manager needs to be able to update things like 'status' on a sysid/advisory combination, also needs to delete
GRANT UPDATE, DELETE ON system_advisories TO manager;
-- manager needs to be able to update opt_out column
GRANT UPDATE (opt_out) ON system_platform TO manager;
-- listener deletes systems, TODO: temporary added evaluator permissions to listener
GRANT SELECT, INSERT, UPDATE, DELETE ON system_advisories TO listener;
-- vmaas_sync needs to delete culled systems, which cascades to system_advisories
GRANT SELECT, DELETE ON system_advisories TO vmaas_sync;

-- advisory_account_data
CREATE TABLE IF NOT EXISTS advisory_account_data
(
    advisory_id              INT NOT NULL,
    rh_account_id            INT NOT NULL,
    status_id                INT NOT NULL DEFAULT 0,
    systems_affected         INT NOT NULL DEFAULT 0,
    systems_status_divergent INT NOT NULL DEFAULT 0,
    CONSTRAINT advisory_metadata_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id),
    CONSTRAINT rh_account_id
        FOREIGN KEY (rh_account_id)
            REFERENCES rh_account (id),
    CONSTRAINT status_id
        FOREIGN KEY (status_id)
            REFERENCES status (id),
    UNIQUE (advisory_id, rh_account_id),
    PRIMARY KEY (rh_account_id, advisory_id)
) WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
  TABLESPACE pg_default;

-- manager user needs to change this table for opt-out functionality
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO manager;
-- evaluator user needs to change this table
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO evaluator;
-- listner user needs to change this table when deleting system
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO listener;
-- vmaas_sync needs to update stale mark, which creates and deletes advisory_account_data
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO vmaas_sync;

-- repo
CREATE TABLE IF NOT EXISTS repo
(
    id   INT GENERATED BY DEFAULT AS IDENTITY,
    name TEXT NOT NULL UNIQUE,
    CHECK (NOT empty(name)),
    PRIMARY KEY (id)
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON repo TO listener;


-- system_repo
CREATE TABLE IF NOT EXISTS system_repo
(
    system_id INT NOT NULL,
    repo_id   INT NOT NULL,
    UNIQUE (system_id, repo_id),
    CONSTRAINT system_platform_id
        FOREIGN KEY (system_id)
            REFERENCES system_platform (id),
    CONSTRAINT repo_id
        FOREIGN KEY (repo_id)
            REFERENCES repo (id)
) TABLESPACE pg_default;

CREATE INDEX ON system_repo (system_id);
CREATE INDEX ON system_repo (repo_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_repo TO listener;
GRANT DELETE ON system_repo TO manager;
GRANT SELECT ON system_repo TO evaluator;
GRANT SELECT, DELETE on system_repo to vmaas_sync;

CREATE TABLE IF NOT EXISTS package_name
(
    id   INT GENERATED BY DEFAULT AS IDENTITY NOT NULL PRIMARY KEY,
    name TEXT                                 NOT NULL CHECK (NOT empty(name)) UNIQUE
);

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE package_name TO vmaas_sync;

CREATE TABLE IF NOT EXISTS strings
(
    id    BYTEA NOT NULL PRIMARY KEY,
    value TEXT  NOT NULL
);

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE strings TO vmaas_sync;

CREATE TABLE IF NOT EXISTS package
(
    id               INT GENERATED BY DEFAULT AS IDENTITY NOT NULL PRIMARY KEY,
    name_id          INT                                  NOT NULL REFERENCES package_name,
    evra             TEXT                                 NOT NULL CHECK (NOT empty(evra)),
    description_hash BYTEA                                NOT NULL REFERENCES strings (id),
    summary_hash     BYTEA                                NOT NULL REFERENCES strings (id),
    advisory_id      INT REFERENCES advisory_metadata (id),
    UNIQUE (name_id, evra)
) WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
  TABLESPACE pg_default;

CREATE UNIQUE INDEX IF NOT EXISTS package_evra_idx on package (evra, name_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE package TO vmaas_sync;

-- Optimistic cache of latest released version for each package
-- Used for retrieving package descriptions on screens where we only show package(note nevra) level data
CREATE MATERIALIZED VIEW IF NOT EXISTS package_latest_cache
AS
SELECT DISTINCT ON (p.name_id) p.name_id, p.id as package_id, sum.value as summary
FROM package p
         INNER JOIN strings sum on p.summary_hash = sum.id
         LEFT JOIN advisory_metadata am on p.advisory_id = am.id
ORDER BY p.name_id, am.public_date;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE package_latest_cache TO vmaas_sync;

CREATE TABLE IF NOT EXISTS system_package
(
    rh_account_id INT NOT NULL REFERENCES rh_account,
    -- Missing foreign key fixes performance issues when deleting systems
    -- We manually ensure consistency by deleting entries in the `delete_system`
    -- stored function. We should check whether constraint referencing  + cascade fixes it
    system_id     INT NOT NULL,
    package_id    INT NOT NULL REFERENCES package,
    -- Use null to represent up-to-date packages
    update_data   JSONB DEFAULT NULL,
    PRIMARY KEY (rh_account_id, system_id, package_id)
) PARTITION BY HASH (rh_account_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_package TO evaluator;
GRANT SELECT, UPDATE, DELETE ON system_package TO listener;
GRANT SELECT, UPDATE, DELETE ON system_package TO manager;
GRANT SELECT, UPDATE, DELETE ON system_package TO vmaas_sync;

SELECT create_table_partitions('system_package', 16,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

-- timestamp_kv
CREATE TABLE IF NOT EXISTS timestamp_kv
(
    name  TEXT                     NOT NULL UNIQUE,
    CHECK (NOT empty(name)),
    value TIMESTAMP WITH TIME ZONE NOT NULL
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON timestamp_kv TO vmaas_sync;

-- vmaas_sync needs to delete from this tables to sync CVEs correctly
GRANT DELETE ON system_advisories TO vmaas_sync;
GRANT DELETE ON advisory_account_data TO vmaas_sync;

-- ----------------------------------------------------------------------------
-- Read access for all users
-- ----------------------------------------------------------------------------

-- user for evaluator component
GRANT SELECT ON ALL TABLES IN SCHEMA public TO evaluator;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO evaluator;

-- user for listener component
GRANT SELECT ON ALL TABLES IN SCHEMA public TO listener;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO listener;

-- user for UI manager component
GRANT SELECT ON ALL TABLES IN SCHEMA public TO manager;

-- user for VMaaS sync component
GRANT SELECT ON ALL TABLES IN SCHEMA public TO vmaas_sync;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vmaas_sync;


CREATE SCHEMA IF NOT EXISTS inventory;

-- The admin ROLE that allows the inventory schema to be managed
CREATE ROLE cyndi_admin;
GRANT ALL PRIVILEGES ON SCHEMA inventory TO cyndi_admin;

-- The reader ROLE that provides SELECT access to the inventory.hosts view
CREATE ROLE cyndi_reader;
GRANT USAGE ON SCHEMA inventory TO cyndi_reader;

-- The application user is granted the reader role only to eliminate any interference with Cyndi
GRANT cyndi_reader to listener;
GRANT cyndi_reader to evaluator;
GRANT cyndi_reader to manager;
GRANT cyndi_reader TO vmaas_sync;

GRANT cyndi_admin to cyndi;
