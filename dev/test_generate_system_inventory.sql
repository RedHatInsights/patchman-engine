-- Load-test generator for system_inventory (+ required rh_account, system_patch).
-- Faster subset of dev/test_generate_data.sql (no advisories, repos, packages, etc.).
\timing on
\set ON_ERROR_STOP on

CREATE TABLE IF NOT EXISTS _const (
    key TEXT PRIMARY KEY,
    val INT
);

TRUNCATE _const;
INSERT INTO _const VALUES
    ('accounts', 50),       -- rh_account rows
    ('systems', 7500),      -- system_inventory + system_patch rows
    ('progress_pct', 10);   -- progress NOTICE every N%

-- Minimal vmaas_json samples (same as test_generate_data.sql)
CREATE TABLE IF NOT EXISTS _json (
    id   INT PRIMARY KEY,
    data TEXT,
    hash TEXT
);
INSERT INTO _json VALUES
    (1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}'),
    (2, '{ "package_list": [ "libsmbclient-4.6.2-12.el7_4.x86_64", "dconf-0.26.0-2.el7.x86_64"]}'),
    (3, '{ "repository_list": [ "rhel-7-server-rpms" ], "releasever": "7Server", "basearch": "x86_64", "package_list": [ "libsmbclient-4.6.2-12.el7_4.x86_64"]}')
ON CONFLICT DO NOTHING;
UPDATE _json SET hash = encode(sha256(data::bytea), 'hex');

-- Wipes accounts and all dependent host data (inventory, patch, advisories links, packages, …)
TRUNCATE rh_account CASCADE;

ALTER SEQUENCE rh_account_id_seq RESTART WITH 1;
DO $$
DECLARE
    cnt    INT := 0;
    wanted INT;
    id     INT;
BEGIN
    SELECT val INTO wanted FROM _const WHERE key = 'accounts';
    WHILE cnt < wanted LOOP
        id := nextval('rh_account_id_seq');
        INSERT INTO rh_account (id, org_id) VALUES (id, 'RHACCOUNT-' || id);
        cnt := cnt + 1;
    END LOOP;
    RAISE NOTICE 'created % rh_accounts', wanted;
END;
$$;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
ALTER SEQUENCE system_inventory_id_seq RESTART WITH 1;

DO $$
DECLARE
    cnt           INT := 0;
    wanted        INT;
    progress      INT;
    gen_uuid      UUID;
    rh_accounts   INT;
    rnd           FLOAT;
    json_data     TEXT[];
    json_hash     TEXT[];
    rnd_date1     TIMESTAMPTZ;
    rnd_date2     TIMESTAMPTZ;
    acc_id        INT;
    new_id        BIGINT;
    ji            INT;
    workspace_ids UUID[];
BEGIN
    SELECT val INTO wanted FROM _const WHERE key = 'systems';
    SELECT val INTO progress FROM _const WHERE key = 'progress_pct';
    SELECT count(*) INTO rh_accounts FROM rh_account;
    json_data := array(SELECT data FROM _json ORDER BY id);
    json_hash := array(SELECT hash FROM _json ORDER BY id);
    workspace_ids := array(SELECT uuid_generate_v4() FROM generate_series(1, 3));

    WHILE cnt < wanted LOOP
        gen_uuid := uuid_generate_v4();
        rnd := random();
        rnd_date1 := now() - make_interval(days => (rnd * 30)::INT);
        rnd_date2 := rnd_date1 + make_interval(days => (rnd * 10)::INT);
        acc_id := trunc(rnd * rh_accounts) + 1;
        ji := trunc(rnd * 3) + 1;

        INSERT INTO system_inventory
            (inventory_id, display_name, rh_account_id, vmaas_json, json_checksum,
             last_updated, last_upload, arch, tags, created, os_name, os_major, rhsm_version,
             workspaces, workspace_id, workspace_name)
        VALUES
            (gen_uuid, gen_uuid::text, acc_id, json_data[ji], json_hash[ji],
             rnd_date2, rnd_date2, 'x86_64', '[]'::jsonb, rnd_date1, 'RHEL', 8, '8.0',
             jsonb_build_array(jsonb_build_object(
                 'id', workspace_ids[cnt % 3 + 1]::text,
                 'name', workspace_ids[cnt % 3 + 1]::text)),
             workspace_ids[cnt % 3 + 1], workspace_ids[cnt % 3 + 1]::text)
        RETURNING id INTO new_id;

        INSERT INTO system_patch
            (rh_account_id, system_id, last_evaluation,
             packages_installed, packages_installable, packages_applicable)
        VALUES
            (acc_id, new_id, rnd_date2, trunc(rnd * 1000), trunc(rnd * 50), trunc(rnd * 50));

        IF mod(cnt, greatest(1, ceil((wanted::numeric * progress) / 100.0)::int)) = 0 THEN
            RAISE NOTICE 'created % systems (inventory + patch)', cnt;
        END IF;
        cnt := cnt + 1;
    END LOOP;
    RAISE NOTICE 'created % systems (inventory + patch)', wanted;
END;
$$;

SELECT 'rh_account' AS tbl, count(*) FROM rh_account
UNION ALL
SELECT 'system_inventory', count(*) FROM system_inventory
UNION ALL
SELECT 'system_patch', count(*) FROM system_patch;

SELECT parent.relname AS parent,
       child.relname  AS child,
       pg_size_pretty(pg_relation_size(child.oid)) AS size
FROM pg_inherits
         JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
         JOIN pg_class child ON pg_inherits.inhrelid = child.oid
WHERE parent.relname IN ('system_inventory', 'system_patch')
ORDER BY 1, 2;
