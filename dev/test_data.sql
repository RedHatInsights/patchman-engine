DELETE FROM system_advisories;
DELETE FROM system_repo;
DELETE FROM system_package;
DELETE FROM system_platform;
DELETE FROM deleted_system;
DELETE FROM repo;
DELETE FROM timestamp_kv;
DELETE FROM advisory_account_data;
DELETE FROM package;
DELETE FROM package_name;
DELETE FROM advisory_metadata;
DELETE FROM baseline;
DELETE FROM rh_account;
DELETE FROM strings;

INSERT INTO rh_account (id, name) VALUES
(1, 'acc-1'), (2, 'acc-2'), (3, 'acc-3'), (4, 'acc-4');

INSERT INTO baseline (id, rh_account_id, name, config) VALUES
(1, 1, 'baseline_1-1', '{"to_time": "2010-09-22T00:00:00+00:00"}'),
(2, 1, 'baseline_1-2', '{"to_time": "2021-01-01T00:00:00+00:00"}'),
(3, 1, 'baseline_1-3', '{"to_time": "2000-01-01T00:00:00+00:00"}');

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id, reporter_id, vmaas_json, json_checksum, last_evaluation, last_upload, packages_installed, packages_updatable, third_party, baseline_id, baseline_uptodate) VALUES
(1, '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001', 1, 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2020-09-22 12:00:00-04',0,0, true, 1, true),
(2, '00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000002', 1, 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04',0,0, false, 1, true),
(3, '00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000003', 1, 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0, false, 2, false),
(4, '00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000004', 1, 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0, false, NULL, NULL),
(5, '00000000-0000-0000-0000-000000000005','00000000-0000-0000-0000-000000000005', 1, 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0, false, NULL, NULL),
(6, '00000000-0000-0000-0000-000000000006','00000000-0000-0000-0000-000000000006', 1, 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0, false, NULL, NULL);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_updated, unchanged_since, last_upload, packages_installed, packages_updatable) VALUES
(7, '00000000-0000-0000-0000-000000000007','00000000-0000-0000-0000-000000000007', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-10-04 14:13:12-04', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload, packages_installed, packages_updatable) VALUES
(8, '00000000-0000-0000-0000-000000000008','00000000-0000-0000-0000-000000000008', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0),
(9, '00000000-0000-0000-0000-000000000009','00000000-0000-0000-0000-000000000009', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(10, '00000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000010', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(11, '00000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000011', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(12, '00000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000012', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',2,2);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload, packages_installed, packages_updatable) VALUES
(13, '00000000-0000-0000-0000-000000000013','00000000-0000-0000-0000-000000000013', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04', 1,0),
(14, '00000000-0000-0000-0000-000000000014','00000000-0000-0000-0000-000000000014', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04', 0,0);

INSERT INTO advisory_metadata (id, name, description, synopsis, summary, solution, advisory_type_id,
                               public_date, modified_date, url, severity_id, cve_list, release_versions) VALUES
(1, 'RH-1', 'adv-1-des', 'adv-1-syn', 'adv-1-sum', 'adv-1-sol', 1, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url1', NULL, NULL, '["7.0","7Server"]'),
(2, 'RH-2', 'adv-2-des', 'adv-2-syn', 'adv-2-sum', 'adv-2-sol', 2, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url2', NULL, NULL, NULL),
(3, 'RH-3', 'adv-3-des', 'adv-3-syn', 'adv-3-sum', 'adv-3-sol', 3, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url3', 2, '["CVE-1","CVE-2"]', NULL),
(4, 'RH-4', 'adv-4-des', 'adv-4-syn', 'adv-4-sum', 'adv-4-sol', 1, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url4', NULL, NULL, '["8.0","8.1"]'),
(5, 'RH-5', 'adv-5-des', 'adv-5-syn', 'adv-5-sum', 'adv-5-sol', 2, '2016-09-22 12:00:00-05', '2017-09-22 12:00:00-05', 'url5', NULL, NULL, '["8.0"]'),
(6, 'RH-6', 'adv-6-des', 'adv-6-syn', 'adv-6-sum', 'adv-6-sol', 3, '2016-09-22 12:00:00-06', '2017-09-22 12:00:00-06', 'url6', 4, '["CVE-2","CVE-3"]', NULL),
(7, 'RH-7', 'adv-7-des', 'adv-7-syn', 'adv-7-sum', 'adv-7-sol', 1, '2017-09-22 12:00:00-07', '2017-09-22 12:00:00-07', 'url7', NULL, NULL, NULL),
(8, 'RH-8', 'adv-8-des', 'adv-8-syn', 'adv-8-sum', 'adv-8-sol', 2, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url8', NULL, NULL, NULL),
(9, 'RH-9', 'adv-9-des', 'adv-9-syn', 'adv-9-sum', 'adv-9-sol', 3, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url9', NULL, '["CVE-4"]', '["8.2","8.4"]'),
(10, 'UNSPEC-10', 'adv-10-des', 'adv-10-syn', 'adv-10-sum', 'adv-10-sol', 4, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url10', NULL, NULL, NULL),
(11, 'UNSPEC-11', 'adv-11-des', 'adv-11-syn', 'adv-11-sum', 'adv-11-sol', 4, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url11', NULL, NULL, NULL),
(12, 'CUSTOM-12', 'adv-12-des', 'adv-12-syn', 'adv-12-sum', 'adv-12-sol', 0, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url12', NULL, NULL, NULL),
(13, 'CUSTOM-13', 'adv-13-des', 'adv-13-syn', 'adv-13-sum', 'adv-13-sol', 0, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url13', NULL, NULL, NULL);

UPDATE advisory_metadata SET package_data = '{"firefox": "77.0.1-1.fc31.x86_64"}' WHERE name = 'RH-9';

INSERT INTO system_advisories (rh_account_id, system_id, advisory_id, first_reported, when_patched, status_id) VALUES
(1, 1, 1, '2016-09-22 12:00:00-04', NULL, 0),
(1, 1, 2, '2016-09-22 12:00:00-04', NULL, 1),
(1, 1, 3, '2016-09-22 12:00:00-04', NULL, 0),
(1, 1, 4, '2016-09-22 12:00:00-04', NULL, 1),
(1, 1, 5, '2016-09-22 12:00:00-04', NULL, 2),
(1, 1, 6, '2016-09-22 12:00:00-04', NULL, 0),
(1, 1, 7, '2016-09-22 12:00:00-04', NULL, 1),
(1, 1, 8, '2016-09-22 12:00:00-04', NULL, 0),
(1, 1, 9, '2016-09-22 12:00:00-04', '2016-09-23 12:00:00-04', 0), -- some "patched" items to test correct filtering
(1, 2, 1, '2016-09-22 12:00:00-04', NULL, 1),
(1, 3, 1, '2016-09-22 12:00:00-04', NULL, 2),
(1, 4, 1, '2016-09-22 12:00:00-04', NULL, 0),
(1, 5, 1, '2016-09-22 12:00:00-04', NULL, 1),
(1, 6, 1, '2016-09-22 12:00:00-04', NULL, 0),
(1, 7, 1, '2016-09-22 12:00:00-04', '2016-09-23 12:00:00-04', 1),
(1, 8, 1, '2016-09-22 12:00:00-04', '2016-09-23 12:00:00-04', 2),
(1, 8, 10, '2016-09-22 12:00:00-04', NULL, 0),
(1, 8, 11, '2016-09-22 12:00:00-04', NULL, 0),
(1, 8, 12, '2016-09-22 12:00:00-04', NULL, 0),
(1, 8, 13, '2016-09-22 12:00:00-04', NULL, 0),
(2, 9, 1, '2016-09-22 12:00:00-04', '2016-09-23 12:00:00-04', 0),
(2, 10, 1, '2016-09-22 12:00:00-04', NULL, 1),
(2, 11, 1, '2016-09-22 12:00:00-04', NULL, 0);

INSERT INTO repo (id, name, third_party) VALUES
(1, 'repo1', false),
(2, 'repo2', false),
(3, 'repo3', false),
-- repo4 is not in platform mock for a purpose
(4, 'repo4', true);

INSERT INTO system_repo (rh_account_id, system_id, repo_id) VALUES
(1, 2, 1),
(1, 3, 1),
(1, 2, 2);


INSERT INTO package_name(id,name) VALUES
(101, 'kernel'),
(102, 'firefox'),
(103, 'bash'),
(104, 'curl'),
(105, 'tar'),
(106, 'systemd'),
(107, 'sed'),
(108, 'grep'),
(109, 'which'),
(110, 'passwd');

INSERT INTO strings(id, value) VALUES
('1', 'The Linux kernel'), -- kernel summary
('11', 'The kernel meta package'), -- kernel description
('2', 'Mozilla Firefox Web browser'), -- firefox summary
('22', 'Mozilla Firefox is an open-source web browser...'), -- firefox description
('3', 'The GNU Bourne Again shell'), -- bash summary
('33', 'The GNU Bourne Again shell (Bash) is a shell...'), -- bash description
('4', 'A utility for getting files from remote servers...'), -- curl summary
('44', 'curl is a command line tool for transferring data...'), -- curl description
('5', 'A GNU file archiving program'), -- tar summary
('55', 'The GNU tar program saves many files together in one archive...'), -- tar description
('6', 'System and Service Manager'), -- systemd summary
('66', 'systemd is a system and service manager that runs as PID 1...'), -- systemd description
('7', 'A GNU stream text editor'), -- sed summary
('77', 'The sed (Stream EDitor) editor is a stream or batch...'), -- sed description
('8', 'Pattern matching utilities'), -- grep summary
('88', 'The GNU versions of commonly used grep utilities...'), -- grep description
('9', 'Displays where a particular program in your path is located'), -- which summary
('99', 'The which command shows the full pathname of a specific program...'), -- which description
('10', 'An utility for setting or changing passwords using PAM'), -- passwd summary
('1010', 'This package contains a system utility (passwd) which sets...'); -- passwd description

INSERT INTO package(id, name_id, evra, description_hash, summary_hash, advisory_id) VALUES
(1, 101, '5.6.13-200.fc31.x86_64', '11', '1', 1), -- kernel
(2, 102, '76.0.1-1.fc31.x86_64', '22', '2', 1), -- firefox
(3, 103, '4.4.19-8.el8_0.x86_64', '33', '3', 3), -- bas
(4, 104, '7.61.1-8.el8.x86_64', '44', '4', 4), -- curl
(5, 105, '1.30-4.el8.x86_64', '55', '5', 5), -- tar
(6, 106, '239-13.el8_0.5.x86_64', '66', '6', 6), -- systemd
(7, 107, '4.5-1.el8.x86_64', '77', '7', 7), -- sed
(8, 108, '3.1-6.el8.x86_64', '88', '8', 8), -- grep
(9, 109, '2.21-10.el8.x86_64', '99', '9', 9), -- which
(10, 110, '0.80-2.el8.x86_64', '1010', '10', 9), -- passwd
(11, 101, '5.6.13-201.fc31.x86_64', '11', '1', 7), -- kernel
(12, 102, '76.0.1-2.fc31.x86_64', '22', '2', null); -- firefox

INSERT INTO system_package (rh_account_id, system_id, package_id, name_id, update_data) VALUES
(3, 12, 1, 101, '[{"evra": "5.10.13-200.fc31.x86_64", "advisory": "RH-100"}]'),
(3, 12, 2, 102, '[{"evra": "77.0.1-1.fc31.x86_64", "advisory": "RH-1"}, {"evra": "76.0.1-1.fc31.x86_64", "advisory": "RH-2"}]'),
(3, 13, 1, 101, null),
(3, 13, 2, 102, '[{"evra": "77.0.1-1.fc31.x86_64", "advisory": "RH-1"}, {"evra": "76.0.1-1.fc31.x86_64", "advisory": "RH-2"}]'),
(3, 13, 3, 103, null),
(3, 13, 4, 104, null);

INSERT INTO timestamp_kv (name, value) VALUES
('last_eval_repo_based', '2018-04-05T01:23:45+02:00');

SELECT refresh_all_cached_counts();
SELECT refresh_latest_packages_view();

ALTER TABLE advisory_metadata ALTER COLUMN id RESTART WITH 100;
ALTER TABLE system_platform ALTER COLUMN id RESTART WITH 100;
ALTER TABLE rh_account ALTER COLUMN id RESTART WITH 100;
ALTER TABLE repo ALTER COLUMN id RESTART WITH 100;
ALTER TABLE package ALTER COLUMN id RESTART WITH 100;
ALTER TABLE baseline ALTER COLUMN id RESTART WITH 100;

-- Create "inventory.hosts" for testing purposes. In deployment it's created by remote Cyndi service.

CREATE TABLE IF NOT EXISTS inventory.hosts_v1_0 (
    id uuid NOT NULL,
    insights_id uuid,
    account character varying(10) NOT NULL,
    display_name character varying(200) NOT NULL,
    tags jsonb NOT NULL,
    updated timestamp with time zone NOT NULL,
    created timestamp with time zone NOT NULL,
    stale_timestamp timestamp with time zone NOT NULL,
    system_profile jsonb NOT NULL,
    PRIMARY KEY (id)
);

DELETE FROM inventory.hosts_v1_0;

CREATE INDEX IF NOT EXISTS hosts_v1_0_account_index ON inventory.hosts_v1_0 USING btree (account);
CREATE INDEX IF NOT EXISTS hosts_v1_0_display_name_index ON inventory.hosts_v1_0 USING btree (display_name);
CREATE INDEX IF NOT EXISTS hosts_v1_0_stale_timestamp_index ON inventory.hosts_v1_0 USING btree (stale_timestamp);
CREATE INDEX IF NOT EXISTS hosts_v1_0_system_profile_index ON inventory.hosts_v1_0 USING gin (system_profile jsonb_path_ops);
CREATE INDEX IF NOT EXISTS hosts_v1_0_tags_index ON inventory.hosts_v1_0 USING gin (tags jsonb_path_ops);

CREATE OR REPLACE VIEW inventory.hosts AS
 SELECT hosts_v1_0.id,
    hosts_v1_0.insights_id,
    hosts_v1_0.account,
    hosts_v1_0.display_name,
    hosts_v1_0.created,
    hosts_v1_0.updated,
    hosts_v1_0.stale_timestamp,
    (hosts_v1_0.stale_timestamp + ('1 day'::interval day * '7'::double precision)) AS stale_warning_timestamp,
    (hosts_v1_0.stale_timestamp + ('1 day'::interval day * '14'::double precision)) AS culled_timestamp,
    hosts_v1_0.tags,
    hosts_v1_0.system_profile
 FROM inventory.hosts_v1_0;

GRANT SELECT ON TABLE inventory.hosts TO cyndi_reader;

INSERT INTO inventory.hosts_v1_0 (id, insights_id, account, display_name, tags, updated, created, stale_timestamp, system_profile) VALUES
('00000000000000000000000000000001', '00000000-0000-0000-0001-000000000001', '1', '00000000-0000-0000-0000-000000000001', '[{"key": "k1", "value": "val1", "namespace": "ns1"},{"key": "k2", "value": "val2", "namespace": "ns1"}]',
'2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "sap_sids": ["ABC", "DEF"], "operating_system": {"name": "RHEL", "major": 8, "minor": 10}, "rhsm": {"version": "8.10"}}'),
('00000000000000000000000000000002', '00000000-0000-0000-0002-000000000001', '1', '00000000-0000-0000-0000-000000000002', '[{"key": "k1", "value": "val1", "namespace": "ns1"},{"key": "k2", "value": "val2", "namespace": "ns1"},{"key": "k3", "value": "val3", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "sap_sids": ["ABC"], "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.1"}}'),
('00000000000000000000000000000003', '00000000-0000-0000-0003-000000000001', '1', '00000000-0000-0000-0000-000000000003', '[{"key": "k1", "value": "val1", "namespace": "ns1"}, {"key": "k3", "value": "val4", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.0"}}'),
('00000000000000000000000000000004', '00000000-0000-0000-0004-000000000001', '1', '00000000-0000-0000-0000-000000000004', '[{"key": "k3", "value": "val4", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.3"}}'),
('00000000000000000000000000000005', '00000000-0000-0000-0005-000000000001', '1', '00000000-0000-0000-0000-000000000005', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 3}, "rhsm": {"version": "8.3"}}'),
('00000000000000000000000000000006', '00000000-0000-0000-0006-000000000001', '1', '00000000-0000-0000-0000-000000000006', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 7, "minor": 3}, "rhsm": {"version": "7.3"}}'),
('00000000000000000000000000000007', '00000000-0000-0000-0007-000000000001', '1', '00000000-0000-0000-0000-000000000007','[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": "x"}, "rhsm": {"version": "8.x"}}'),
('00000000000000000000000000000008', '00000000-0000-0000-0008-000000000001', '1', '00000000-0000-0000-0000-000000000008', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 3}, "rhsm": {"version": "8.3"}}'),
('00000000000000000000000000000009', '00000000-0000-0000-0009-000000000001', '2', '00000000-0000-0000-0000-000000000009', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.1"}}'),
('00000000000000000000000000000010', '00000000-0000-0000-0010-000000000001', '2', '00000000-0000-0000-0000-000000000010', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.2"}}'),
('00000000000000000000000000000011', '00000000-0000-0000-0011-000000000001', '2', '00000000-0000-0000-0000-000000000011', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 3}, "rhsm": {"version": "8.3"}}'),
('00000000000000000000000000000012', '00000000-0000-0000-0012-000000000001', '3', '00000000-0000-0000-0000-000000000012', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.1"}}'),
('00000000000000000000000000000013', '00000000-0000-0000-0013-000000000001', '3', '00000000-0000-0000-0000-000000000013', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.2"}}'),
('00000000000000000000000000000014', '00000000-0000-0000-0014-000000000001', '3', '00000000-0000-0000-0000-000000000014', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "operating_system": {"name": "RHEL", "major": 8, "minor": 3}}'),
('00000000000000000000000000000015', '00000000-0000-0000-0015-000000000001', '3', '00000000-0000-0000-0000-000000000015', '[{"key": "k3", "value": "val4", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": false, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.1"}}'),
('00000000000000000000000000000016', '00000000-0000-0000-0016-000000000001', '3', '00000000-0000-0000-0000-000000000016', '[]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": false, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.2"}}');
