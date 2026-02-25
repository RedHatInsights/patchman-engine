CREATE TABLE IF NOT EXISTS schema_migrations
(
    version bigint  NOT NULL,
    dirty   boolean NOT NULL,
    PRIMARY KEY (version)
) TABLESPACE pg_default;


INSERT INTO schema_migrations
VALUES (146, false);

-- ---------------------------------------------------------------------------
-- Functions
-- ---------------------------------------------------------------------------

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE COLLATION IF NOT EXISTS numeric (provider = icu, locale = 'en-u-kn-true');

-- empty
CREATE OR REPLACE FUNCTION empty(t TEXT)
    RETURNS BOOLEAN as
$$
BEGIN
    RETURN t ~ '^[[:space:]]*$';
END;
$$ LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE;

CREATE OR REPLACE FUNCTION ternary(cond BOOL, iftrue ANYELEMENT, iffalse ANYELEMENT)
    RETURNS ANYELEMENT
AS
$$
SELECT CASE WHEN cond = TRUE THEN iftrue else iffalse END;
$$ LANGUAGE SQL IMMUTABLE;

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
       (NEW.json_checksum <> OLD.json_checksum OR NEW.yum_checksum <> OLD.yum_checksum) THEN
        NEW.unchanged_since := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$check_unchanged$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION on_system_update()
-- this trigger updates advisory_account_data when server changes its stale flag
    RETURNS TRIGGER
AS
$system_update$
DECLARE
    was_counted  BOOLEAN;
    should_count BOOLEAN;
    change       INT;
BEGIN
    -- Ignore not yet evaluated systems
    IF TG_OP != 'UPDATE' OR NOT EXISTS (
        SELECT 1
        FROM system_patch
        WHERE system_id = NEW.id 
          AND rh_account_id = NEW.rh_account_id
          AND last_evaluation IS NOT NULL
    ) THEN
        RETURN NEW;
    END IF;

    was_counted := OLD.stale = FALSE;
    should_count := NEW.stale = FALSE;

    -- Determine what change we are performing
    IF was_counted and NOT should_count THEN
        change := -1;
    ELSIF NOT was_counted AND should_count THEN
        change := 1;
    ELSE
        -- No change
        RETURN NEW;
    END IF;

    -- insert/update advisories linked to the server
    INSERT
      INTO advisory_account_data (advisory_id, rh_account_id, systems_installable, systems_applicable)
    SELECT sa.advisory_id, NEW.rh_account_id,
           case when sa.status_id = 0 then change else 0 end as systems_installable,
           change as systems_applicable
      FROM system_advisories sa
     WHERE sa.system_id = NEW.id AND sa.rh_account_id = NEW.rh_account_id
     ORDER BY sa.advisory_id
        ON CONFLICT (advisory_id, rh_account_id) DO UPDATE
           SET systems_installable = advisory_account_data.systems_installable + EXCLUDED.systems_installable,
               systems_applicable = advisory_account_data.systems_applicable + EXCLUDED.systems_applicable;
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;

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
        SELECT sa.advisory_id, sa.rh_account_id,
               count(sa.*) filter (where sa.status_id = 0) as systems_installable,
               count(sa.*) as systems_applicable
          FROM system_advisories sa
          JOIN system_inventory si
            ON sa.rh_account_id = si.rh_account_id AND sa.system_id = si.id
          JOIN system_patch sp
            ON si.id = sp.system_id AND sp.rh_account_id = si.rh_account_id
         WHERE sp.last_evaluation IS NOT NULL
           AND si.stale = FALSE
           AND (sa.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
           AND (si.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
         GROUP BY sa.advisory_id, sa.rh_account_id
    ),
        upserted AS (
            INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_installable, systems_applicable)
                 SELECT advisory_id, rh_account_id, systems_installable, systems_applicable
                   FROM current_counts
            ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_installable = EXCLUDED.systems_installable,
                     systems_applicable = EXCLUDED.systems_applicable
         )
    DELETE FROM advisory_account_data
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

CREATE OR REPLACE FUNCTION refresh_system_caches(system_id_in BIGINT DEFAULT NULL,
                                                 rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS INTEGER AS
$refresh_system$
DECLARE
    COUNT INTEGER;
BEGIN
    WITH system_advisories_count AS (
        SELECT si.rh_account_id, si.id,
               COUNT(advisory_id) FILTER (WHERE sa.status_id = 0) as installable_total,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 1 AND sa.status_id = 0) AS installable_enhancement,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 2 AND sa.status_id = 0) AS installable_bugfix,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 3 AND sa.status_id = 0) as installable_security,
               COUNT(advisory_id) as applicable_total,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 1) AS applicable_enhancement,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 2) AS applicable_bugfix,
               COUNT(advisory_id) FILTER (WHERE am.advisory_type_id = 3) as applicable_security
          FROM system_inventory si  -- this table ensures even systems without any system_advisories are in results
          LEFT JOIN system_advisories sa
            ON si.rh_account_id = sa.rh_account_id AND si.id = sa.system_id
          LEFT JOIN advisory_metadata am
            ON sa.advisory_id = am.id
         WHERE (si.id = system_id_in OR system_id_in IS NULL)
           AND (si.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
         GROUP BY si.rh_account_id, si.id
         ORDER BY si.rh_account_id, si.id
    )
        UPDATE system_patch sp
           SET installable_advisory_count_cache = sc.installable_total,
               installable_advisory_enh_count_cache = sc.installable_enhancement,
               installable_advisory_bug_count_cache = sc.installable_bugfix,
               installable_advisory_sec_count_cache = sc.installable_security,
               applicable_advisory_count_cache = sc.applicable_total,
               applicable_advisory_enh_count_cache = sc.applicable_enhancement,
               applicable_advisory_bug_count_cache = sc.applicable_bugfix,
               applicable_advisory_sec_count_cache = sc.applicable_security
          FROM system_advisories_count sc
         WHERE sp.rh_account_id = sc.rh_account_id AND sp.system_id = sc.id
           AND (sp.system_id = system_id_in OR system_id_in IS NULL)
           AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);

    GET DIAGNOSTICS COUNT = ROW_COUNT;
    RETURN COUNT;
END;
$refresh_system$ LANGUAGE plpgsql;

-- update system advisories counts (all and according types)
CREATE OR REPLACE FUNCTION update_system_caches(system_id_in BIGINT)
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
    advisory_id_id BIGINT;
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
    advisory_md_id   BIGINT;
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

    SELECT id FROM system_inventory WHERE inventory_id = inventory_id_in INTO system_id;

    PERFORM refresh_system_caches(system_id, NULL);
END;
$refresh_system_cached_counts$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION delete_system(inventory_id_in uuid)
    RETURNS TABLE
            (
                deleted_inventory_id uuid
            )
AS
$delete_system$
DECLARE
    v_system_id  INT;
    v_account_id INT;
BEGIN
    -- opt out to refresh cache and then delete
    SELECT id, rh_account_id
    FROM system_inventory
    WHERE inventory_id = inventory_id_in
    LIMIT 1
        FOR UPDATE OF system_inventory
    INTO v_system_id, v_account_id;

    IF v_system_id IS NULL OR v_account_id IS NULL THEN
        RAISE NOTICE 'Not found';
        RETURN;
    END IF;

    UPDATE system_inventory
    SET stale = true
    WHERE rh_account_id = v_account_id
      AND id = v_system_id;

    DELETE
    FROM system_advisories
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    DELETE
    FROM system_repo
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    DELETE
    FROM system_package2
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    DELETE
    FROM system_patch
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    RETURN QUERY DELETE FROM system_inventory
        WHERE rh_account_id = v_account_id AND
              id = v_system_id
        RETURNING inventory_id;
END;
$delete_system$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION delete_systems(inventory_ids UUID[])
    RETURNS INTEGER
AS
$$
DECLARE
    tmp_cnt INTEGER;
BEGIN

    WITH systems as (
        SELECT rh_account_id, id
        FROM system_inventory
        WHERE inventory_id = ANY (inventory_ids)
        ORDER BY rh_account_id, id FOR UPDATE OF system_inventory),
         marked as (
             UPDATE system_inventory sp
                 SET stale = true
                 WHERE (rh_account_id, id) in (select rh_account_id, id from systems)
         ),
         advisories as (
             DELETE
                 FROM system_advisories
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         repos as (
             DELETE
                 FROM system_repo
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         packages2 as (
             DELETE
                 FROM system_package2
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         patch_systems as (
             DELETE
                 FROM system_patch
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         deleted as (
             DELETE
                 FROM system_inventory
                     WHERE (rh_account_id, id) in (select rh_account_id, id from systems)
                     RETURNING id
         )
    SELECT count(*)
    FROM deleted
    INTO tmp_cnt;

    RETURN tmp_cnt;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION delete_culled_systems(delete_limit INTEGER)
    RETURNS INTEGER
AS
$fun$
DECLARE
    ids UUID[];
BEGIN
    ids := ARRAY(
            SELECT inventory_id
            FROM system_inventory
            WHERE culled_timestamp < now()
            ORDER BY id
            LIMIT delete_limit
        );
    return delete_systems(ids);
END;
$fun$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION mark_stale_systems(mark_limit integer)
    RETURNS INTEGER
AS
$fun$
DECLARE
    marked integer;
BEGIN
    WITH ids AS (
        SELECT rh_account_id, id, stale_warning_timestamp < now() as expired
        FROM system_inventory
        WHERE stale != (stale_warning_timestamp < now())
        ORDER BY rh_account_id, id FOR UPDATE OF system_inventory
        LIMIT mark_limit
    )
    UPDATE system_inventory si
    SET stale = ids.expired
    FROM ids
    WHERE si.rh_account_id = ids.rh_account_id
      AND si.id = ids.id;
    GET DIAGNOSTICS marked = ROW_COUNT;
    RETURN marked;
END;
$fun$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION hash_partition_id(id int, parts int)
    RETURNS int AS
$$
    BEGIN
        -- src/include/common/hashfn.h:83
        --  a ^= b + UINT64CONST(0x49a0f4dd15e5a8e3) + (a << 54) + (a >> 7);
        -- => 8816678312871386365
        -- src/include/catalog/partition.h:20
        --  #define HASH_PARTITION_SEED UINT64CONST(0x7A5B22367996DCFD)
        -- => 5305509591434766563
        RETURN (((hashint4extended(id, 8816678312871386365)::numeric + 5305509591434766563) % parts + parts)::int % parts);
    END;
$$ LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE;

CREATE OR REPLACE FUNCTION create_table_partitions(tbl regclass, parts INTEGER, rest text)
    RETURNS VOID AS
$$
DECLARE
    I INTEGER;
BEGIN
    I := 0;
    WHILE I < parts
        LOOP
            EXECUTE 'CREATE TABLE IF NOT EXISTS ' || text(tbl) || '_' || text(I) || ' PARTITION OF ' || text(tbl) ||
                    ' FOR VALUES WITH ' || ' ( MODULUS ' || text(parts) || ', REMAINDER ' || text(I) || ')' ||
                    rest || ';';
            I = I + 1;
        END LOOP;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_table_partition_triggers(name text, trig_type text, tbl regclass, trig_text text)
    RETURNS VOID AS
$$
DECLARE
    r record;
    trig_name text;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(tbl)
    LOOP
        trig_name := name || substr(r.relname, length(text(tbl)) +1 );
        EXECUTE 'DROP TRIGGER IF EXISTS ' || trig_name || ' ON ' || r.relname;
        EXECUTE 'CREATE TRIGGER ' || trig_name ||
                ' ' || trig_type || ' ON ' || r.relname || ' ' || trig_text || ';';
    END LOOP;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION rename_table_with_partitions(tbl regclass, oldtext text, newtext text)
    RETURNS VOID AS
$$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(tbl)
    LOOP
        EXECUTE 'ALTER TABLE IF EXISTS ' || r.relname || ' RENAME TO ' || replace(r.relname, oldtext, newtext);
    END LOOP;
    EXECUTE 'ALTER TABLE IF EXISTS ' || text(tbl) || ' RENAME TO ' || replace(text(tbl), oldtext, newtext);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION drop_table_partition_triggers(name text, trig_type text, tbl regclass, trig_text text)
    RETURNS VOID AS
$$
DECLARE
    r record;
    trig_name text;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(tbl)
    LOOP
        trig_name := name || substr(r.relname, length(text(tbl)) +1 );
        EXECUTE 'DROP TRIGGER IF EXISTS ' || trig_name || ' ON ' || r.relname;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION rename_index_with_partitions(idx regclass, oldtext text, newtext text)
    RETURNS VOID AS
$$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(idx)
    LOOP
        EXECUTE 'ALTER INDEX IF EXISTS ' || r.relname || ' RENAME TO ' || replace(r.relname, oldtext, newtext);
    END LOOP;
    EXECUTE 'ALTER INDEX IF EXISTS ' || text(idx) || ' RENAME TO ' || replace(text(idx), oldtext, newtext);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION grant_table_partitions(perms text, tbl regclass, grantie text)
    RETURNS VOID AS
$$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(tbl)
    LOOP
        EXECUTE 'GRANT ' || perms || ' ON TABLE ' || r.relname || ' TO ' || grantie;
    END LOOP;
    EXECUTE 'GRANT ' || perms || ' ON TABLE ' || text(tbl) || ' TO ' || grantie;
END;
$$ LANGUAGE plpgsql;


-- ---------------------------------------------------------------------------
-- Tables
-- ---------------------------------------------------------------------------

-- rh_account
CREATE TABLE IF NOT EXISTS rh_account
(
    id                      INT GENERATED BY DEFAULT AS IDENTITY,
    name                    TEXT UNIQUE CHECK (NOT empty(name)),
    org_id                  TEXT UNIQUE CHECK (NOT empty(org_id)),
    valid_package_cache     BOOLEAN NOT NULL DEFAULT FALSE,
    valid_advisory_cache    BOOLEAN NOT NULL DEFAULT FALSE,
    CHECK (name IS NOT NULL OR org_id IS NOT NULL),
    PRIMARY KEY (id)
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON rh_account TO listener;
GRANT SELECT, UPDATE ON rh_account TO evaluator;
GRANT SELECT, INSERT, UPDATE ON rh_account TO manager;
GRANT UPDATE ON rh_account TO vmaas_sync;

CREATE TABLE reporter
(
    id   INT  NOT NULL,
    name TEXT NOT NULL UNIQUE CHECK ( not empty(name) ),
    PRIMARY KEY (id)
);

INSERT INTO reporter (id, name)
VALUES (1, 'puptoo'),
       (2, 'rhsm-conduit'),
       (3, 'yupana'),
       (4, 'rhsm-system-profile-bridge'),
       (5, 'satellite'),
       (6, 'discovery')
ON CONFLICT DO NOTHING;

-- templates
CREATE TABLE IF NOT EXISTS template
(
    id            BIGINT            GENERATED BY DEFAULT AS IDENTITY,
    rh_account_id INT               NOT NULL REFERENCES rh_account (id),
    uuid          UUID              NOT NULL,
    name          TEXT              NOT NULL CHECK (not empty(name)),
    description   TEXT              CHECK (NOT empty(description)),
    config        JSONB,
    creator       TEXT              CHECK (NOT empty(creator)),
    published     TIMESTAMP WITH TIME ZONE,
    last_edited   TIMESTAMP WITH TIME ZONE,
    environment_id TEXT             NOT NULL CHECK (not empty(environment_id)),
    arch           TEXT             CHECK (not empty(arch)),
    version        TEXT             CHECK (not empty(version)),
    PRIMARY KEY (rh_account_id, id),
    UNIQUE(rh_account_id, uuid)
) PARTITION BY HASH (rh_account_id);

SELECT create_table_partitions('template', 16,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

SELECT grant_table_partitions('SELECT', 'template', 'manager');
SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'template', 'listener');
SELECT grant_table_partitions('SELECT', 'template', 'evaluator');
SELECT grant_table_partitions('SELECT', 'template', 'vmaas_sync');

-- system_inventory
CREATE TABLE IF NOT EXISTS system_inventory
(
    id                                  BIGINT GENERATED BY DEFAULT AS IDENTITY,
    inventory_id                        UUID        NOT NULL,
    rh_account_id                       INT         NOT NULL REFERENCES rh_account (id),
    vmaas_json                          TEXT        CHECK (NOT empty(vmaas_json)),
    json_checksum                       TEXT        CHECK (NOT empty(json_checksum)),
    last_updated                        TIMESTAMPTZ NOT NULL,
    unchanged_since                     TIMESTAMPTZ NOT NULL,
    last_upload                         TIMESTAMPTZ,
    stale                               BOOLEAN     NOT NULL DEFAULT false,
    display_name                        TEXT        NOT NULL CHECK (NOT empty(display_name)),
    reporter_id                         INT         REFERENCES reporter (id),
    yum_updates                         JSONB,
    yum_checksum                        TEXT        CHECK (NOT empty(yum_checksum)),
    satellite_managed                   BOOLEAN     NOT NULL DEFAULT false,
    built_pkgcache                      BOOLEAN     NOT NULL DEFAULT false,
    arch                                TEXT        CHECK (NOT empty(arch)),
    bootc                               BOOLEAN     NOT NULL DEFAULT false,
    tags                                JSONB       NOT NULL,
    created                             TIMESTAMPTZ NOT NULL,
    stale_timestamp                     TIMESTAMPTZ,
    stale_warning_timestamp             TIMESTAMPTZ,
    culled_timestamp                    TIMESTAMPTZ,
    os_name                             TEXT        CHECK (NOT empty(os_name)),
    os_major                            SMALLINT,
    os_minor                            SMALLINT,
    rhsm_version                        TEXT        CHECK (NOT empty(rhsm_version)),
    subscription_manager_id             UUID,
    sap_workload                        BOOLEAN     NOT NULL DEFAULT false,
    sap_workload_sids                   TEXT ARRAY  CHECK (array_length(sap_workload_sids,1) > 0 or sap_workload_sids is null),
    ansible_workload                    BOOLEAN     NOT NULL DEFAULT false,
    ansible_workload_controller_version TEXT        CHECK (NOT empty(ansible_workload_controller_version)),
    mssql_workload                      BOOLEAN     NOT NULL DEFAULT false,
    mssql_workload_version              TEXT        CHECK (NOT empty(mssql_workload_version)),
    workspaces                          JSONB,
    PRIMARY KEY (rh_account_id, id),
    UNIQUE (rh_account_id, inventory_id)
) PARTITION BY HASH (rh_account_id);

SELECT create_table_partitions('system_inventory', 16,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
                                 TABLESPACE pg_default$$);

GRANT SELECT, INSERT, UPDATE ON system_inventory TO listener;
GRANT SELECT, UPDATE, DELETE ON system_inventory TO vmaas_sync; -- vmaas_sync performs system culling
GRANT SELECT, UPDATE (stale) ON system_inventory TO manager; -- manager needs to be able to update opt_out column
GRANT SELECT, UPDATE ON system_inventory TO manager; -- manager needs to be able to update opt_out column
GRANT SELECT, UPDATE ON system_inventory TO evaluator;

SELECT create_table_partition_triggers('system_inventory_set_last_updated',
                                       $$BEFORE INSERT OR UPDATE$$,
                                       'system_inventory',
                                       $$FOR EACH ROW EXECUTE PROCEDURE set_last_updated()$$);

SELECT create_table_partition_triggers('system_inventory_check_unchanged',
                                       $$BEFORE INSERT OR UPDATE$$,
                                       'system_inventory',
                                       $$FOR EACH ROW EXECUTE PROCEDURE check_unchanged()$$);

SELECT create_table_partition_triggers('system_inventory_on_update',
                                       $$AFTER UPDATE$$,
                                       'system_inventory',
                                       $$FOR EACH ROW EXECUTE PROCEDURE on_system_update()$$);

CREATE INDEX IF NOT EXISTS system_inventory_inventory_id_idx ON system_inventory (inventory_id);
CREATE INDEX IF NOT EXISTS system_inventory_tags_index ON system_inventory USING GIN (tags JSONB_PATH_OPS);
CREATE INDEX IF NOT EXISTS system_inventory_stale_timestamp_index ON system_inventory (stale_timestamp);
CREATE INDEX IF NOT EXISTS system_inventory_workspaces_index ON system_inventory USING GIN (workspaces);

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
    preference INTEGER NOT NULL DEFAULT 0,
    CHECK (NOT empty(name)),
    PRIMARY KEY (id)
) TABLESPACE pg_default;

INSERT INTO advisory_type (id, name, preference)
VALUES (0, 'unknown', 100),
       (1, 'enhancement', 300),
       (2, 'bugfix', 400),
       (3, 'security', 500),
       (4, 'unspecified', 200)
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
    id               BIGINT GENERATED BY DEFAULT AS IDENTITY,
    name             TEXT                     NOT NULL CHECK (NOT empty(name)),
    description      TEXT                     NOT NULL CHECK (NOT empty(description)),
    synopsis         TEXT                     NOT NULL CHECK (NOT empty(synopsis)),
    summary          TEXT                     NOT NULL CHECK (NOT empty(summary)),
    solution         TEXT                     CHECK (NOT empty(solution)),
    advisory_type_id INT                      NOT NULL,
    public_date      TIMESTAMP WITH TIME ZONE NULL,
    modified_date    TIMESTAMP WITH TIME ZONE NULL,
    url              TEXT CHECK (NOT empty(url)),
    severity_id      INT,
    package_data     JSONB,
    cve_list         JSONB,
    reboot_required  BOOLEAN NOT NULL DEFAULT false,
    release_versions JSONB,
    synced           BOOLEAN NOT NULL DEFAULT false,
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
VALUES (0, 'Installable'),
       (1, 'Applicable')
ON CONFLICT DO NOTHING;


-- system_advisories
CREATE TABLE IF NOT EXISTS system_advisories
(
    rh_account_id  INT                      NOT NULL,
    system_id      BIGINT                   NOT NULL,
    advisory_id    BIGINT                   NOT NULL,
    first_reported TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status_id      INT                      NOT NULL,
    PRIMARY KEY (rh_account_id, system_id, advisory_id),
    CONSTRAINT advisory_metadata_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id)
) PARTITION BY HASH (rh_account_id);

SELECT create_table_partitions('system_advisories', 32,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_advisories TO evaluator;
-- manager needs to be able to update things like 'status' on a sysid/advisory combination, also needs to delete
GRANT UPDATE, DELETE ON system_advisories TO manager;
-- listener deletes systems, TODO: temporary added evaluator permissions to listener
GRANT SELECT, INSERT, UPDATE, DELETE ON system_advisories TO listener;
-- vmaas_sync needs to delete culled systems, which cascades to system_advisories
GRANT SELECT, DELETE ON system_advisories TO vmaas_sync;

-- advisory_account_data
CREATE TABLE IF NOT EXISTS advisory_account_data
(
    advisory_id              BIGINT NOT NULL,
    rh_account_id            INT NOT NULL,
    systems_applicable       INT NOT NULL DEFAULT 0,
    systems_installable      INT NOT NULL DEFAULT 0,
    notified                 TIMESTAMP WITH TIME ZONE NULL,
    CONSTRAINT advisory_metadata_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id),
    CONSTRAINT rh_account_id
        FOREIGN KEY (rh_account_id)
            REFERENCES rh_account (id),
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

-- indexes for filtering systems_applicable, systems_installable
CREATE INDEX ON advisory_account_data (systems_applicable);
CREATE INDEX ON advisory_account_data (systems_installable);

-- repo
CREATE TABLE IF NOT EXISTS repo
(
    id              BIGINT GENERATED BY DEFAULT AS IDENTITY,
    name            TEXT NOT NULL UNIQUE,
    third_party     BOOLEAN NOT NULL DEFAULT true,
    CHECK (NOT empty(name)),
    PRIMARY KEY (id)
) TABLESPACE pg_default;

GRANT SELECT, INSERT, UPDATE, DELETE ON repo TO listener;
GRANT SELECT, INSERT, UPDATE, DELETE ON repo TO evaluator;


-- system_repo
CREATE TABLE IF NOT EXISTS system_repo
(
    system_id     BIGINT NOT NULL,
    repo_id       BIGINT NOT NULL,
    rh_account_id INT NOT NULL,
    UNIQUE (rh_account_id, system_id, repo_id),
    CONSTRAINT system_inventory_id
        FOREIGN KEY (rh_account_id, system_id)
            REFERENCES system_inventory (rh_account_id, id),
    CONSTRAINT repo_id
        FOREIGN KEY (repo_id)
            REFERENCES repo (id)
) TABLESPACE pg_default;

CREATE INDEX ON system_repo (repo_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_repo TO listener;
GRANT DELETE ON system_repo TO manager;
GRANT SELECT, INSERT, UPDATE, DELETE ON system_repo TO evaluator;
GRANT SELECT, DELETE on system_repo to vmaas_sync;

-- the following constraints are enabled here not directly in the table definitions
-- to make new schema equal to the migrated schema
ALTER TABLE system_advisories
    ADD CONSTRAINT system_inventory_id
        FOREIGN KEY (rh_account_id, system_id)
            REFERENCES system_inventory (rh_account_id, id),
    ADD CONSTRAINT status_id
        FOREIGN KEY (status_id)
            REFERENCES status (id);

CREATE TABLE IF NOT EXISTS package_name
(
    id   BIGINT GENERATED BY DEFAULT AS IDENTITY NOT NULL PRIMARY KEY,
    name TEXT                                 NOT NULL CHECK (NOT empty(name)) UNIQUE,
    -- "cache" latest summary for given package name here to display it on /packages API
    -- without joining other tables
    summary      TEXT                         CHECK (NOT empty(summary))
);

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE package_name TO vmaas_sync;
GRANT SELECT, INSERT, UPDATE ON TABLE package_name TO evaluator;

CREATE TABLE IF NOT EXISTS strings
(
    id    BYTEA NOT NULL PRIMARY KEY,
    value TEXT  NOT NULL CHECK (NOT empty(value))
);

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE strings TO vmaas_sync;

CREATE TABLE IF NOT EXISTS package
(
    id               BIGINT GENERATED BY DEFAULT AS IDENTITY NOT NULL PRIMARY KEY,
    name_id          BIGINT                                  NOT NULL REFERENCES package_name,
    evra             TEXT                                 NOT NULL CHECK (NOT empty(evra)),
    description_hash BYTEA                                         REFERENCES strings (id),
    summary_hash     BYTEA                                         REFERENCES strings (id),
    advisory_id      BIGINT REFERENCES advisory_metadata (id),
    synced           BOOLEAN                              NOT NULL DEFAULT false,
    UNIQUE (name_id, evra)
) WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
  TABLESPACE pg_default;

CREATE UNIQUE INDEX IF NOT EXISTS package_evra_idx on package (evra, name_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE package TO vmaas_sync;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE package TO evaluator;

CREATE TABLE IF NOT EXISTS system_package2
(
    rh_account_id  INT    NOT NULL,
    system_id      BIGINT NOT NULL,
    name_id        BIGINT NOT NULL REFERENCES package_name (id),
    package_id     BIGINT NOT NULL REFERENCES package (id),
    -- Use null to represent up-to-date packages
    installable_id BIGINT REFERENCES package (id),
    applicable_id  BIGINT REFERENCES package (id),

    PRIMARY KEY (rh_account_id, system_id, package_id),
    CONSTRAINT system_inventory_id
        FOREIGN KEY (rh_account_id, system_id)
            REFERENCES system_inventory (rh_account_id, id)
) PARTITION BY HASH (rh_account_id);

CREATE INDEX IF NOT EXISTS system_package2_account_pkg_name_idx
    ON system_package2 (rh_account_id, name_id) INCLUDE (system_id, package_id, installable_id, applicable_id);

CREATE INDEX IF NOT EXISTS system_package2_package_id_idx on system_package2 (package_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_package2 TO evaluator;
GRANT SELECT, UPDATE, DELETE ON system_package2 TO listener;
GRANT SELECT, UPDATE, DELETE ON system_package2 TO manager;
GRANT SELECT, UPDATE, DELETE ON system_package2 TO vmaas_sync;

SELECT create_table_partitions('system_package2', 128,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

-- package_account_data
CREATE TABLE IF NOT EXISTS package_account_data
(
    package_name_id          BIGINT NOT NULL,
    rh_account_id            INT NOT NULL,
    systems_installed        INT NOT NULL DEFAULT 0,
    systems_installable      INT NOT NULL DEFAULT 0,
    systems_applicable       INT NOT NULL DEFAULT 0,
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

-- system_patch
CREATE TABLE IF NOT EXISTS system_patch
(
    system_id                            BIGINT      NOT NULL,
    rh_account_id                        INT         NOT NULL,
    last_evaluation                      TIMESTAMPTZ,
    installable_advisory_count_cache     INT         NOT NULL DEFAULT 0,
    installable_advisory_enh_count_cache INT         NOT NULL DEFAULT 0,
    installable_advisory_bug_count_cache INT         NOT NULL DEFAULT 0,
    installable_advisory_sec_count_cache INT         NOT NULL DEFAULT 0,
    packages_installed                   INT         NOT NULL DEFAULT 0,
    packages_installable                 INT         NOT NULL DEFAULT 0,
    packages_applicable                  INT         NOT NULL DEFAULT 0,
    third_party                          BOOLEAN     NOT NULL DEFAULT false,
    applicable_advisory_count_cache      INT         NOT NULL DEFAULT 0,
    applicable_advisory_enh_count_cache  INT         NOT NULL DEFAULT 0,
    applicable_advisory_bug_count_cache  INT         NOT NULL DEFAULT 0,
    applicable_advisory_sec_count_cache  INT         NOT NULL DEFAULT 0,
    template_id                          BIGINT,
    PRIMARY KEY (rh_account_id, system_id),
    FOREIGN KEY (rh_account_id, template_id) REFERENCES template (rh_account_id, id),
    FOREIGN KEY (rh_account_id, system_id) REFERENCES system_inventory (rh_account_id, id)
) PARTITION BY HASH (rh_account_id);

SELECT create_table_partitions('system_patch', 16,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
                                 TABLESPACE pg_default$$);

GRANT SELECT, UPDATE ON system_patch TO evaluator;
GRANT SELECT, INSERT, UPDATE ON system_patch TO listener;
GRANT SELECT, UPDATE (installable_advisory_count_cache,
              installable_advisory_enh_count_cache,
              installable_advisory_bug_count_cache,
              installable_advisory_sec_count_cache,
              applicable_advisory_count_cache,
              applicable_advisory_enh_count_cache,
              applicable_advisory_bug_count_cache,
              applicable_advisory_sec_count_cache,
              template_id) ON system_patch TO manager;
GRANT SELECT, UPDATE ON system_patch TO manager;
GRANT SELECT, UPDATE, DELETE ON system_patch to vmaas_sync; -- vmaas_sync performs system culling

-- system_platform
DROP TABLE IF EXISTS system_platform;
CREATE OR REPLACE VIEW system_platform AS SELECT
    si.id,
    si.inventory_id,
    si.rh_account_id,
    si.vmaas_json,
    si.json_checksum,
    si.last_updated,
    si.unchanged_since,
    sp.last_evaluation,
    sp.installable_advisory_count_cache,
    sp.installable_advisory_enh_count_cache,
    sp.installable_advisory_bug_count_cache,
    sp.installable_advisory_sec_count_cache,
    si.last_upload,
    si.stale_timestamp,
    si.stale_warning_timestamp,
    si.culled_timestamp,
    si.stale,
    si.display_name,
    sp.packages_installed,
    sp.packages_installable,
    si.reporter_id,
    sp.third_party,
    si.yum_updates,
    sp.applicable_advisory_count_cache,
    sp.applicable_advisory_enh_count_cache,
    sp.applicable_advisory_bug_count_cache,
    sp.applicable_advisory_sec_count_cache,
    si.satellite_managed,
    si.built_pkgcache,
    sp.packages_applicable,
    sp.template_id,
    si.yum_checksum,
    si.arch,
    si.bootc
FROM system_inventory si JOIN system_patch sp
    ON si.id = sp.system_id AND si.rh_account_id = sp.rh_account_id;

CREATE OR REPLACE FUNCTION system_platform_insert()
    RETURNS TRIGGER AS
$system_platform_insert$
DECLARE
    new_system_id BIGINT;
    created TIMESTAMPTZ := CURRENT_TIMESTAMP;
BEGIN
    INSERT INTO system_inventory (
        inventory_id,
        rh_account_id,
        vmaas_json,
        json_checksum,
        last_updated,
        unchanged_since,
        last_upload,
        stale_timestamp,
        stale_warning_timestamp,
        culled_timestamp,
        stale,
        display_name,
        reporter_id,
        yum_updates,
        satellite_managed,
        built_pkgcache,
        yum_checksum,
        arch,
        bootc,
        tags,
        created
    ) VALUES (
        NEW.inventory_id,
        NEW.rh_account_id,
        NEW.vmaas_json,
        NEW.json_checksum,
        NEW.last_updated,
        NEW.unchanged_since,
        NEW.last_upload,
        NEW.stale_timestamp,
        NEW.stale_warning_timestamp,
        NEW.culled_timestamp,
        COALESCE(NEW.stale, false),
        NEW.display_name,
        NEW.reporter_id,
        NEW.yum_updates,
        COALESCE(NEW.satellite_managed, false),
        COALESCE(NEW.built_pkgcache, false),
        NEW.yum_checksum,
        NEW.arch,
        COALESCE(NEW.bootc, false),
        '[]'::JSONB,
        created
    )
    RETURNING id INTO new_system_id;

    INSERT INTO system_patch (
        system_id,
        rh_account_id,
        last_evaluation,
        installable_advisory_count_cache,
        installable_advisory_enh_count_cache,
        installable_advisory_bug_count_cache,
        installable_advisory_sec_count_cache,
        packages_installed,
        packages_installable,
        packages_applicable,
        third_party,
        applicable_advisory_count_cache,
        applicable_advisory_enh_count_cache,
        applicable_advisory_bug_count_cache,
        applicable_advisory_sec_count_cache,
        template_id
    ) VALUES (
        new_system_id,
        NEW.rh_account_id,
        NEW.last_evaluation,
        COALESCE(NEW.installable_advisory_count_cache, 0),
        COALESCE(NEW.installable_advisory_enh_count_cache, 0),
        COALESCE(NEW.installable_advisory_bug_count_cache, 0),
        COALESCE(NEW.installable_advisory_sec_count_cache, 0),
        COALESCE(NEW.packages_installed, 0),
        COALESCE(NEW.packages_installable, 0),
        COALESCE(NEW.packages_applicable, 0),
        COALESCE(NEW.third_party, false),
        COALESCE(NEW.applicable_advisory_count_cache, 0),
        COALESCE(NEW.applicable_advisory_enh_count_cache, 0),
        COALESCE(NEW.applicable_advisory_bug_count_cache, 0),
        COALESCE(NEW.applicable_advisory_sec_count_cache, 0),
        NEW.template_id
    );

    NEW.id := new_system_id;
    RETURN NEW;
END;
$system_platform_insert$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION system_platform_update()
    RETURNS TRIGGER AS
$system_platform_update$
BEGIN
    UPDATE system_inventory SET
        inventory_id = NEW.inventory_id,
        vmaas_json = NEW.vmaas_json,
        json_checksum = NEW.json_checksum,
        last_updated = NEW.last_updated,
        unchanged_since = NEW.unchanged_since,
        last_upload = NEW.last_upload,
        stale_timestamp = NEW.stale_timestamp,
        stale_warning_timestamp = NEW.stale_warning_timestamp,
        culled_timestamp = NEW.culled_timestamp,
        stale = NEW.stale,
        display_name = NEW.display_name,
        reporter_id = NEW.reporter_id,
        yum_updates = NEW.yum_updates,
        satellite_managed = NEW.satellite_managed,
        built_pkgcache = NEW.built_pkgcache,
        yum_checksum = NEW.yum_checksum,
        arch = NEW.arch,
        bootc = NEW.bootc
    WHERE id = OLD.id AND rh_account_id = OLD.rh_account_id;

    UPDATE system_patch SET
        last_evaluation = NEW.last_evaluation,
        installable_advisory_count_cache = NEW.installable_advisory_count_cache,
        installable_advisory_enh_count_cache = NEW.installable_advisory_enh_count_cache,
        installable_advisory_bug_count_cache = NEW.installable_advisory_bug_count_cache,
        installable_advisory_sec_count_cache = NEW.installable_advisory_sec_count_cache,
        packages_installed = NEW.packages_installed,
        packages_installable = NEW.packages_installable,
        packages_applicable = NEW.packages_applicable,
        third_party = NEW.third_party,
        applicable_advisory_count_cache = NEW.applicable_advisory_count_cache,
        applicable_advisory_enh_count_cache = NEW.applicable_advisory_enh_count_cache,
        applicable_advisory_bug_count_cache = NEW.applicable_advisory_bug_count_cache,
        applicable_advisory_sec_count_cache = NEW.applicable_advisory_sec_count_cache,
        template_id = NEW.template_id
    WHERE system_id = OLD.id AND rh_account_id = OLD.rh_account_id;

    RETURN NEW;
END;
$system_platform_update$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION system_platform_delete()
    RETURNS TRIGGER AS
$system_platform_delete$
BEGIN
    DELETE FROM system_patch
     WHERE system_id = OLD.id AND rh_account_id = OLD.rh_account_id;

    DELETE FROM system_inventory
     WHERE id = OLD.id AND rh_account_id = OLD.rh_account_id;

    RETURN OLD;
END;
$system_platform_delete$
    LANGUAGE 'plpgsql';

CREATE TRIGGER system_platform_insert_trigger
    INSTEAD OF INSERT ON system_platform
    FOR EACH ROW
    EXECUTE FUNCTION system_platform_insert();

CREATE TRIGGER system_platform_update_trigger
    INSTEAD OF UPDATE ON system_platform
    FOR EACH ROW
    EXECUTE FUNCTION system_platform_update();

CREATE TRIGGER system_platform_delete_trigger
    INSTEAD OF DELETE ON system_platform
    FOR EACH ROW
    EXECUTE FUNCTION system_platform_delete();

GRANT SELECT, INSERT, UPDATE, DELETE ON system_platform TO listener;
-- evaluator needs to update last_evaluation
GRANT UPDATE ON system_platform TO evaluator;
-- manager needs to update cache and delete systems
GRANT SELECT, UPDATE, DELETE ON system_platform TO manager;
-- VMaaS sync needs to be able to perform system culling tasks
GRANT SELECT, UPDATE, DELETE ON system_platform to vmaas_sync;

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
GRANT SELECT, UPDATE ON repo TO vmaas_sync;
GRANT SELECT, INSERT, UPDATE, DELETE ON deleted_system TO vmaas_sync;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'repack') THEN
        GRANT USAGE ON SCHEMA repack TO vmaas_sync;
    END IF;
END
$$;


CREATE SCHEMA IF NOT EXISTS inventory;

-- The admin ROLE that allows the inventory schema to be managed
DO $$
BEGIN
  CREATE ROLE cyndi_admin;
  EXCEPTION WHEN DUPLICATE_OBJECT THEN
    RAISE NOTICE 'cyndi_admin already exists';
END
$$;
GRANT ALL PRIVILEGES ON SCHEMA inventory TO cyndi_admin;

-- The reader ROLE that provides SELECT access to the inventory.hosts view
DO $$
BEGIN
  CREATE ROLE cyndi_reader;
  EXCEPTION WHEN DUPLICATE_OBJECT THEN
    RAISE NOTICE 'cyndi_reader already exists';
END
$$;
GRANT USAGE ON SCHEMA inventory TO cyndi_reader;

-- The application user is granted the reader role only to eliminate any interference with Cyndi
GRANT cyndi_reader to listener;
GRANT cyndi_reader to evaluator;
GRANT cyndi_reader to manager;
GRANT cyndi_reader TO vmaas_sync;

GRANT cyndi_admin to cyndi;
