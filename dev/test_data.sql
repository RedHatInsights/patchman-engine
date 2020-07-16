DELETE FROM system_advisories;
DELETE FROM system_repo;
DELETE FROM system_platform;
DELETE FROM deleted_system;
DELETE FROM repo;
DELETE FROM timestamp_kv;
DELETE FROM advisory_account_data;
DELETE FROM advisory_metadata;
DELETE FROM rh_account;

INSERT INTO rh_account (id, name) VALUES
(1, '1'), (2, '2'), (3, '3'), (4, '4');

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload, packages_installed, packages_updatable) VALUES
(1, 'INV-1', 'INV-1', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2020-09-22 12:00:00-04',0,0),
(2, 'INV-2', 'INV-2', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04',0,0),
(3, 'INV-3', 'INV-3', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0),
(4, 'INV-4', 'INV-4', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0),
(5, 'INV-5', 'INV-5', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-18 12:00:00-04',0,0),
(6, 'INV-6', 'INV-6', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, first_reported, last_updated, unchanged_since, last_upload, packages_installed, packages_updatable) VALUES
(7, 'INV-7','INV-7', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2017-12-31 08:22:33-04', '2018-10-04 14:13:12-04', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload, packages_installed, packages_updatable) VALUES
(8, 'INV-8', 'INV-8', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04',0,0),
(9, 'INV-9', 'INV-9', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(10, 'INV-10', 'INV-10', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(11, 'INV-11', 'INV-11', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',0,0),
(12, 'INV-12', 'INV-12', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04',2,2);

INSERT INTO system_platform (id, inventory_id, display_name, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload, opt_out, packages_installed, packages_updatable) VALUES
(13, 'INV-13', 'INV-13', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04', true,0,0),
(14, 'INV-14', 'INV-14', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-01-22 12:00:00-04', true,0,0);

insert into system_tags (tag, system_id) VALUES
('satellite/organization=rh', 1),
('satellite/organization=rh', 2),
('satellite/organization=ibm', 1);

INSERT INTO package_name(id,name) VALUES
(1, 'kernel'),
(2, 'firefox');

INSERT INTO strings(id, value) VALUES
('1', 'kernel'),
('2', 'firefox');

INSERT INTO package(id, name_id, evra, description_hash, summary_hash) VALUES
(1, 1, '5.6.13-200.fc31-x86_64', '1', '1'),
(2, 2, '76.0.1-1.fc31-x86_64', '2', '2');

INSERT INTO system_package (system_id, package_id, update_data) VALUES
(12, 1, '[{"evra": "5.10.13-200.fc31-x86_64", "advisory": "RH-100"}]'),
(12, 2, '[{"evra": "77.0.1-1.fc31-x86_64", "advisory": "RH-1"}, {"evra": "76.0.1-1.fc31-x86_64", "advisory": "RH-2"}]'),
(13, 1, null);

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
