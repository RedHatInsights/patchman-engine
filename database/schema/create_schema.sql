CREATE TABLE IF NOT EXISTS db_version (
  name TEXT NOT NULL,
  version INT NOT NULL,
  PRIMARY KEY (name)
) TABLESPACE pg_default;

-- set the schema version directly in the insert statement here!!
INSERT INTO db_version (name, version) VALUES ('schema_version', 1);
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
        -- decrement affected errata counts for system
        WITH to_update_advisories AS (
          SELECT ead.errata_id, ead.status_id AS global_status_id, sa.status_id
          FROM errata_account_data ead INNER JOIN
               system_advisories sa ON ead.errata_id = sa.errata_id
          WHERE ead.rh_account_id = NEW.rh_account_id AND
                sa.system_id = NEW.id AND
                sa.when_patched IS NULL
          ORDER BY ead.errata_id
          FOR UPDATE OF ead
        -- decrement systems_affected and systems_status_divergent in case status is different
        ), update_divergent AS (
          UPDATE errata_account_data ead
          SET systems_affected = systems_affected - 1,
              systems_status_divergent = systems_status_divergent - 1
          FROM to_update_advisories
          WHERE ead.errata_id = to_update_advisories.errata_id AND
                ead.rh_account_id = NEW.rh_account_id AND
                to_update_advisories.global_status_id != to_update_advisories.status_id
        )
        -- decrement only systems_affected in case status is same
        UPDATE errata_account_data ead
        SET systems_affected = systems_affected - 1
        FROM to_update_advisories
        WHERE ead.errata_id = to_update_advisories.errata_id AND
              ead.rh_account_id = NEW.rh_account_id AND
              to_update_advisories.global_status_id = to_update_advisories.status_id;
        -- delete zero errata counts
        DELETE FROM errata_account_data
        WHERE rh_account_id = NEW.rh_account_id AND
              systems_affected = 0;

      -- system opted in
      ELSIF OLD.opt_out = TRUE AND NEW.opt_out = FALSE THEN
        -- increment affected errata counts for system
        WITH to_update_advisories AS (
          SELECT ead.errata_id, ead.status_id AS global_status_id, sa.status_id
          FROM errata_account_data ead INNER JOIN
               system_advisories sa ON ead.errata_id = sa.errata_id
          WHERE ead.rh_account_id = NEW.rh_account_id AND
                sa.system_id = NEW.id AND
                sa.when_patched IS NULL
          ORDER BY ead.errata_id
          FOR UPDATE OF ead
        -- increment systems_affected and systems_status_divergent in case status is different
        ), update_divergent AS (
          UPDATE errata_account_data ead
          SET systems_affected = systems_affected + 1,
              systems_status_divergent = systems_status_divergent + 1
          FROM to_update_advisories
          WHERE ead.errata_id = to_update_advisories.errata_id AND
                ead.rh_account_id = NEW.rh_account_id AND
                to_update_advisories.global_status_id != to_update_advisories.status_id
        )
        -- increment only systems_affected in case status is same
        UPDATE errata_account_data ead
        SET systems_affected = systems_affected + 1
        FROM to_update_advisories
        WHERE ead.errata_id = to_update_advisories.errata_id AND
              ead.rh_account_id = NEW.rh_account_id AND
              to_update_advisories.global_status_id = to_update_advisories.status_id;
        -- insert cache if not exists
        INSERT INTO errata_account_data (errata_id, rh_account_id, systems_affected)
        SELECT sa.errata_id, NEW.rh_account_id, 1
        FROM system_advisories sa
        WHERE sa.system_id = NEW.id AND
              sa.when_patched IS NULL AND
              NOT EXISTS (
                SELECT 1 FROM errata_account_data
                WHERE rh_account_id = NEW.rh_account_id AND
                      errata_id = sa.errata_id
              )
        ON CONFLICT (errata_id, rh_account_id) DO UPDATE SET
          systems_affected = errata_account_data.systems_affected + EXCLUDED.systems_affected;
      END IF;
    END IF;
    RETURN NEW;
  END;
$opt_out_system_update_cache$
  LANGUAGE 'plpgsql';

-- refresh_all_cached_counts
-- WARNING: executing this procedure takes long time,
--          use only when necessary, e.g. during upgrade to populate initial caches
CREATE OR REPLACE FUNCTION refresh_all_cached_counts()
  RETURNS void AS
$refresh_all_cached_counts$
  BEGIN
    -- update errata count for ordered systems
    WITH to_update_systems AS (
      SELECT sp.id
      FROM system_platform sp
      ORDER BY sp.rh_account_id, sp.id
      FOR UPDATE OF sp
    )
    UPDATE system_platform sp SET errata_count_cache = (
      SELECT COUNT(errata_id) FROM system_advisories sa
      WHERE sa.system_id = sp.id AND sa.when_patched IS NULL
    )
    FROM to_update_systems
    WHERE sp.id = to_update_systems.id;

    -- update system count for ordered errata
    WITH locked_rows AS (
      SELECT ead.rh_account_id, ead.errata_id
      FROM errata_account_data ead
      ORDER BY ead.rh_account_id, ead.errata_id
      FOR UPDATE OF ead
    ), current_counts AS (
      SELECT sa.errata_id, sp.rh_account_id, count(sa.system_id) as systems_affected
      FROM system_advisories sa INNER JOIN
           system_platform sp ON sa.system_id = sp.id
      WHERE sp.last_evaluation IS NOT NULL AND
            sp.opt_out = FALSE AND
            sa.when_patched IS NULL
      GROUP BY sa.errata_id, sp.rh_account_id
    ), upserted AS (
      INSERT INTO errata_account_data (errata_id, rh_account_id, systems_affected)
        SELECT errata_id, rh_account_id, systems_affected FROM current_counts
      ON CONFLICT (errata_id, rh_account_id) DO UPDATE SET
        systems_affected = EXCLUDED.systems_affected
    )
    DELETE FROM errata_account_data WHERE (errata_id, rh_account_id) NOT IN (SELECT errata_id, rh_account_id FROM current_counts);
  END;
$refresh_all_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_account_cached_counts(rh_account_in varchar)
  RETURNS void AS
$refresh_account_cached_counts$
  DECLARE
    rh_account_id_in INT;
  BEGIN
    -- update errata count for ordered systems
    SELECT id FROM rh_account WHERE name = rh_account_in INTO rh_account_id_in;
    WITH to_update_systems AS (
      SELECT sp.id
      FROM system_platform sp
      WHERE sp.rh_account_id = rh_account_id_in
      ORDER BY sp.id
      FOR UPDATE OF sp
    )
    UPDATE system_platform sp SET errata_count_cache = (
      SELECT COUNT(errata_id) FROM system_advisories sa
      WHERE sa.system_id = sp.id AND sa.when_patched IS NULL
    )
    FROM to_update_systems
    WHERE sp.id = to_update_systems.id;

    -- update system count for ordered errata
    WITH locked_rows AS (
      SELECT ead.errata_id
      FROM errata_account_data ead
      WHERE ead.rh_account_id = rh_account_id_in
      ORDER BY ead.errata_id
      FOR UPDATE OF ead
    ), current_counts AS (
      SELECT sa.errata_id, count(sa.system_id) as systems_affected
      FROM system_advisories sa INNER JOIN
           system_platform sp ON sa.system_id = sp.id
      WHERE sp.last_evaluation IS NOT NULL AND
            sp.opt_out = FALSE AND
            sa.when_patched IS NULL AND
            sp.rh_account_id = rh_account_id_in
      GROUP BY sa.errata_id
    ), upserted AS (
      INSERT INTO errata_account_data (errata_id, rh_account_id, systems_affected)
        SELECT errata_id, rh_account_id_in, systems_affected FROM current_counts
      ON CONFLICT (errata_id, rh_account_id) DO UPDATE SET
        systems_affected = EXCLUDED.systems_affected
    )
    DELETE FROM errata_account_data WHERE errata_id NOT IN (SELECT errata_id FROM current_counts)
      AND rh_account_id = rh_account_id_in;
  END;
$refresh_account_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_errata_cached_counts(advisory_in varchar)
  RETURNS void AS
$refresh_errata_cached_counts$
  DECLARE
    errata_md_id INT;
  BEGIN
    -- update system count for errata
    SELECT id FROM errata_metadata WHERE advisory = advisory_in INTO errata_md_id;
    WITH locked_rows AS (
      SELECT ead.rh_account_id
      FROM errata_account_data ead
      WHERE ead.errata_id = errata_md_id
      ORDER BY ead.rh_account_id
      FOR UPDATE OF ead
    ), current_counts AS (
      SELECT sp.rh_account_id, count(sa.system_id) as systems_affected
      FROM system_advisories sa INNER JOIN
           system_platform sp ON sa.system_id = sp.id
      WHERE sp.last_evaluation IS NOT NULL AND
            sp.opt_out = FALSE AND
            sa.when_patched IS NULL AND
            sa.errata_id = errata_md_id
      GROUP BY sp.rh_account_id
    ), upserted AS (
      INSERT INTO errata_account_data (errata_id, rh_account_id, systems_affected)
        SELECT errata_md_id, rh_account_id, systems_affected FROM current_counts
      ON CONFLICT (errata_id, rh_account_id) DO UPDATE SET
        systems_affected = EXCLUDED.systems_affected
    )
    DELETE FROM errata_account_data WHERE rh_account_id NOT IN (SELECT rh_account_id FROM current_counts)
      AND errata_id = errata_md_id;
  END;
$refresh_errata_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_errata_account_cached_counts(advisory_in varchar, rh_account_in varchar)
  RETURNS void AS
$refresh_errata_account_cached_counts$
  DECLARE
    errata_md_id INT;
    rh_account_id_in INT;
  BEGIN
    -- update system count for ordered errata
    SELECT id FROM errata_metadata WHERE advisory = advisory_in INTO errata_md_id;
    SELECT id FROM rh_account WHERE name = rh_account_in INTO rh_account_id_in;
    WITH locked_rows AS (
      SELECT ead.rh_account_id, ead.errata_id
      FROM errata_account_data ead
      WHERE ead.errata_id = errata_md_id AND
            ead.rh_account_id = rh_account_id_in
      FOR UPDATE OF ead
    ), current_counts AS (
      SELECT sa.errata_id, sp.rh_account_id, count(sa.system_id) as systems_affected
      FROM system_advisories sa INNER JOIN
           system_platform sp ON sa.system_id = sp.id
      WHERE sp.last_evaluation IS NOT NULL AND
            sp.opt_out = FALSE AND
            sa.when_patched IS NULL AND
            sa.errata_id = errata_md_id AND
            sp.rh_account_id = rh_account_id_in
      GROUP BY sa.errata_id, sp.rh_account_id
    ), upserted AS (
      INSERT INTO errata_account_data (errata_id, rh_account_id, systems_affected)
        SELECT errata_md_id, rh_account_id_in, systems_affected FROM current_counts
      ON CONFLICT (errata_id, rh_account_id) DO UPDATE SET
        systems_affected = EXCLUDED.systems_affected
    )
    DELETE FROM errata_account_data WHERE NOT EXISTS (SELECT 1 FROM current_counts)
      AND errata_id = errata_md_id
      AND rh_account_id = rh_account_id_in;
  END;
$refresh_errata_account_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_system_cached_counts(inventory_id_in varchar)
  RETURNS void AS
$refresh_system_cached_counts$
  BEGIN
    -- update errata count for system
    UPDATE system_platform sp SET errata_count_cache = (
      SELECT COUNT(errata_id) FROM system_advisories sa
      WHERE sa.system_id = sp.id AND sa.when_patched IS NULL
    ) WHERE sp.inventory_id = inventory_id_in;
  END;
$refresh_system_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION delete_system(inventory_id_in varchar)
  RETURNS TABLE (deleted_inventory_id TEXT) AS
$delete_system$
  BEGIN
    -- opt out to refresh cache and then delete
    WITH locked_row AS (
      SELECT id
      FROM system_platform
      WHERE inventory_id = inventory_id_in
      FOR UPDATE
    )
    UPDATE system_platform SET opt_out = true
    WHERE inventory_id = inventory_id_in;
    DELETE FROM system_advisories
    WHERE system_id = (SELECT id from system_platform WHERE inventory_id = inventory_id_in);
    DELETE FROM system_repo
    WHERE system_id = (SELECT id from system_platform WHERE inventory_id = inventory_id_in);
    RETURN QUERY DELETE FROM system_platform
    WHERE inventory_id = inventory_id_in
    RETURNING inventory_id;
  END;
$delete_system$
  LANGUAGE 'plpgsql';


-- ----------------------------------------------------------------------------
-- Tables
-- ----------------------------------------------------------------------------

-- db_upgrade_log
CREATE TABLE IF NOT EXISTS db_upgrade_log (
  id SERIAL,
  version INT NOT NULL,
  status TEXT NOT NULL,
  script TEXT,
  returncode INT,
  stdout TEXT,
  stderr TEXT,
  last_updated TIMESTAMP WITH TIME ZONE NOT NULL
) TABLESPACE pg_default;

CREATE INDEX ON db_upgrade_log(version);

CREATE TRIGGER db_upgrade_log_set_last_updated
  BEFORE INSERT OR UPDATE ON db_upgrade_log
  FOR EACH ROW EXECUTE PROCEDURE set_last_updated();

-- rh_account
CREATE TABLE IF NOT EXISTS rh_account (
  id SERIAL,
  name TEXT NOT NULL UNIQUE, CHECK (NOT empty(name)),
  PRIMARY KEY (id)
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON rh_account TO listener;
-- manager needs to delete systems
GRANT DELETE ON rh_account TO manager;

-- system_platform
CREATE TABLE IF NOT EXISTS system_platform (
  id SERIAL,
  inventory_id TEXT NOT NULL, CHECK (NOT empty(inventory_id)),
  rh_account_id INT NOT NULL,
  first_reported TIMESTAMP WITH TIME ZONE NOT NULL,
  s3_url TEXT,
  vmaas_json TEXT,
  json_checksum TEXT,
  last_updated TIMESTAMP WITH TIME ZONE NOT NULL,
  unchanged_since TIMESTAMP WITH TIME ZONE NOT NULL,
  last_evaluation TIMESTAMP WITH TIME ZONE,
  opt_out BOOLEAN NOT NULL DEFAULT FALSE,
  errata_count_cache INT NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  last_upload TIMESTAMP WITH TIME ZONE,
  UNIQUE (inventory_id),
  CONSTRAINT rh_account_id
    FOREIGN KEY (rh_account_id)
    REFERENCES rh_account (id)
) TABLESPACE pg_default;

CREATE INDEX ON system_platform(rh_account_id);

CREATE TRIGGER system_platform_set_first_reported
  BEFORE INSERT ON system_platform
  FOR EACH ROW EXECUTE PROCEDURE set_first_reported();

CREATE TRIGGER system_platform_set_last_updated
  BEFORE INSERT OR UPDATE ON system_platform
  FOR EACH ROW EXECUTE PROCEDURE set_last_updated();

CREATE TRIGGER system_platform_check_unchanged
  BEFORE INSERT OR UPDATE ON system_platform
  FOR EACH ROW EXECUTE PROCEDURE check_unchanged();

CREATE TRIGGER system_platform_opt_out_cache
  AFTER UPDATE OF opt_out ON system_platform
  FOR EACH ROW EXECUTE PROCEDURE opt_out_system_update_cache();

GRANT SELECT, INSERT, UPDATE, DELETE ON system_platform TO listener;
-- evaluator needs to update last_evaluation
GRANT UPDATE ON system_platform TO evaluator;
-- manager needs to update cache and delete systems
GRANT UPDATE (errata_count_cache), DELETE ON system_platform TO manager;

-- errata_type
CREATE TABLE IF NOT EXISTS errata_type (
  id INT NOT NULL,
  name TEXT NOT NULL UNIQUE, CHECK (NOT empty(name)),
  PRIMARY KEY (id)
)TABLESPACE pg_default;

INSERT INTO errata_type (id, name) VALUES
  (0, 'NotSet'),
  (1, 'Product Enhancement Advisory'),
  (2, 'Bug Fix Advisory'),
  (3, 'Security Advisory');


-- errata_metadata
CREATE TABLE IF NOT EXISTS errata_metadata (
  id SERIAL,
  advisory TEXT NOT NULL, CHECK (NOT empty(advisory)),
  advisory_name TEXT NOT NULL, CHECK (NOT empty(advisory_name)),
  description TEXT NOT NULL, CHECK (NOT empty(description)),
  synopsis TEXT NOT NULL, CHECK (NOT empty(synopsis)),
  topic TEXT NOT NULL, CHECK (NOT empty(topic)),
  solution TEXT NOT NULL, CHECK (NOT empty(solution)),
  errata_type_id INT NOT NULL,
  public_date TIMESTAMP WITH TIME ZONE NULL,
  modified_date TIMESTAMP WITH TIME ZONE NULL,
  url TEXT,
  UNIQUE (advisory),
  PRIMARY KEY (id),
  CONSTRAINT errata_type_id
    FOREIGN KEY (errata_type_id)
    REFERENCES errata_type (id)
) TABLESPACE pg_default;

CREATE INDEX ON errata_metadata(errata_type_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON errata_metadata TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON errata_metadata TO vmaas_sync;


-- status table
CREATE TABLE IF NOT EXISTS status (
  id INT NOT NULL,
  name TEXT NOT NULL UNIQUE, CHECK (NOT empty(name)),
  PRIMARY KEY (id)
)TABLESPACE pg_default;

INSERT INTO status (id, name) VALUES
  (0, 'Not Reviewed'), (1, 'In-Review'), (2, 'On-Hold'), (3, 'Scheduled for Patch'), (4, 'Resolved'),
  (5, 'No Action');


-- system_advisories
CREATE TABLE IF NOT EXISTS system_advisories (
  id SERIAL,
  system_id INT NOT NULL,
  errata_id INT NOT NULL,
  first_reported TIMESTAMP WITH TIME ZONE NOT NULL,
  when_patched TIMESTAMP WITH TIME ZONE DEFAULT NULL,
  status_id INT DEFAULT 0,
  status_text TEXT,
  UNIQUE (system_id, errata_id),
  PRIMARY KEY (id),
  CONSTRAINT system_platform_id
    FOREIGN KEY (system_id)
    REFERENCES system_platform (id),
  CONSTRAINT errata_metadata_errata_id
    FOREIGN KEY (errata_id)
    REFERENCES errata_metadata (id),
  CONSTRAINT status_id
    FOREIGN KEY (status_id)
    REFERENCES status (id)
) TABLESPACE pg_default;

CREATE INDEX ON system_advisories(status_id);

CREATE TRIGGER system_advisories_set_first_reported BEFORE INSERT ON system_advisories
  FOR EACH ROW EXECUTE PROCEDURE set_first_reported();

GRANT SELECT, INSERT, UPDATE, DELETE ON system_advisories TO evaluator;
-- manager needs to be able to update things like 'status' on a sysid/errata combination, also needs to delete
GRANT UPDATE, DELETE ON system_advisories TO manager;
-- manager needs to be able to update opt_out column
GRANT UPDATE (opt_out) ON system_platform TO manager;
-- listener deletes systems
GRANT DELETE ON system_advisories TO listener;

-- business_risk table
CREATE TABLE IF NOT EXISTS business_risk (
  id INT NOT NULL,
  name VARCHAR NOT NULL UNIQUE,
  CHECK (NOT empty(name)),
  PRIMARY KEY (id)
) TABLESPACE pg_default;

INSERT INTO business_risk (id, name) VALUES
  (0, 'Not Defined'), (1, 'Low'), (2, 'Medium'), (3, 'High');

-- errata_account_data
CREATE TABLE IF NOT EXISTS errata_account_data (
  errata_id INT NOT NULL,
  rh_account_id INT NOT NULL,
  business_risk_id INT NOT NULL DEFAULT 0,
  business_risk_text TEXT,
  status_id INT NOT NULL DEFAULT 0,
  status_text TEXT,
  systems_affected INT NOT NULL DEFAULT 0,
  systems_status_divergent INT NOT NULL DEFAULT 0,
  CONSTRAINT errata_id
    FOREIGN KEY (errata_id)
    REFERENCES errata_metadata (id),
  CONSTRAINT rh_account_id
    FOREIGN KEY (rh_account_id)
    REFERENCES rh_account (id),
  CONSTRAINT business_risk_id
    FOREIGN KEY (business_risk_id)
    REFERENCES business_risk (id),
  CONSTRAINT status_id
    FOREIGN KEY (status_id)
    REFERENCES status (id),
  UNIQUE (errata_id, rh_account_id)
) TABLESPACE pg_default;

-- manager needs to write into errata_account_data table
GRANT SELECT, INSERT, UPDATE, DELETE ON errata_account_data TO manager;

-- manager user needs to change this table for opt-out functionality
GRANT SELECT, INSERT, UPDATE, DELETE ON errata_account_data TO manager;
-- evaluator user needs to change this table
GRANT SELECT, INSERT, UPDATE, DELETE ON errata_account_data TO evaluator;
-- listner user needs to change this table when deleting system
GRANT SELECT, INSERT, UPDATE, DELETE ON errata_account_data TO listener;


CREATE TABLE IF NOT EXISTS deleted_systems (
  inventory_id TEXT NOT NULL, CHECK (NOT empty(inventory_id)),
  when_deleted TIMESTAMP WITH TIME ZONE NOT NULL,
  UNIQUE (inventory_id)
) TABLESPACE pg_default;

CREATE INDEX ON deleted_systems(when_deleted);

GRANT SELECT, INSERT, UPDATE, DELETE ON deleted_systems TO listener;
GRANT SELECT, INSERT, UPDATE, DELETE ON deleted_systems TO manager;


-- repo
CREATE TABLE IF NOT EXISTS repo (
  id SERIAL,
  name TEXT NOT NULL UNIQUE, CHECK (NOT empty(name)),
  PRIMARY KEY (id)
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON repo TO listener;


-- system_repo
CREATE TABLE IF NOT EXISTS system_repo (
  system_id INT NOT NULL,
  repo_id INT NOT NULL,
  UNIQUE (system_id, repo_id),
  CONSTRAINT system_platform_id
    FOREIGN KEY (system_id)
    REFERENCES system_platform (id),
  CONSTRAINT repo_id
    FOREIGN KEY (repo_id)
    REFERENCES repo (id)
) TABLESPACE pg_default;

CREATE INDEX ON system_repo(system_id);
CREATE INDEX ON system_repo(repo_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_repo TO listener;
GRANT DELETE ON system_repo TO manager;


-- timestamp_kv
CREATE TABLE IF NOT EXISTS timestamp_kv (
  name TEXT NOT NULL UNIQUE, CHECK (NOT empty(name)),
  value TIMESTAMP WITH TIME ZONE NOT NULL
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON timestamp_kv TO vmaas_sync;

-- vmaas_sync needs to delete from this tables to sync CVEs correctly
GRANT DELETE ON system_advisories TO vmaas_sync;
GRANT DELETE ON errata_account_data TO vmaas_sync;

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
