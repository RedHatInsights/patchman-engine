\timing on

-- constants to drive number of items generateg
create table if not exists _const (
    key text primary key,
    val int
);

insert into _const values   
    ('accounts',   50),     -- 5000     -- number of rh_accounts
    ('systems',    7500),   -- 750000   -- number of systems(_platform)
    ('advisories', 320),    -- 32000    -- number of advisory_metadata
    ('repos',      350),    -- 35000    -- number of repos
    ('adv_per_system', 10),  -- ??       -- should be system_advisories/systems
    ('repo_per_system', 10),  -- ??       -- should be system_repo/systems
                            -- ^ counts in prod
    ('package_names', 300), -- 40000    -- number of package_name
    ('packages', 4500),     -- 700000   -- number of package
    ('progress_pct', 10)   -- print progress message on every X% reached
    on conflict do nothing;

-- prepare some pseudorandom vmaas jsons
create table if not exists _json (
    id int primary key,
    data text,
    hash text
);
insert into _json values 
    (1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}'),
    (2, '{ "package_list": [ "libsmbclient-4.6.2-12.el7_4.x86_64", "dconf-0.26.0-2.el7.x86_64", "texlive-mdwtools-doc-svn15878.1.05.4-38.el7.noarch", "python34-pyroute2-0.4.13-1.el7.noarch", "python-backports-ssl_match_hostname-3.4.0.2-4.el7.noarch", "ghc-aeson-0.6.2.1-3.el7.x86_64"]}'),
    (3, '{ "repository_list": [ "rhel-7-server-rpms" ], "releasever": "7Server", "basearch": "x86_64", "package_list": [ "libsmbclient-4.6.2-12.el7_4.x86_64", "dconf-0.26.0-2.el7.x86_64"]}')
    on conflict do nothing;
update _json set hash = encode(sha256(data::bytea), 'hex');


-- !!! BIG WARNING !!!
--  this script will remove existing data from (nearly) all tables
truncate table rh_account cascade;
truncate table advisory_metadata cascade;

-- generate rh_accounts
-- duration: 250ms / 5000 accounts (on RDS)
alter sequence rh_account_id_seq restart with 1;
do $$
  declare
    cnt int :=0;
    wanted int;
    id int;
  begin
    --select count(*) into cnt from rh_account;
    select val into wanted from _const where key = 'accounts';
    while cnt < wanted loop
        id := nextval('rh_account_id_seq');
        insert into rh_account (id, name)
        values (id, 'RHACCOUNT-' || id );
        cnt := cnt + 1;
    end loop;
    raise notice 'created % rh_accounts', wanted;
  end;
$$
;


-- generate systems
-- duration: 55s / 750k systems (on RDS)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
alter sequence system_platform_id_seq restart with 1;
do $$
  declare
    cnt int := 0;
    wanted int;
    progress int;
    gen_uuid uuid;
    rh_accounts int;
    rnd float;
    json_data text[];
    json_hash text[];
    json_rnd int;
    rnd_date1 timestamp with time zone;
    rnd_date2 timestamp with time zone;
  begin
    --select count(*) into cnt from system_platform;
    select val into wanted from _const where key = 'systems';
    select val into progress from _const where key = 'progress_pct';
    select count(*) into rh_accounts from rh_account;
    json_data := array(select data from _json order by id);
    json_hash := array(select hash from _json order by id);
    while cnt < wanted loop
        gen_uuid := uuid_generate_v4();
        rnd := random();
        rnd_date1 := now() - make_interval(days => (rnd*30)::int);
        rnd_date2 := rnd_date1 + make_interval(days => (rnd*10)::int);
        insert into system_platform
            (inventory_id, display_name, rh_account_id, vmaas_json, json_checksum, last_updated, unchanged_since, last_upload, packages_installed, packages_updatable)
        values
            (gen_uuid, gen_uuid, trunc(rnd*rh_accounts)+1, json_data[trunc(rnd*3)], json_hash[trunc(rnd*3)], rnd_date2, rnd_date1, rnd_date2, trunc(rnd*1000), trunc(rnd*50))
        on conflict do nothing;
        if mod(cnt, (wanted*progress/100)::int) = 0 then
            raise notice 'created % system_platforms', cnt;
        end if;
        cnt := cnt + 1;
    end loop;
    raise notice 'created % system_platforms', wanted;
  end;
$$
;

-- generate advisory_metadata
-- duration: 2s / 32k advisories (on RDS)
alter sequence advisory_metadata_id_seq restart with 1;
do $$
  declare
    cnt int := 0;
    wanted int;
    adv_type int;
    sev int;
    id int;
    rnd float;
    rnd_date1 timestamp with time zone;
    rnd_date2 timestamp with time zone;
  begin
    select val into wanted from _const where key = 'advisories';
    select count(*)-1 into adv_type from advisory_type;
    select count(*) into sev from advisory_severity;
    while cnt < wanted loop
        id := nextval('advisory_metadata_id_seq');
        rnd := random();
        rnd_date1 := now() - make_interval(days => (rnd*365)::int);
        rnd_date2 := rnd_date1 + make_interval(days => (rnd*100)::int);
        insert into advisory_metadata
            (id, name, description, synopsis, summary, solution, advisory_type_id, public_date, modified_date, url, severity_id, cve_list)
        values
            (id, 'ADV-2020-' || id, 'Decription of advisory ' || id, 'Synopsis of advisory ' || id,
                'Summary of advisory ' || id, 'Solution of advisory ' || id, trunc(rnd*adv_type)+1,
                rnd_date1, rnd_date2, 'http://errata.example.com/errata/' || id, trunc(rnd*sev)+1, NULL);
        cnt := cnt + 1;
    end loop;
    raise notice 'created % advisory_metadata', wanted;
  end;
$$
;

-- generate system_advisories
-- duration: 325s (05:25) / 7.5M system_advisories (a.k.a. 750k systems with 10 adv in avg) (on RDS) 
do $$
  declare
    cnt int := 0;
    wanted int;
    adv_per_system int;
    progress int;
    systems int;
    advs int;
    stat int;
    patched_pct float := 0.80;
    rnd float;
    rnd2 float;
    rnd_date1 timestamp with time zone;
    rnd_date2 timestamp with time zone;
    row record;
  begin
    select val into adv_per_system from _const where key = 'adv_per_system';
    select val * adv_per_system into wanted from _const where key = 'systems';
    select val into progress from _const where key = 'progress_pct';
    select count(*) into systems from system_platform;
    select count(*) into advs from advisory_metadata;
    select count(*) into stat from status;
    <<systems>>
    for row in select rh_account_id, id from system_platform
    loop
      -- assign random 0-2*adv_per_system advisories to system
      rnd := random() * 2 * adv_per_system;
      --while cnt < wanted loop
      for i in 0..rnd loop
          rnd2 := random();
          rnd_date1 := now() - make_interval(days => (rnd*365)::int);
          rnd_date2 := rnd_date1 + make_interval(days => (rnd*100)::int);
          insert into system_advisories
              (rh_account_id, system_id, advisory_id, first_reported, when_patched, status_id)
          values
              (row.rh_account_id, row.id, trunc(advs*rnd2)+1, rnd_date1, case when random() < patched_pct then rnd_date2 else NULL end, trunc(stat*rnd2))
          on conflict do nothing;
          if mod(cnt, (wanted*progress/100)::int) = 0 then
              raise notice 'created % system_advisories', cnt;
          end if;
          cnt := cnt + 1;
          exit systems when cnt > wanted;
      end loop;
    end loop;  -- <<systems>>
    raise notice 'created % system_advisories', wanted;
  end;
$$
;

-- generate repos
-- duration: 2s / 35k advisories (on RDS)
alter sequence repo_id_seq restart with 1;
do $$
  declare
    cnt int :=0;
    wanted int;
    id int;
  begin
    select val into wanted from _const where key = 'repos';
    while cnt < wanted loop
        id := nextval('repo_id_seq');
        insert into repo (id, name)
               values (id, 'REPO-' || id )
               on conflict do nothing;
        cnt := cnt + 1;
    end loop;
    raise notice 'created % repos', wanted;
  end;
$$
;

-- generate system_repo
-- duration: 325s (05:25) / 7.5M system_repo (a.k.a. 750k systems with 10 repo in avg) (on RDS) 
do $$
  declare
    cnt int := 0;
    wanted int;
    repo_per_system int;
    progress int;
    systems int;
    repos int;
    rnd float;
    rnd2 float;
    row record;
  begin
    select val into repo_per_system from _const where key = 'repo_per_system';
    select val * repo_per_system into wanted from _const where key = 'systems';
    select val into progress from _const where key = 'progress_pct';
    select count(*) into systems from system_platform;
    select count(*) into repos from repo;
    <<systems>>
    for row in select rh_account_id, id from system_platform
    loop
      -- assign random 0-2*repo_per_system repos per system
      rnd := random() * 2 * repo_per_system;
      for i in 0..rnd loop
          rnd2 := random();
          insert into system_repo
              (rh_account_id, system_id, repo_id)
          values
              (row.rh_account_id, row.id, trunc(repos*rnd2)+1)
          on conflict do nothing;
          if mod(cnt, (wanted*progress/100)::int) = 0 then
              raise notice 'created % system_repos', cnt;
          end if;
          cnt := cnt + 1;
          exit systems when cnt > wanted;
      end loop;
    end loop;  -- <<systems>>
    raise notice 'created % system_repos', wanted;
  end;
$$
;

-- generate package_name
alter sequence package_name_id_seq restart with 1;
do $$
  declare
    cnt int :=0;
    wanted int; id int; progress int;
  begin
    select val into wanted from _const where key = 'package_names';
    select val into progress from _const where key = 'progress_pct';
    while cnt < wanted loop
        id := nextval('package_name_id_seq');
        insert into package_name(id, name)
               values (id, 'package' || id )
               on conflict do nothing;
        cnt := cnt + 1;
        if mod(cnt, (wanted*progress/100)::int) = 0 then
            raise notice 'created % packages', cnt;
        end if;
    end loop;
    raise notice 'created package names %', wanted;
  end;
$$
;

-- add fake strings item to use as summary and description in packages
insert into strings(id, value) values ('0', 'testing string value')
on conflict do nothing;

-- generate package
alter sequence package_id_seq restart with 1;
do $$
  declare
    cnt int := 0;
    wanted int; n_names int; n_advisories int; id int; name_id int; advisory_id int; progress int;
  begin
    select val into wanted from _const where key = 'packages';
    select val into progress from _const where key = 'progress_pct';
    select count(*) into n_names from package_name;
    select count(*) into n_advisories from advisory_metadata;
    while cnt < wanted loop
        id := nextval('package_id_seq');
        name_id := id % n_names + 1;
        advisory_id := id % n_advisories + 1;
        insert into package(id, name_id, evra, description_hash, summary_hash, advisory_id)
               values (id, name_id, id || '.' || id || '-1.el8.x86_64', '0', '0', advisory_id)
               on conflict do nothing;
        cnt := cnt + 1;
        if mod(cnt, (wanted*progress/100)::int) = 0 then
            raise notice 'created % packages', cnt;
        end if;
    end loop;
    raise notice 'created packages %', wanted;
  end;
$$
;
