\timing on

-- constants to drive number of items generateg
create table if not exists _const (
    key text primary key,
    val int
);

insert into _const values           -- counts in prod 2022/02
    ('accounts',   50),             --  50k     -- number of rh_accounts
    ('systems',    7500),           -- 750k     -- number of systems(_platform)
    ('advisories', 320),            --  50k     -- number of advisory_metadata
    ('repos',      350),            --  55k     -- number of repos
    ('package_names', 300),         --  58k     -- number of package_name
    ('packages', 4500),             -- 1650k    -- number of package
    ('adv_per_system', 10),         -- 100      (71M system_advisories)
    ('repo_per_system', 10),        --   8      (6.1M system_repo)
    ('packages_per_system', 1000),  -- 780      (580M system_packages)
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
-- Time: 7254938.008 ms (02:00:54.938)  for 75M system_advisories (RDS)
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
            raise notice 'created % package names', cnt;
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

-- generate system_packages
-- duration: 493s (8:13) for 9000000 system_packages
do $$
  declare
    cnt int := 0;
    wanted int;
    pkg_per_system int;
    progress int;
    systems int;
    pkgs int;
    pkg_names int;
    -- patched_pct float := 0.80;
    update_data jsonb := '[{"evra": "5.10.13-200.fc31.x86_64", "advisory": "RH-100"}]'::jsonb;
    rnd float;
    rnd2 float;
    row record;
  begin
    select val into pkg_per_system from _const where key = 'packages_per_system';
    select val * pkg_per_system into wanted from _const where key = 'systems';
    select val into progress from _const where key = 'progress_pct';
    select count(*) into systems from system_platform;
    select count(*) into pkgs from package;
    select count(*) into pkg_names from package_name;
    <<systems>>
    for row in select rh_account_id, id from system_platform
    loop
      -- assign random 0.8-1.2*pkg_per_system packages to system
      rnd := (0.8 + random() * 0.4) * pkg_per_system;
      for i in 0..rnd loop
          rnd2 := random();
          insert into system_package
              (rh_account_id, system_id, package_id, update_data, name_id)
          values
              (row.rh_account_id, row.id, trunc(pkgs*rnd2)+1, update_data, trunc(pkg_names*rnd2)+1)
          on conflict do nothing;
          if mod(cnt, (wanted*progress/100)::int) = 0 then
              raise notice 'created % system_packages', cnt;
          end if;
          cnt := cnt + 1;
          exit systems when cnt > wanted;
      end loop;
    end loop;  -- <<systems>>
    raise notice 'created % system_packages', wanted;
  end;
$$
;

-- 58M rows system_packages, contains cca 5k rows with mod(advisory_id,300)=0
-- table size 11GB, total size 20GB
-- delete from package where id in 
--       (select id from package where mod(advisory_id,300)=0 and NOT EXISTS (SELECT 1 FROM system_package sp WHERE package.id = sp.package_id) limit 1000);
-- CREATE INDEX IF NOT EXISTS system_package_package_id_idx on system_package (package_id);
--  Time: 24545.275 ms (00:24.545)
--  index size 1.2GB
-- with index
--  Delete on package  (cost=538049.27..546012.32 rows=1000 width=34)
--   ->  Nested Loop  (cost=538049.27..546012.32 rows=1000 width=34)
--         ->  HashAggregate  (cost=538048.84..538058.84 rows=1000 width=32)
--               Group Key: "ANY_subquery".id
--               ->  Subquery Scan on "ANY_subquery"  (cost=0.42..538046.34 rows=1000 width=32)
--                     ->  Limit  (cost=0.42..538036.34 rows=1000 width=4)
--                           ->  Nested Loop Anti Join  (cost=0.42..2219398.58 rows=4125 width=4)
--                                 ->  Seq Scan on package package_1  (cost=0.00..46738.00 rows=8250 width=4)
--                                       Filter: (mod(advisory_id, 300) = 0)
--                                 ->  Append  (cost=0.42..510.77 rows=128 width=4)
--                                       ->  Index Only Scan using system_package_0_package_id_idx on system_package_0 sp  (cost=0.42..4.02 rows=1 width=4)
--                                             Index Cond: (package_id = package_1.id)
--                                        ...
--  Time: 320.596 ms
-- without index
--  Delete on package  (cost=3569900.96..3577864.01 rows=1000 width=34)
--   ->  Nested Loop  (cost=3569900.96..3577864.01 rows=1000 width=34)
--         ->  HashAggregate  (cost=3569900.53..3569910.53 rows=1000 width=32)
--               Group Key: "ANY_subquery".id
--               ->  Subquery Scan on "ANY_subquery"  (cost=3299748.63..3569898.03 rows=1000 width=32)
--                     ->  Limit  (cost=3299748.63..3569888.03 rows=1000 width=4)
--                           ->  Hash Anti Join  (cost=3299748.63..4414073.67 rows=4125 width=4)
--                                 Hash Cond: (package_1.id = sp.package_id)
--                                 ->  Seq Scan on package package_1  (cost=0.00..46738.00 rows=8250 width=4)
--                                       Filter: (mod(advisory_id, 300) = 0)
--                                 ->  Hash  (cost=2340004.62..2340004.62 rows=58498641 width=4)
--                                       ->  Append  (cost=0.00..2340004.62 rows=58498641 width=4)
--                                             ->  Seq Scan on system_package_0 sp  (cost=0.00..16412.07 rows=468907 width=4)
--                                             ...
--  Time: 24448.174 ms (00:24.448)


-- 233M rows 44GB table, 78GB total
--  Delete on package  (cost=13636342.85..13644305.90 rows=1000 width=34) Time: 163558.425 ms (02:43.558)
-- CREATE INDEX Time: 182975.788 ms (03:02.976)
--  Delete on package  (cost=1309087.93..1317050.98 rows=1000 width=34) Time: 1757.323 ms (00:01.757)

-- 584M rows, 111GB table, 216GB total
--  Delete on package  (cost=33763897.37..33771860.41 rows=1000 width=34) Time: 1498463.270 ms (24:58.463)
-- CREATE INDEX Time: 624372.198 ms (10:24.372)
--  Delete on package  (cost=1199882.97..1207846.02 rows=1000 width=34) Time: 3678.649 ms (00:03.679)

-- delete from system_package sp where sp.package_id in (select id from package p where mod(p.advisory_id,301)=0 limit 1000);
-- DELETE 355104 Time: 248567.674 ms (04:08.568)
-- delete from package where id in (select id from package where mod(advisory_id,301)=0 and NOT EXISTS (SELECT 1 FROM system_package sp WHERE package.id = sp.package_id) limit 1000);
-- DELETE 1000 Time: 17861.151 ms (00:17.861)
