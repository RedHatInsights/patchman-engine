DELETE FROM system_repo; DELETE FROM system_platform; DELETE FROM rh_account; DELETE FROM repo;
DELETE FROM timestamp_kv;

INSERT INTO rh_account (id, name) VALUES
(0, '0'), (1, '1'), (2, '2'), (3, '3');

INSERT INTO system_platform (id, inventory_id, rh_account_id, s3_url, vmaas_json, json_checksum, last_upload) VALUES
(0, 'INV-0', 0, 'https://s3/1', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04'),
(1, 'INV-1', 0, 'https://s3/2', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04'),
(2, 'INV-2', 0, 'https://s3/3', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04');

INSERT INTO system_platform (id, inventory_id, rh_account_id, s3_url, vmaas_json, json_checksum, last_evaluation, last_upload) VALUES
(3, 'INV-3', 0, 'https://s3/4', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(4, 'INV-4', 0, 'https://s3/5', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(5, 'INV-5', 0, 'https://s3/6', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04');

INSERT INTO system_platform (id, inventory_id, rh_account_id, s3_url, vmaas_json, json_checksum, first_reported, last_updated, unchanged_since, last_upload) VALUES
(6, 'INV-6', 0, 'https://s3/7', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2017-12-31 08:22:33-04', '2018-10-04 14:13:12-04', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04');

INSERT INTO system_platform (id, inventory_id, rh_account_id, s3_url, vmaas_json, json_checksum, last_evaluation, last_upload) VALUES
(7, 'INV-7', 0, 'https://s3/7', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04');

INSERT INTO system_platform (id, inventory_id, rh_account_id, s3_url, vmaas_json, json_checksum, last_evaluation, last_upload) VALUES
(8, 'INV-8', 1, 'https://s3/8', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(9, 'INV-9', 1, 'https://s3/9', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(10, 'INV-10', 1, 'https://s3/10', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04'),
(11, 'INV-11', 2, 'https://s3/11', '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ]}', '1', '2018-09-22 12:00:00-04', '2018-09-22 12:00:00-04');

-- TODO add advasories_metadata, system_advasories, adv. account metadata

INSERT INTO repo (id, name) VALUES
(1, 'repo1'),
(2, 'repo2');

INSERT INTO system_repo (system_id, repo_id) VALUES
(1, 1),
(2, 1);

INSERT INTO timestamp_kv (name, value) VALUES
('last_eval_repo_based', '2018-04-05T01:23:45+02:00');

SELECT refresh_all_cached_counts();

SELECT setval('system_platform_id_seq', 100);
SELECT setval('rh_account_id_seq', 100);
