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
DELETE FROM rh_account;
DELETE FROM strings;

INSERT INTO rh_account (id, name) VALUES
(1, '1'), (2, '2'), (3, '3'), (4, '4');

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload, packages_installed, packages_updatable) VALUES
(1, '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2020-09-22 12:00:00-04',0,0),
(2, '00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000002', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04',0,0),
(3, '00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000003', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0),
(4, '00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000004', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0),
(5, '00000000-0000-0000-0000-000000000005','00000000-0000-0000-0000-000000000005', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0),
(6, '00000000-0000-0000-0000-000000000006','00000000-0000-0000-0000-000000000006', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, first_reported, last_updated, unchanged_since, last_upload, packages_installed, packages_updatable) VALUES
(7, '00000000-0000-0000-0000-000000000007','00000000-0000-0000-0000-000000000007', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2017-12-31 08:22:33-04', '2018-10-04 14:13:12-04', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload, packages_installed, packages_updatable) VALUES
(8, '00000000-0000-0000-0000-000000000008','00000000-0000-0000-0000-000000000008', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0),
(9, '00000000-0000-0000-0000-000000000009','00000000-0000-0000-0000-000000000009', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(10, '00000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000010', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(11, '00000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000011', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(12, '00000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000012', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',2,2);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload, opt_out, packages_installed, packages_updatable) VALUES
(13, '00000000-0000-0000-0000-000000000013','00000000-0000-0000-0000-000000000013', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04', true,1,0),
(14, '00000000-0000-0000-0000-000000000014','00000000-0000-0000-0000-000000000014', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04', true,0,0);

INSERT INTO package_name(id,name) VALUES
(1, 'kernel'),
(2, 'firefox'),
(3, 'bash'),
(4, 'curl'),
(5, 'tar'),
(6, 'systemd'),
(7, 'sed'),
(8, 'grep'),
(9, 'which'),
(10, 'passwd');

INSERT INTO strings(id, value) VALUES
('1', 'kernel'),
('2', 'firefox'),
('3', 'bash'),
('4', 'curl'),
('5', 'tar'),
('6', 'systemd'),
('7', 'sed'),
('8', 'grep'),
('9', 'which'),
('10', 'passwd');

INSERT INTO package(id, name_id, evra, description_hash, summary_hash) VALUES
(1, 1, '5.6.13-200.fc31.x86_64', '1', '1'), -- kernel
(2, 2, '76.0.1-1.fc31.x86_64', '2', '2'), -- firefox
(3, 3, '4.4.19-8.el8_0.x86_64', '3', '3'), -- bas
(4, 4, '7.61.1-8.el8.x86_64', '4', '4'), -- curl
(5, 5, '1.30-4.el8.x86_64', '5', '5'), -- tar
(6, 6, '239-13.el8_0.5.x86_64', '6', '6'), -- systemd
(7, 7, '4.5-1.el8.x86_64', '7', '7'), -- sed
(8, 8, '3.1-6.el8.x86_64', '8', '8'), -- grep
(9, 9, '2.21-10.el8.x86_64', '9', '9'), -- which
(10, 10, '0.80-2.el8.x86_64', '10', '10'), -- passwd
(11, 1, '5.6.13-201.fc31.x86_64', '1', '1'); -- kernel

INSERT INTO system_package (rh_account_id, system_id, package_id, update_data) VALUES
(3, 12, 1, '[{"evra": "5.10.13-200.fc31-x86_64", "advisory": "RH-100"}]'),
(3, 12, 2, '[{"evra": "77.0.1-1.fc31-x86_64", "advisory": "RH-1"}, {"evra": "76.0.1-1.fc31-x86_64", "advisory": "RH-2"}]'),
(3, 13, 3, null),
(3, 13, 4, null),
(3, 13, 1, null);

INSERT INTO advisory_metadata (id, name, description, synopsis, summary, solution, advisory_type_id,
                               public_date, modified_date, url, severity_id, cve_list) VALUES
(1, 'RH-1', 'adv-1-des', 'adv-1-syn', 'adv-1-sum', 'adv-1-sol', 1, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url1', NULL, NULL),
(2, 'RH-2', 'adv-2-des', 'adv-2-syn', 'adv-2-sum', 'adv-2-sol', 2, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url2', NULL, NULL),
(3, 'RH-3', 'adv-3-des', 'adv-3-syn', 'adv-3-sum', 'adv-3-sol', 3, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url3', 2, '["CVE-1","CVE-2"]'),
(4, 'RH-4', 'adv-4-des', 'adv-4-syn', 'adv-4-sum', 'adv-4-sol', 1, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url4', NULL, NULL),
(5, 'RH-5', 'adv-5-des', 'adv-5-syn', 'adv-5-sum', 'adv-5-sol', 2, '2016-09-22 12:00:00-05', '2017-09-22 12:00:00-05', 'url5', NULL, NULL),
(6, 'RH-6', 'adv-6-des', 'adv-6-syn', 'adv-6-sum', 'adv-6-sol', 3, '2016-09-22 12:00:00-06', '2017-09-22 12:00:00-06', 'url6', 4, '["CVE-2","CVE-3"]'),
(7, 'RH-7', 'adv-7-des', 'adv-7-syn', 'adv-7-sum', 'adv-7-sol', 1, '2017-09-22 12:00:00-07', '2017-09-22 12:00:00-07', 'url7', NULL, NULL),
(8, 'RH-8', 'adv-8-des', 'adv-8-syn', 'adv-8-sum', 'adv-8-sol', 2, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url8', NULL, NULL),
(9, 'RH-9', 'adv-9-des', 'adv-9-syn', 'adv-9-sum', 'adv-9-sol', 3, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url9', NULL, '["CVE-4"]');

UPDATE advisory_metadata SET package_data = '{"firefox": "77.0.1-1.fc31-x86_64"}' WHERE name = 'RH-9';

UPDATE package SET advisory_id = 1 WHERE id = 1;
UPDATE package SET advisory_id = 7 WHERE id = 11;

INSERT INTO system_advisories (system_id, advisory_id, first_reported, when_patched, status_id) VALUES
(1, 1, '2016-09-22 12:00:00-04', NULL, 0),
(1, 2, '2016-09-22 12:00:00-04', NULL, 1),
(1, 3, '2016-09-22 12:00:00-04', NULL, 0),
(1, 4, '2016-09-22 12:00:00-04', NULL, 1),
(1, 5, '2016-09-22 12:00:00-04', NULL, 2),
(1, 6, '2016-09-22 12:00:00-04', NULL, 0),
(1, 7, '2016-09-22 12:00:00-04', NULL, 1),
(1, 8, '2016-09-22 12:00:00-04', NULL, 0),
(1, 9, '2016-09-22 12:00:00-04', '2016-09-22 12:00:00-04', 0),
(2, 1, '2016-09-22 12:00:00-04', NULL, 1),
(3, 1, '2016-09-22 12:00:00-04', NULL, 2),
(4, 1, '2016-09-22 12:00:00-04', NULL, 0),
(5, 1, '2016-09-22 12:00:00-04', NULL, 1),
(6, 1, '2016-09-22 12:00:00-04', NULL, 0),
(7, 1, '2016-09-22 12:00:00-04', NULL, 1),
(8, 1, '2016-09-22 12:00:00-04', NULL, 2),
(9, 1, '2016-09-22 12:00:00-04', NULL, 0),
(10, 1, '2016-09-22 12:00:00-04', NULL, 1),
(11, 1, '2016-09-22 12:00:00-04', NULL, 0);

INSERT INTO repo (id, name) VALUES
(1, 'repo1'),
(2, 'repo2'),
(3, 'repo3');

INSERT INTO system_repo (system_id, repo_id) VALUES
(2, 1),
(3, 1),
(2, 2);

INSERT INTO timestamp_kv (name, value) VALUES
('last_eval_repo_based', '2018-04-05T01:23:45+02:00');

SELECT refresh_all_cached_counts();

ALTER TABLE advisory_metadata ALTER COLUMN id RESTART WITH 100;
ALTER TABLE system_platform ALTER COLUMN id RESTART WITH 100;
ALTER TABLE rh_account ALTER COLUMN id RESTART WITH 100;
ALTER TABLE repo ALTER COLUMN id RESTART WITH 100;

-- Create "inventory.hosts" for testing purposes. In deployment it's created by remote Cyndi service.

CREATE TABLE IF NOT EXISTS inventory.hosts_v1_0 (
    id uuid NOT NULL,
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

INSERT INTO inventory.hosts_v1_0 (id, account, display_name, tags, updated, created, stale_timestamp, system_profile) VALUES
('00000000000000000000000000000001', '1', '00000000-0000-0000-0000-000000000001', '[{"key": "k1", "value": "val1", "namespace": "ns1"},{"key": "k2", "value": "val2", "namespace": "ns1"}]',
'2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "sap_sids": ["ABC", "DEF"]}'),
('00000000000000000000000000000002', '1', '00000000-0000-0000-0000-000000000002', '[{"key": "k1", "value": "val1", "namespace": "ns1"}, {"key": "k2", "value": "val2", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true, "sap_sids": ["ABC"]}'),
('00000000000000000000000000000003', '1', '00000000-0000-0000-0000-000000000003', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000004', '1', '00000000-0000-0000-0000-000000000004', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000005', '1', '00000000-0000-0000-0000-000000000005', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000006', '1', '00000000-0000-0000-0000-000000000006', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000007', '1', '00000000-0000-0000-0000-000000000007', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000008', '1', '00000000-0000-0000-0000-000000000008', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000009', '2', '00000000-0000-0000-0000-000000000009', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000010', '2', '00000000-0000-0000-0000-000000000010', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000011', '2', '00000000-0000-0000-0000-000000000011', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000012', '3', '00000000-0000-0000-0000-000000000012', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000013', '3', '00000000-0000-0000-0000-000000000013', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}'),
('00000000000000000000000000000014', '3', '00000000-0000-0000-0000-000000000014', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"sap_system": true}');
