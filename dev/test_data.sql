DELETE FROM system_advisories;
DELETE FROM system_repo;
DELETE FROM system_platform;
DELETE FROM repo;
DELETE FROM timestamp_kv;
DELETE FROM advisory_account_data;
DELETE FROM advisory_metadata;
DELETE FROM rh_account;

INSERT INTO rh_account (id, name) VALUES
(0, '0'), (1, '1'), (2, '2'), (3, '3');

INSERT INTO system_platform (id, inventory_id, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload) VALUES
(0, 'INV-0', 0, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(1, 'INV-1', 0, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(2, 'INV-2', 0, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(3, 'INV-3', 0, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(4, 'INV-4', 0, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(5, 'INV-5', 0, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04');

INSERT INTO system_platform (id, inventory_id, rh_account_id,  vmaas_json, json_checksum, first_reported, last_updated, unchanged_since, last_upload) VALUES
(6, 'INV-6', 0, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2017-12-31 08:22:33-04', '2018-10-04 14:13:12-04', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04');

INSERT INTO system_platform (id, inventory_id, rh_account_id,  vmaas_json, json_checksum, last_evaluation, last_upload) VALUES
(7, 'INV-7', 0, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(8, 'INV-8', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(9, 'INV-9', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(10, 'INV-10', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(11, 'INV-11', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04');

INSERT INTO advisory_metadata (id, name, description, synopsis, summary, solution, advisory_type_id,
                               public_date, modified_date, url, severity_id) VALUES
(1, 'RH-1', 'adv-1-des', 'adv-1-syn', 'adv-1-sum', 'adv-1-sol', 1, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url1', NULL),
(2, 'RH-2', 'adv-2-des', 'adv-2-syn', 'adv-2-sum', 'adv-2-sol', 2, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url2', NULL),
(3, 'RH-3', 'adv-3-des', 'adv-3-syn', 'adv-3-sum', 'adv-3-sol', 3, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url3', 2),
(4, 'RH-4', 'adv-4-des', 'adv-4-syn', 'adv-4-sum', 'adv-4-sol', 1, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url4', NULL),
(5, 'RH-5', 'adv-5-des', 'adv-5-syn', 'adv-5-sum', 'adv-5-sol', 2, '2016-09-22 12:00:00-05', '2017-09-22 12:00:00-05', 'url5', NULL),
(6, 'RH-6', 'adv-6-des', 'adv-6-syn', 'adv-6-sum', 'adv-6-sol', 3, '2016-09-22 12:00:00-06', '2017-09-22 12:00:00-06', 'url6', 4),
(7, 'RH-7', 'adv-7-des', 'adv-7-syn', 'adv-7-sum', 'adv-7-sol', 1, '2017-09-22 12:00:00-07', '2017-09-22 12:00:00-07', 'url7', NULL),
(8, 'RH-8', 'adv-8-des', 'adv-8-syn', 'adv-8-sum', 'adv-8-sol', 2, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url8', NULL),
(9, 'RH-9', 'adv-9-des', 'adv-9-syn', 'adv-9-sum', 'adv-9-sol', 3, '2016-09-22 12:00:00-08', '2018-09-22 12:00:00-08', 'url9', NULL);

INSERT INTO system_advisories (system_id, advisory_id, first_reported, when_patched, status_id) VALUES
(0, 1, '2016-09-22 12:00:00-04', NULL, 0),
(0, 2, '2016-09-22 12:00:00-04', NULL, 1),
(0, 3, '2016-09-22 12:00:00-04', NULL, 0),
(0, 4, '2016-09-22 12:00:00-04', NULL, 1),
(0, 5, '2016-09-22 12:00:00-04', NULL, 2),
(0, 6, '2016-09-22 12:00:00-04', NULL, 0),
(0, 7, '2016-09-22 12:00:00-04', NULL, 1),
(0, 8, '2016-09-22 12:00:00-04', NULL, 0),
(0, 9, '2016-09-22 12:00:00-04', '2016-09-22 12:00:00-04', 0),
(1, 1, '2016-09-22 12:00:00-04', NULL, 1),
(2, 1, '2016-09-22 12:00:00-04', NULL, 2),
(3, 1, '2016-09-22 12:00:00-04', NULL, 0),
(4, 1, '2016-09-22 12:00:00-04', NULL, 1),
(5, 1, '2016-09-22 12:00:00-04', NULL, 0),
(6, 1, '2016-09-22 12:00:00-04', NULL, 1),
(7, 1, '2016-09-22 12:00:00-04', NULL, 2),
(8, 1, '2016-09-22 12:00:00-04', NULL, 0),
(9, 1, '2016-09-22 12:00:00-04', NULL, 1),
(10, 1, '2016-09-22 12:00:00-04', NULL, 0);

INSERT INTO repo (id, name) VALUES
(1, 'repo1'),
(2, 'repo2');

INSERT INTO system_repo (system_id, repo_id) VALUES
(1, 1),
(2, 1);

INSERT INTO timestamp_kv (name, value) VALUES
('last_eval_repo_based', '2018-04-05T01:23:45+02:00');

SELECT refresh_all_cached_counts();

ALTER TABLE advisory_metadata ALTER COLUMN id RESTART WITH 100;
ALTER TABLE system_platform ALTER COLUMN id RESTART WITH 100;
ALTER TABLE rh_account ALTER COLUMN id RESTART WITH 100;
