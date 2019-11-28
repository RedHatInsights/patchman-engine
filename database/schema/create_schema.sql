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
        -- decrement affected cve counts for system
        WITH to_update_cves AS (
          SELECT cad.cve_id, cad.status_id AS global_status_id, sv.status_id
          FROM cve_account_data cad INNER JOIN
               system_vulnerabilities sv ON cad.cve_id = sv.cve_id
          WHERE cad.rh_account_id = NEW.rh_account_id AND
                sv.system_id = NEW.id AND
                sv.when_mitigated IS NULL
          ORDER BY cad.cve_id
          FOR UPDATE OF cad
        -- decrement systems_affected and systems_status_divergent in case status is different
        ), update_divergent AS (
          UPDATE cve_account_data cad
          SET systems_affected = systems_affected - 1,
              systems_status_divergent = systems_status_divergent - 1
          FROM to_update_cves
          WHERE cad.cve_id = to_update_cves.cve_id AND
                cad.rh_account_id = NEW.rh_account_id AND
                to_update_cves.global_status_id != to_update_cves.status_id
        )
        -- decrement only systems_affected in case status is same
        UPDATE cve_account_data cad
        SET systems_affected = systems_affected - 1
        FROM to_update_cves
        WHERE cad.cve_id = to_update_cves.cve_id AND
              cad.rh_account_id = NEW.rh_account_id AND
              to_update_cves.global_status_id = to_update_cves.status_id;
        -- delete zero cve counts
        DELETE FROM cve_account_data
        WHERE rh_account_id = NEW.rh_account_id AND
              systems_affected = 0;

      -- system opted in
      ELSIF OLD.opt_out = TRUE AND NEW.opt_out = FALSE THEN
        -- increment affected cve counts for system
        WITH to_update_cves AS (
          SELECT cad.cve_id, cad.status_id AS global_status_id, sv.status_id
          FROM cve_account_data cad INNER JOIN
               system_vulnerabilities sv ON cad.cve_id = sv.cve_id
          WHERE cad.rh_account_id = NEW.rh_account_id AND
                sv.system_id = NEW.id AND
                sv.when_mitigated IS NULL
          ORDER BY cad.cve_id
          FOR UPDATE OF cad
        -- increment systems_affected and systems_status_divergent in case status is different
        ), update_divergent AS (
          UPDATE cve_account_data cad
          SET systems_affected = systems_affected + 1,
              systems_status_divergent = systems_status_divergent + 1
          FROM to_update_cves
          WHERE cad.cve_id = to_update_cves.cve_id AND
                cad.rh_account_id = NEW.rh_account_id AND
                to_update_cves.global_status_id != to_update_cves.status_id
        )
        -- increment only systems_affected in case status is same
        UPDATE cve_account_data cad
        SET systems_affected = systems_affected + 1
        FROM to_update_cves
        WHERE cad.cve_id = to_update_cves.cve_id AND
              cad.rh_account_id = NEW.rh_account_id AND
              to_update_cves.global_status_id = to_update_cves.status_id;
        -- insert cache if not exists
        INSERT INTO cve_account_data (cve_id, rh_account_id, systems_affected)
        SELECT sv.cve_id, NEW.rh_account_id, 1
        FROM system_vulnerabilities sv
        WHERE sv.system_id = NEW.id AND
              sv.when_mitigated IS NULL AND
              NOT EXISTS (
                SELECT 1 FROM cve_account_data
                WHERE rh_account_id = NEW.rh_account_id AND
                      cve_id = sv.cve_id
              )
        ON CONFLICT (cve_id, rh_account_id) DO UPDATE SET
          systems_affected = cve_account_data.systems_affected + EXCLUDED.systems_affected;
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
    -- update cve count for ordered systems
    WITH to_update_systems AS (
      SELECT sp.id
      FROM system_platform sp
      ORDER BY sp.rh_account_id, sp.id
      FOR UPDATE OF sp
    )
    UPDATE system_platform sp SET cve_count_cache = (
      SELECT COUNT(cve_id) FROM system_vulnerabilities sv
      WHERE sv.system_id = sp.id AND sv.when_mitigated IS NULL
    )
    FROM to_update_systems
    WHERE sp.id = to_update_systems.id;

    -- update system count for ordered cves
    WITH locked_rows AS (
      SELECT cad.rh_account_id, cad.cve_id
      FROM cve_account_data cad
      ORDER BY cad.rh_account_id, cad.cve_id
      FOR UPDATE OF cad
    ), current_counts AS (
      SELECT sv.cve_id, sp.rh_account_id, count(sv.system_id) as systems_affected
      FROM system_vulnerabilities sv INNER JOIN
           system_platform sp ON sv.system_id = sp.id
      WHERE sp.last_evaluation IS NOT NULL AND
            sp.opt_out = FALSE AND
            sv.when_mitigated IS NULL
      GROUP BY sv.cve_id, sp.rh_account_id
    ), upserted AS (
      INSERT INTO cve_account_data (cve_id, rh_account_id, systems_affected)
        SELECT cve_id, rh_account_id, systems_affected FROM current_counts
      ON CONFLICT (cve_id, rh_account_id) DO UPDATE SET
        systems_affected = EXCLUDED.systems_affected
    )
    DELETE FROM cve_account_data WHERE (cve_id, rh_account_id) NOT IN (SELECT cve_id, rh_account_id FROM current_counts);
  END;
$refresh_all_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_account_cached_counts(rh_account_in varchar)
  RETURNS void AS
$refresh_account_cached_counts$
  DECLARE
    rh_account_id_in INT;
  BEGIN
    -- update cve count for ordered systems
    SELECT id FROM rh_account WHERE name = rh_account_in INTO rh_account_id_in;
    WITH to_update_systems AS (
      SELECT sp.id
      FROM system_platform sp
      WHERE sp.rh_account_id = rh_account_id_in
      ORDER BY sp.id
      FOR UPDATE OF sp
    )
    UPDATE system_platform sp SET cve_count_cache = (
      SELECT COUNT(cve_id) FROM system_vulnerabilities sv
      WHERE sv.system_id = sp.id AND sv.when_mitigated IS NULL
    )
    FROM to_update_systems
    WHERE sp.id = to_update_systems.id;

    -- update system count for ordered cves
    WITH locked_rows AS (
      SELECT cad.cve_id
      FROM cve_account_data cad
      WHERE cad.rh_account_id = rh_account_id_in
      ORDER BY cad.cve_id
      FOR UPDATE OF cad
    ), current_counts AS (
      SELECT sv.cve_id, count(sv.system_id) as systems_affected
      FROM system_vulnerabilities sv INNER JOIN
           system_platform sp ON sv.system_id = sp.id
      WHERE sp.last_evaluation IS NOT NULL AND
            sp.opt_out = FALSE AND
            sv.when_mitigated IS NULL AND
            sp.rh_account_id = rh_account_id_in
      GROUP BY sv.cve_id
    ), upserted AS (
      INSERT INTO cve_account_data (cve_id, rh_account_id, systems_affected)
        SELECT cve_id, rh_account_id_in, systems_affected FROM current_counts
      ON CONFLICT (cve_id, rh_account_id) DO UPDATE SET
        systems_affected = EXCLUDED.systems_affected
    )
    DELETE FROM cve_account_data WHERE cve_id NOT IN (SELECT cve_id FROM current_counts)
      AND rh_account_id = rh_account_id_in;
  END;
$refresh_account_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_cve_cached_counts(cve_in varchar)
  RETURNS void AS
$refresh_cve_cached_counts$
  DECLARE
    cve_md_id INT;
  BEGIN
    -- update system count for cve
    SELECT id FROM cve_metadata WHERE cve = cve_in INTO cve_md_id;
    WITH locked_rows AS (
      SELECT cad.rh_account_id
      FROM cve_account_data cad
      WHERE cad.cve_id = cve_md_id
      ORDER BY cad.rh_account_id
      FOR UPDATE OF cad
    ), current_counts AS (
      SELECT sp.rh_account_id, count(sv.system_id) as systems_affected
      FROM system_vulnerabilities sv INNER JOIN
           system_platform sp ON sv.system_id = sp.id
      WHERE sp.last_evaluation IS NOT NULL AND
            sp.opt_out = FALSE AND
            sv.when_mitigated IS NULL AND
            sv.cve_id = cve_md_id
      GROUP BY sp.rh_account_id
    ), upserted AS (
      INSERT INTO cve_account_data (cve_id, rh_account_id, systems_affected)
        SELECT cve_md_id, rh_account_id, systems_affected FROM current_counts
      ON CONFLICT (cve_id, rh_account_id) DO UPDATE SET
        systems_affected = EXCLUDED.systems_affected
    )
    DELETE FROM cve_account_data WHERE rh_account_id NOT IN (SELECT rh_account_id FROM current_counts)
      AND cve_id = cve_md_id;
  END;
$refresh_cve_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_cve_account_cached_counts(cve_in varchar, rh_account_in varchar)
  RETURNS void AS
$refresh_cve_account_cached_counts$
  DECLARE
    cve_md_id INT;
    rh_account_id_in INT;
  BEGIN
    -- update system count for ordered cves
    SELECT id FROM cve_metadata WHERE cve = cve_in INTO cve_md_id;
    SELECT id FROM rh_account WHERE name = rh_account_in INTO rh_account_id_in;
    WITH locked_rows AS (
      SELECT cad.rh_account_id, cad.cve_id
      FROM cve_account_data cad
      WHERE cad.cve_id = cve_md_id AND
            cad.rh_account_id = rh_account_id_in
      FOR UPDATE OF cad
    ), current_counts AS (
      SELECT sv.cve_id, sp.rh_account_id, count(sv.system_id) as systems_affected
      FROM system_vulnerabilities sv INNER JOIN
           system_platform sp ON sv.system_id = sp.id
      WHERE sp.last_evaluation IS NOT NULL AND
            sp.opt_out = FALSE AND
            sv.when_mitigated IS NULL AND
            sv.cve_id = cve_md_id AND
            sp.rh_account_id = rh_account_id_in
      GROUP BY sv.cve_id, sp.rh_account_id
    ), upserted AS (
      INSERT INTO cve_account_data (cve_id, rh_account_id, systems_affected)
        SELECT cve_md_id, rh_account_id_in, systems_affected FROM current_counts
      ON CONFLICT (cve_id, rh_account_id) DO UPDATE SET
        systems_affected = EXCLUDED.systems_affected
    )
    DELETE FROM cve_account_data WHERE NOT EXISTS (SELECT 1 FROM current_counts)
      AND cve_id = cve_md_id
      AND rh_account_id = rh_account_id_in;
  END;
$refresh_cve_account_cached_counts$
  LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_system_cached_counts(inventory_id_in varchar)
  RETURNS void AS
$refresh_system_cached_counts$
  BEGIN
    -- update cve count for system
    UPDATE system_platform sp SET cve_count_cache = (
      SELECT COUNT(cve_id) FROM system_vulnerabilities sv
      WHERE sv.system_id = sp.id AND sv.when_mitigated IS NULL
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
    DELETE FROM system_vulnerabilities
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
  cve_count_cache INT NOT NULL DEFAULT 0,
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
GRANT UPDATE (cve_count_cache), DELETE ON system_platform TO manager;

-- cve_impact
CREATE TABLE IF NOT EXISTS cve_impact (
  id INT NOT NULL,
  name TEXT NOT NULL UNIQUE, CHECK (NOT empty(name)),
  PRIMARY KEY (id)
)TABLESPACE pg_default;

INSERT INTO cve_impact (id, name) VALUES
  (0, 'NotSet'), (1, 'None'), (2, 'Low'), (3, 'Medium'), (4, 'Moderate'),
  (5, 'Important'), (6, 'High'), (7, 'Critical');


-- cve_metadata
CREATE TABLE IF NOT EXISTS cve_metadata (
  id SERIAL,
  cve TEXT NOT NULL, CHECK (NOT empty(cve)),
  description TEXT NOT NULL, CHECK (NOT empty(description)),
  impact_id INT NOT NULL,
  public_date TIMESTAMP WITH TIME ZONE NULL,
  modified_date TIMESTAMP WITH TIME ZONE NULL,
  cvss3_score NUMERIC(5,3),
  cvss3_metrics TEXT,
  cvss2_score NUMERIC(5,3),
  cvss2_metrics TEXT,
  redhat_url TEXT,
  secondary_url TEXT,
  UNIQUE (cve),
  PRIMARY KEY (id),
  CONSTRAINT impact_id
    FOREIGN KEY (impact_id)
    REFERENCES cve_impact (id)
) TABLESPACE pg_default;

CREATE INDEX ON cve_metadata(impact_id);
CREATE INDEX ON cve_metadata(cvss3_score);
CREATE INDEX ON cve_metadata(cvss2_score);

GRANT SELECT, INSERT, UPDATE, DELETE ON cve_metadata TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON cve_metadata TO vmaas_sync;


-- status table
CREATE TABLE IF NOT EXISTS status (
  id INT NOT NULL,
  name TEXT NOT NULL UNIQUE, CHECK (NOT empty(name)),
  PRIMARY KEY (id)
)TABLESPACE pg_default;

INSERT INTO status (id, name) VALUES
  (0, 'Not Reviewed'), (1, 'In-Review'), (2, 'On-Hold'), (3, 'Scheduled for Patch'), (4, 'Resolved'),
  (5, 'No Action - Risk Accepted'), (6, 'Resolved via Mitigation (e.g. done without deploying a patch)') ;


-- system_vulnerabilities
CREATE TABLE IF NOT EXISTS system_vulnerabilities (
  id SERIAL,
  system_id INT NOT NULL,
  cve_id INT NOT NULL,
  first_reported TIMESTAMP WITH TIME ZONE NOT NULL,
  when_mitigated TIMESTAMP WITH TIME ZONE DEFAULT NULL,
  status_id INT DEFAULT 0,
  status_text TEXT,
  UNIQUE (system_id, cve_id),
  PRIMARY KEY (id),
  CONSTRAINT system_platform_id
    FOREIGN KEY (system_id)
    REFERENCES system_platform (id),
  CONSTRAINT cve_metadata_cve_id
    FOREIGN KEY (cve_id)
    REFERENCES cve_metadata (id),
  CONSTRAINT status_id
    FOREIGN KEY (status_id)
    REFERENCES status (id)
) TABLESPACE pg_default;

CREATE INDEX ON system_vulnerabilities(status_id);

CREATE TRIGGER system_vulnerabilities_set_first_reported BEFORE INSERT ON system_vulnerabilities
  FOR EACH ROW EXECUTE PROCEDURE set_first_reported();

GRANT SELECT, INSERT, UPDATE, DELETE ON system_vulnerabilities TO evaluator;
-- manager needs to be able to update things like 'status' on a sysid/cve combination, also needs to delete
GRANT UPDATE, DELETE ON system_vulnerabilities TO manager;
-- manager needs to be able to update opt_out column
GRANT UPDATE (opt_out) ON system_platform TO manager;
-- listener deletes systems
GRANT DELETE ON system_vulnerabilities TO listener;

-- business_risk table
CREATE TABLE IF NOT EXISTS business_risk (
  id INT NOT NULL,
  name VARCHAR NOT NULL UNIQUE,
  CHECK (NOT empty(name)),
  PRIMARY KEY (id)
) TABLESPACE pg_default;

INSERT INTO business_risk (id, name) VALUES
  (0, 'Not Defined'), (1, 'Low'), (2, 'Medium'), (3, 'High');

-- cve_preferences
CREATE TABLE IF NOT EXISTS cve_account_data (
  cve_id INT NOT NULL,
  rh_account_id INT NOT NULL,
  business_risk_id INT NOT NULL DEFAULT 0,
  business_risk_text TEXT,
  status_id INT NOT NULL DEFAULT 0,
  status_text TEXT,
  systems_affected INT NOT NULL DEFAULT 0,
  systems_status_divergent INT NOT NULL DEFAULT 0,
  CONSTRAINT cve_id
    FOREIGN KEY (cve_id)
    REFERENCES cve_metadata (id),
  CONSTRAINT rh_account_id
    FOREIGN KEY (rh_account_id)
    REFERENCES rh_account (id),
  CONSTRAINT business_risk_id
    FOREIGN KEY (business_risk_id)
    REFERENCES business_risk (id),
  CONSTRAINT status_id
    FOREIGN KEY (status_id)
    REFERENCES status (id),
  UNIQUE (cve_id, rh_account_id)
) TABLESPACE pg_default;

-- manager needs to write into cve_account_preferences table
GRANT SELECT, INSERT, UPDATE, DELETE ON cve_account_data TO manager;

-- manager user needs to change this table for opt-out functionality
GRANT SELECT, INSERT, UPDATE, DELETE ON cve_account_data TO manager;
-- evaluator user needs to change this table
GRANT SELECT, INSERT, UPDATE, DELETE ON cve_account_data TO evaluator;
-- listner user needs to change this table when deleting system
GRANT SELECT, INSERT, UPDATE, DELETE ON cve_account_data TO listener;


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
GRANT DELETE ON system_vulnerabilities TO vmaas_sync;
GRANT DELETE ON cve_account_data TO vmaas_sync;

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
