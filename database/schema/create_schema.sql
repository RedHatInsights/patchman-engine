CREATE TABLE IF NOT EXISTS db_version
(
    version bigint  NOT NULL,
    dirty   boolean NOT NULL,
    PRIMARY KEY (version)
) TABLESPACE pg_default;

-- set the schema version directly in the insert statement here!!
--INSERT INTO db_version (name, version)
--VALUES ('schema_version', 1);
-- INSERT INTO db_version (name, version) VALUES ('schema_version', :schema_version);


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
$empty$
    LANGUAGE 'plpgsql';

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

-- opt_out_system_update_cache
CREATE OR REPLACE FUNCTION opt_out_system_update_cache()
    RETURNS TRIGGER AS
$opt_out_system_update_cache$
BEGIN
    IF (TG_OP = 'UPDATE') AND NEW.last_evaluation IS NOT NULL THEN
        -- system opted out
        IF OLD.opt_out = FALSE AND NEW.opt_out = TRUE THEN
            -- decrement affected advisory counts for system
            WITH to_update_advisories AS (
                SELECT ead.advisory_id, ead.status_id AS global_status_id, sa.status_id
                FROM advisory_account_data ead
                         INNER JOIN
                     system_advisories sa ON ead.advisory_id = sa.advisory_id
                WHERE ead.rh_account_id = NEW.rh_account_id
                  AND sa.system_id = NEW.id
                  AND sa.when_patched IS NULL
                ORDER BY ead.advisory_id
                    FOR UPDATE OF ead
                -- decrement systems_affected and systems_status_divergent in case status is different
            ),
                 update_divergent AS (
                     UPDATE advisory_account_data ead
                         SET systems_affected = systems_affected - 1,
                             systems_status_divergent = systems_status_divergent - 1
                         FROM to_update_advisories
                         WHERE ead.advisory_id = to_update_advisories.advisory_id AND
                               ead.rh_account_id = NEW.rh_account_id AND
                               to_update_advisories.global_status_id != to_update_advisories.status_id
                 )
                 -- decrement only systems_affected in case status is same
            UPDATE advisory_account_data ead
            SET systems_affected = systems_affected - 1
            FROM to_update_advisories
            WHERE ead.advisory_id = to_update_advisories.advisory_id
              AND ead.rh_account_id = NEW.rh_account_id
              AND to_update_advisories.global_status_id = to_update_advisories.status_id;
            -- delete zero advisory counts
            DELETE
            FROM advisory_account_data
            WHERE rh_account_id = NEW.rh_account_id
              AND systems_affected = 0;

            -- system opted in
        ELSIF OLD.opt_out = TRUE AND NEW.opt_out = FALSE THEN
            -- increment affected advisory counts for system
            WITH to_update_advisories AS (
                SELECT ead.advisory_id, ead.status_id AS global_status_id, sa.status_id
                FROM advisory_account_data ead
                         INNER JOIN
                     system_advisories sa ON ead.advisory_id = sa.advisory_id
                WHERE ead.rh_account_id = NEW.rh_account_id
                  AND sa.system_id = NEW.id
                  AND sa.when_patched IS NULL
                ORDER BY ead.advisory_id
                    FOR UPDATE OF ead
                -- increment systems_affected and systems_status_divergent in case status is different
            ),
                 update_divergent AS (
                     UPDATE advisory_account_data ead
                         SET systems_affected = systems_affected + 1,
                             systems_status_divergent = systems_status_divergent + 1
                         FROM to_update_advisories
                         WHERE ead.advisory_id = to_update_advisories.advisory_id AND
                               ead.rh_account_id = NEW.rh_account_id AND
                               to_update_advisories.global_status_id != to_update_advisories.status_id
                 )
                 -- increment only systems_affected in case status is same
            UPDATE advisory_account_data ead
            SET systems_affected = systems_affected + 1
            FROM to_update_advisories
            WHERE ead.advisory_id = to_update_advisories.advisory_id
              AND ead.rh_account_id = NEW.rh_account_id
              AND to_update_advisories.global_status_id = to_update_advisories.status_id;
            -- insert cache if not exists
            INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
            SELECT sa.advisory_id, NEW.rh_account_id, 1
            FROM system_advisories sa
            WHERE sa.system_id = NEW.id
              AND sa.when_patched IS NULL
              AND NOT EXISTS(
                    SELECT 1
                    FROM advisory_account_data
                    WHERE rh_account_id = NEW.rh_account_id
                      AND advisory_id = sa.advisory_id
                )
            ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET systems_affected = advisory_account_data.systems_affected + EXCLUDED.systems_affected;
        END IF;
    END IF;
    RETURN NEW;
END;
$opt_out_system_update_cache$
    LANGUAGE 'plpgsql';

-- update system advisories counts (all and according types)
CREATE OR REPLACE FUNCTION update_system_caches(system_id_in INT)
    RETURNS VOID AS
$update_system_caches$
BEGIN
    WITH to_update_systems AS (
        SELECT sp.id
        FROM system_platform sp
        WHERE sp.id = system_id_in
        ORDER BY sp.rh_account_id, sp.id
            FOR UPDATE OF sp
    )
    UPDATE system_platform sp
    SET advisory_count_cache     = (
        SELECT COUNT(advisory_id)
        FROM system_advisories sa
        WHERE sa.system_id = sp.id
          AND sa.when_patched IS NULL
    ),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    FROM to_update_systems;
END;
$update_system_caches$
    LANGUAGE 'plpgsql';

-- count system advisories according to advisory type
CREATE OR REPLACE FUNCTION system_advisories_count(system_id_in INT, advisory_type_id_in INT)
    RETURNS INT AS
$system_advisories_count$
DECLARE
    result_cnt INT;
BEGIN
    SELECT COUNT(advisory_id)
    FROM system_advisories sa
             JOIN advisory_metadata am ON sa.advisory_id = am.id
    WHERE am.advisory_type_id = advisory_type_id_in
      AND sa.system_id = system_id_in
      AND sa.when_patched IS NULL
    INTO result_cnt;
    RETURN result_cnt;
END;
$system_advisories_count$
    LANGUAGE 'plpgsql';

-- refresh_all_cached_counts
-- WARNING: executing this procedure takes long time,
--          use only when necessary, e.g. during upgrade to populate initial caches
CREATE OR REPLACE FUNCTION refresh_all_cached_counts()
    RETURNS void AS
$refresh_all_cached_counts$
BEGIN
    -- update advisories count for ordered systems
    WITH to_update_systems AS (
        SELECT sp.id
        FROM system_platform sp
        ORDER BY sp.rh_account_id, sp.id
            FOR UPDATE OF sp
    )
    UPDATE system_platform sp
    SET advisory_count_cache     = (
        SELECT COUNT(advisory_id)
        FROM system_advisories sa
        WHERE sa.system_id = sp.id
          AND sa.when_patched IS NULL
    ),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    FROM to_update_systems
    WHERE sp.id = to_update_systems.id;

    -- update system count for ordered advisory
    WITH locked_rows AS (
        SELECT ead.rh_account_id, ead.advisory_id
        FROM advisory_account_data ead
        ORDER BY ead.rh_account_id, ead.advisory_id
            FOR UPDATE OF ead
    ),
         current_counts AS (
             SELECT sa.advisory_id, sp.rh_account_id, count(sa.system_id) as systems_affected
             FROM system_advisories sa
                      INNER JOIN
                  system_platform sp ON sa.system_id = sp.id
             WHERE sp.last_evaluation IS NOT NULL
               AND sp.opt_out = FALSE
               AND sa.when_patched IS NULL
             GROUP BY sa.advisory_id, sp.rh_account_id
         ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_id, rh_account_id, systems_affected FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE (advisory_id, rh_account_id) NOT IN (SELECT advisory_id, rh_account_id FROM current_counts);
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
    WITH to_update_systems AS (
        SELECT sp.id
        FROM system_platform sp
        WHERE sp.rh_account_id = rh_account_id_in
        ORDER BY sp.id
            FOR UPDATE OF sp
    )
    UPDATE system_platform sp
    SET advisory_count_cache     = (
        SELECT COUNT(advisory_id)
        FROM system_advisories sa
        WHERE sa.system_id = sp.id
          AND sa.when_patched IS NULL
    ),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    FROM to_update_systems
    WHERE sp.id = to_update_systems.id;

    -- update system count for ordered advisory
    WITH locked_rows AS (
        SELECT ead.advisory_id
        FROM advisory_account_data ead
        WHERE ead.rh_account_id = rh_account_id_in
        ORDER BY ead.advisory_id
            FOR UPDATE OF ead
    ),
         current_counts AS (
             SELECT sa.advisory_id, count(sa.system_id) as systems_affected
             FROM system_advisories sa
                      INNER JOIN
                  system_platform sp ON sa.system_id = sp.id
             WHERE sp.last_evaluation IS NOT NULL
               AND sp.opt_out = FALSE
               AND sa.when_patched IS NULL
               AND sp.rh_account_id = rh_account_id_in
             GROUP BY sa.advisory_id
         ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_id, rh_account_id_in, systems_affected FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE advisory_id NOT IN (SELECT advisory_id FROM current_counts)
      AND rh_account_id = rh_account_id_in;
END;
$refresh_account_cached_counts$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_advisory_cached_counts(advisory_name varchar)
    RETURNS void AS
$refresh_advisory_cached_counts$
DECLARE
    advisory_md_id INT;
BEGIN
    -- update system count for advisory
    SELECT id FROM advisory_metadata WHERE name = advisory_name INTO advisory_md_id;
    WITH locked_rows AS (
        SELECT ead.rh_account_id
        FROM advisory_account_data ead
        WHERE ead.advisory_id = advisory_md_id
        ORDER BY ead.rh_account_id
            FOR UPDATE OF ead
    ),
         current_counts AS (
             SELECT sp.rh_account_id, count(sa.system_id) as systems_affected
             FROM system_advisories sa
                      INNER JOIN
                  system_platform sp ON sa.system_id = sp.id
             WHERE sp.last_evaluation IS NOT NULL
               AND sp.opt_out = FALSE
               AND sa.when_patched IS NULL
               AND sa.advisory_id = advisory_md_id
             GROUP BY sp.rh_account_id
         ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_md_id, rh_account_id, systems_affected FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE rh_account_id NOT IN (SELECT rh_account_id FROM current_counts)
      AND advisory_id = advisory_md_id;
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
    WITH locked_rows AS (
        SELECT ead.rh_account_id, ead.advisory_id
        FROM advisory_account_data ead
        WHERE ead.advisory_id = advisory_md_id
          AND ead.rh_account_id = rh_account_id_in
            FOR UPDATE OF ead
    ),
         current_counts AS (
             SELECT sa.advisory_id, sp.rh_account_id, count(sa.system_id) as systems_affected
             FROM system_advisories sa
                      INNER JOIN
                  system_platform sp ON sa.system_id = sp.id
             WHERE sp.last_evaluation IS NOT NULL
               AND sp.opt_out = FALSE
               AND sa.when_patched IS NULL
               AND sa.advisory_id = advisory_md_id
               AND sp.rh_account_id = rh_account_id_in
             GROUP BY sa.advisory_id, sp.rh_account_id
         ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_md_id, rh_account_id_in, systems_affected FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE NOT EXISTS(SELECT 1 FROM current_counts)
      AND advisory_id = advisory_md_id
      AND rh_account_id = rh_account_id_in;
END;
$refresh_advisory_account_cached_counts$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_system_cached_counts(inventory_id_in varchar)
    RETURNS void AS
$refresh_system_cached_counts$
BEGIN
    -- update advisory count for system
    UPDATE system_platform sp
    SET advisory_count_cache     = (
        SELECT COUNT(advisory_id)
        FROM system_advisories sa
        WHERE sa.system_id = sp.id
          AND sa.when_patched IS NULL
    ),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    WHERE sp.inventory_id = inventory_id_in;
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
BEGIN
    -- opt out to refresh cache and then delete
    WITH locked_row AS (
        SELECT id
        FROM system_platform
        WHERE inventory_id = inventory_id_in
            FOR UPDATE
    )
    UPDATE system_platform
    SET opt_out = true
    WHERE inventory_id = inventory_id_in;
    DELETE
    FROM system_advisories
    WHERE system_id = (SELECT id from system_platform WHERE inventory_id = inventory_id_in);
    DELETE
    FROM system_repo
    WHERE system_id = (SELECT id from system_platform WHERE inventory_id = inventory_id_in);
    RETURN QUERY DELETE FROM system_platform
        WHERE inventory_id = inventory_id_in
        RETURNING inventory_id;
END;
$delete_system$
    LANGUAGE 'plpgsql';

-- rh_account
CREATE TABLE IF NOT EXISTS rh_account
(
    id   INT GENERATED BY DEFAULT AS IDENTITY,
    name TEXT NOT NULL UNIQUE,
    CHECK (NOT empty(name)),
    PRIMARY KEY (id)
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON rh_account TO listener;

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
    PRIMARY KEY (id),
    last_upload              TIMESTAMP WITH TIME ZONE,
    UNIQUE (inventory_id),
    CONSTRAINT rh_account_id
        FOREIGN KEY (rh_account_id)
            REFERENCES rh_account (id)
) TABLESPACE pg_default;

CREATE INDEX ON system_platform (rh_account_id);

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

CREATE TRIGGER system_platform_opt_out_cache
    AFTER UPDATE OF opt_out
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE opt_out_system_update_cache();

GRANT SELECT, INSERT, UPDATE, DELETE ON system_platform TO listener;
-- evaluator needs to update last_evaluation
GRANT UPDATE ON system_platform TO evaluator;
-- manager needs to update cache and delete systems
GRANT UPDATE (advisory_count_cache,
              advisory_enh_count_cache,
              advisory_bug_count_cache,
              advisory_sec_count_cache), DELETE ON system_platform TO manager;

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
       (3, 'security');


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
    UNIQUE (name),
    PRIMARY KEY (id),
    CONSTRAINT advisory_type_id
        FOREIGN KEY (advisory_type_id)
            REFERENCES advisory_type (id)
) TABLESPACE pg_default;

CREATE INDEX ON advisory_metadata (advisory_type_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO vmaas_sync;


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
       (5, 'No Action');


-- system_advisories
CREATE TABLE IF NOT EXISTS system_advisories
(
    id             INT GENERATED BY DEFAULT AS IDENTITY,
    system_id      INT                      NOT NULL,
    advisory_id    INT                      NOT NULL,
    first_reported TIMESTAMP WITH TIME ZONE NOT NULL,
    when_patched   TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    status_id      INT                      DEFAULT 0,
    UNIQUE (system_id, advisory_id),
    PRIMARY KEY (id),
    CONSTRAINT system_platform_id
        FOREIGN KEY (system_id)
            REFERENCES system_platform (id),
    CONSTRAINT advisory_metadata_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id),
    CONSTRAINT status_id
        FOREIGN KEY (status_id)
            REFERENCES status (id)
) TABLESPACE pg_default;

CREATE INDEX ON system_advisories (status_id);

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
GRANT DELETE, INSERT, UPDATE, DELETE ON system_advisories TO listener;

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
    UNIQUE (advisory_id, rh_account_id)
) TABLESPACE pg_default;

-- manager needs to write into advisory_account_data table
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO manager;

-- manager user needs to change this table for opt-out functionality
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO manager;
-- evaluator user needs to change this table
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO evaluator;
-- listner user needs to change this table when deleting system
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO listener;

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
