DELETE FROM system_advisories;
DELETE FROM system_repo;
DELETE FROM system_package2;
DELETE FROM system_patch;
DELETE FROM system_inventory;
DELETE FROM deleted_system;
DELETE FROM repo;
DELETE FROM timestamp_kv;
DELETE FROM advisory_account_data;
DELETE FROM package_account_data;
DELETE FROM package;
DELETE FROM package_name;
DELETE FROM advisory_metadata;
DELETE FROM template;
DELETE FROM rh_account;
DELETE FROM strings;
DELETE FROM inventory.hosts_v1_0;

INSERT INTO rh_account (id, name, org_id) VALUES
(1, 'acc-1', 'org_1'), (2, 'acc-2', 'org_2'), (3, 'acc-3', 'org_3'), (4, 'acc-4', 'org_4');

INSERT INTO template (id, rh_account_id, uuid, environment_id, name, description, config, arch, version, creator) VALUES
(1, 1, '99900000-0000-0000-0000-000000000001', '99900000000000000000000000000001', 'temp1-1', 'desc1', '{"to_time": "2010-09-22T00:00:00+00:00"}', 'x86_64', '8', 'user1'),
(2, 1, '99900000-0000-0000-0000-000000000002', '99900000000000000000000000000002', 'temp2-1', 'desc2', '{"to_time": "2021-01-01T00:00:00+00:00"}', 'x86_64', '8', 'user2'),
(3, 1, '99900000-0000-0000-0000-000000000003', '99900000000000000000000000000003', 'temp3-1',    NULL, '{"to_time": "2000-01-01T00:00:00+00:00"}', 'x86_64', '8', 'user3'),
(4, 3, '99900000-0000-0000-0000-000000000004', '99900000000000000000000000000004', 'temp4-3', 'desc4', '{"to_time": "2000-01-01T00:00:00+00:00"}', 'x86_64', '8', 'user4');

INSERT INTO system_inventory (id, inventory_id, rh_account_id, vmaas_json, json_checksum, last_upload, display_name, reporter_id, arch, tags, created, workspaces, os_name, os_major, os_minor, rhsm_version, subscription_manager_id, sap_workload, sap_workload_sids, mssql_workload, mssql_workload_version) VALUES
(1, '00000000-0000-0000-0000-000000000001', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2020-09-22 12:00:00-04', '00000000-0000-0000-0000-000000000001', 1, 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"},{"key": "k2", "value": "val2", "namespace": "ns1"}]',                                                    '2018-08-26 12:00:00-04', '[{"id": "inventory-group-1", "name": "group1"}]', 'RHEL', 8, 10, '8.10', NULL,                                   true, ARRAY['ABC', 'DEF', 'GHI'], false, NULL),
(2, '00000000-0000-0000-0000-000000000002', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-09-22 12:00:00-04', '00000000-0000-0000-0000-000000000002', 1, 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"},{"key": "k2", "value": "val2", "namespace": "ns1"},{"key": "k3", "value": "val3", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[{"id": "inventory-group-1", "name": "group1"}]', 'RHEL', 8,  1, '8.1',  NULL,                                   true, ARRAY['ABC'],               false, NULL),
(3, '00000000-0000-0000-0000-000000000003', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-09-18 12:00:00-04', '00000000-0000-0000-0000-000000000003', 1, 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}, {"key": "k3", "value": "val4", "namespace": "ns1"}]',                                                   '2018-08-26 12:00:00-04', '[{"id": "inventory-group-1", "name": "group1"}]', 'RHEL', 8,  1, '8.0',  NULL,                                   true, NULL,                       false, NULL),
(4, '00000000-0000-0000-0000-000000000004', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-09-18 12:00:00-04', '00000000-0000-0000-0000-000000000004', 1, 'x86_64', '[{"key": "k3", "value": "val4", "namespace": "ns1"}]',                                                                                                       '2018-08-26 12:00:00-04', '[{"id": "inventory-group-1", "name": "group1"}]', 'RHEL', 8,  2, '8.3',  'cccccccc-0000-0000-0001-000000000004', true, NULL,                       false, NULL),
(5, '00000000-0000-0000-0000-000000000005', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-09-18 12:00:00-04', '00000000-0000-0000-0000-000000000005', 1, 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',                                                                                                       '2018-08-26 12:00:00-04', '[{"id": "inventory-group-1", "name": "group1"}]', 'RHEL', 8,  3, '8.3',  'cccccccc-0000-0000-0001-000000000005', true, NULL,                       false, NULL),
(6, '00000000-0000-0000-0000-000000000006', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-08-26 12:00:00-04', '00000000-0000-0000-0000-000000000006', 1, 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',                                                                                                       '2018-08-26 12:00:00-04', '[{"id": "inventory-group-1", "name": "group1"}]', 'RHEL', 7,  3, '7.3',  NULL,                                   true, NULL,                        true, '15.3.0');
INSERT INTO system_patch (system_id, rh_account_id, last_evaluation, third_party, template_id) VALUES
(1, 1, '2018-09-22 12:00:00-04', true , 1),
(2, 1, '2018-09-22 12:00:00-04', false, 1),
(3, 1, '2018-09-22 12:00:00-04', false, 2),
(4, 1, '2018-09-22 12:00:00-04', false, NULL),
(5, 1, '2018-09-22 12:00:00-04', false, NULL),
(6, 1, '2018-09-22 12:00:00-04', false, NULL);

INSERT INTO system_inventory (id, inventory_id, rh_account_id, vmaas_json, json_checksum, last_updated, unchanged_since, last_upload, display_name, arch, tags, created, workspaces, os_name, os_major, rhsm_version, subscription_manager_id, sap_workload, ansible_workload, ansible_workload_controller_version) VALUES
(7, '00000000-0000-0000-0000-000000000007', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-10-04 14:13:12-04', '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '00000000-0000-0000-0000-000000000007', 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[{"id": "inventory-group-2", "name": "group2"}]', 'RHEL', 8, '8.x', 'cccccccc-0000-0000-0001-000000000007', true, true, '1.0');
INSERT INTO system_patch (system_id, rh_account_id) VALUES
(7, 1);

INSERT INTO system_inventory (id, inventory_id, rh_account_id, vmaas_json, json_checksum, last_upload, display_name, arch, tags, created, workspaces, os_name, os_major, os_minor, rhsm_version, subscription_manager_id, sap_workload) VALUES
( 8, '00000000-0000-0000-0000-000000000008', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-08-26 12:00:00-04', '00000000-0000-0000-0000-000000000008', 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[{"id": "inventory-group-2", "name": "group2"}]', 'RHEL', 8, 3, '8.3', 'cccccccc-0000-0000-0001-000000000008', true),
( 9, '00000000-0000-0000-0000-000000000009', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000009', 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', NULL,                                              'RHEL', 8, 1, '8.1', NULL,                                   true),
(10, '00000000-0000-0000-0000-000000000010', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000010', 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', NULL,                                              'RHEL', 8, 2, '8.2', NULL,                                   true),
(11, '00000000-0000-0000-0000-000000000011', 2, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000011', 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[]',                                              'RHEL', 8, 3, '8.3', NULL,                                   true),
(12, '00000000-0000-0000-0000-000000000012', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000012', 'x86_64', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[]',                                              'RHEL', 8, 1, '8.1', NULL,                                   true);
INSERT INTO system_patch (system_id, rh_account_id, last_evaluation, packages_installed, packages_installable, packages_applicable) VALUES
( 8, 1, '2018-09-22 12:00:00-04', 0, 0, 0),
( 9, 2, '2018-09-22 12:00:00-04', 0, 0, 0),
(10, 2, '2018-09-22 12:00:00-04', 0, 0, 0),
(11, 2, '2018-09-22 12:00:00-04', 0, 0, 0),
(12, 3, '2018-09-22 12:00:00-04', 2, 2, 2);

INSERT INTO system_inventory (id, inventory_id, rh_account_id, vmaas_json, json_checksum, last_upload, display_name, yum_updates, tags, created, workspaces, os_name, os_major, os_minor, rhsm_version, sap_workload) VALUES
(13, '00000000-0000-0000-0000-000000000013', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000013', NULL, '[{"key": "k1", "value": "val1", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[]', 'RHEL', 8, 2, '8.2', true),
(14, '00000000-0000-0000-0000-000000000014', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000014', NULL, '[{"key": "k1", "value": "val1", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[]', 'RHEL', 8, 3,  NULL, true),
(15, '00000000-0000-0000-0000-000000000015', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000015', '{"update_list": {"suricata-0:6.0.3-2.fc35.i686": {"available_updates": [{"erratum": "RHSA-2021:3801", "basearch": "i686", "releasever": "ser1", "repository": "group_oisf:suricata-6.0", "package": "suricata-0:6.0.4-2.fc35.i686"}]}}, "basearch": "i686", "releasever": "ser1"}', '[{"key": "k3", "value": "val4", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[]', 'RHEL', 8, 1, '8.1', false);
INSERT INTO system_patch (system_id, rh_account_id, last_evaluation, packages_installed) VALUES
(13, 3, '2018-09-22 12:00:00-04', 1),
(14, 3, '2018-09-22 12:00:00-04', 0),
(15, 3, '2018-09-22 12:00:00-04', 0);

INSERT INTO system_inventory (id, inventory_id, rh_account_id, vmaas_json, json_checksum, last_upload, display_name, tags, created, workspaces, os_name, os_major, os_minor, rhsm_version, ansible_workload, ansible_workload_controller_version, mssql_workload, mssql_workload_version) VALUES
(16, '00000000-0000-0000-0000-000000000016', 3, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000016', '[]', '2018-08-26 12:00:00-04', '[]', 'RHEL', 8, 2, '8.2', false,  NULL, false, NULL),
(17, '00000000-0000-0000-0000-000000000017', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2018-01-22 12:00:00-04', '00000000-0000-0000-0000-000000000017', '[]', '2018-08-26 12:00:00-04', '[]', 'RHEL', 8, 1, '8.1', true, '1.0', true, '15.3.0');
INSERT INTO system_patch (system_id, rh_account_id, last_evaluation, packages_installed, packages_installable, packages_applicable, template_id) VALUES
(16, 3, '2018-09-22 12:00:00-04', 1, 1, 1, 4),
(17, 1, '2018-09-22 12:00:00-04', 2, 2, 2, NULL);

INSERT INTO system_inventory (id, inventory_id, rh_account_id, vmaas_json, json_checksum, last_upload, display_name, reporter_id, arch, tags, created, workspaces, os_name, os_major, os_minor, rhsm_version, subscription_manager_id, sap_workload) VALUES
(18, '00000000-0000-0000-0000-000000000018', 1, '{ "package_list": [ "kernel-2.6.32-696.20.1.el6.x86_64" ], "repository_list": [ "rhel-6-server-rpms" ] }', '1', '2020-09-22 12:00:00-04', '00000000-0000-0000-0000-000000000018', 1, 'x86_64', '[{"key": "k3", "value": "val4", "namespace": "ns1"}]', '2018-08-26 12:00:00-04', '[{"id": "inventory-group-1", "name": "group1"}]', 'RHEL', 8, 2, '8.3', '99999999-9999-9999-9999-999999999404', true);
INSERT INTO system_patch (system_id, rh_account_id, last_evaluation, third_party) VALUES
(18, 1, '2018-09-22 12:00:00-04', true);

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

INSERT INTO advisory_metadata (id, name, description, synopsis, summary, solution, advisory_type_id,
                               public_date, modified_date, url, severity_id, cve_list, release_versions, synced) VALUES
(14, 'RHSA-2021:3801', 'adv-14-des', 'adv-14-syn', 'adv-14-sum', 'adv-14-sol', 3, '2016-09-22 12:00:00-04', '2017-09-22 12:00:00-04', 'url14', NULL, NULL, '["7.0","ser1"]', true);

UPDATE advisory_metadata SET package_data = '["firefox-77.0.1-1.fc31.x86_64", "firefox-77.0.1-1.fc31.s390"]' WHERE name = 'RH-9';

INSERT INTO system_advisories (rh_account_id, system_id, advisory_id, first_reported, status_id) VALUES
(1, 1, 1, '2016-09-22 12:00:00-04', 0),
(1, 1, 2, '2016-09-22 12:00:00-04', 1),
(1, 1, 3, '2016-09-22 12:00:00-04', 0),
(1, 1, 4, '2016-09-22 12:00:00-04', 1),
(1, 1, 5, '2016-09-22 12:00:00-04', 0),
(1, 1, 6, '2016-09-22 12:00:00-04', 0),
(1, 1, 7, '2016-09-22 12:00:00-04', 1),
(1, 1, 8, '2016-09-22 12:00:00-04', 0),
(1, 2, 1, '2016-09-22 12:00:00-04', 1),
(1, 3, 1, '2016-09-22 12:00:00-04', 0),
(1, 4, 1, '2016-09-22 12:00:00-04', 0),
(1, 5, 1, '2016-09-22 12:00:00-04', 1),
(1, 6, 1, '2016-09-22 12:00:00-04', 0),
(1, 8, 10, '2016-09-22 12:00:00-04', 0),
(1, 8, 11, '2016-09-22 12:00:00-04', 0),
(1, 8, 12, '2016-09-22 12:00:00-04', 0),
(1, 8, 13, '2016-09-22 12:00:00-04', 0),
(2, 10, 1, '2016-09-22 12:00:00-04', 1),
(2, 11, 1, '2016-09-22 12:00:00-04', 0);

INSERT INTO repo (id, name, third_party) VALUES
(1, 'repo1', false),
(2, 'repo2', false),
(3, 'repo3', false),
-- repo4 is not in platform mock for a purpose
(4, 'repo4', true);

INSERT INTO system_repo (rh_account_id, system_id, repo_id) VALUES
(1, 2, 1),
(1, 3, 1),
(1, 2, 2),
(1, 17, 1);


INSERT INTO package_name(id, name, summary) VALUES
(101, 'kernel', 'The Linux kernel'),
(102, 'firefox', 'Mozilla Firefox Web browser'),
(103, 'bash', 'The GNU Bourne Again shell'),
(104, 'curl', 'A utility for getting files from remote servers...'),
(105, 'tar', 'tar summary'),
(106, 'systemd', 'systemd summary'),
(107, 'sed', 'sed summary'),
(108, 'grep', 'grep summary'),
(109, 'which', 'which summary'),
(110, 'passwd', 'passwd summary');

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

INSERT INTO package(id, name_id, evra, description_hash, summary_hash, advisory_id, synced) VALUES
(1, 101, '5.6.13-200.fc31.x86_64', '11', '1', 1, true), -- kernel
(2, 102, '76.0.1-1.fc31.x86_64', '22', '2', 1, true), -- firefox
(3, 103, '4.4.19-8.el8_0.x86_64', '33', '3', 3, true), -- bash
(4, 104, '7.61.1-8.el8.x86_64', '44', '4', 4, true), -- curl
(5, 105, '1.30-4.el8.x86_64', '55', '5', 5, true), -- tar
(6, 106, '239-13.el8_0.5.x86_64', '66', '6', 6, true), -- systemd
(7, 107, '4.5-1.el8.x86_64', '77', '7', 7, true), -- sed
(8, 108, '3.1-6.el8.x86_64', '88', '8', 8, true), -- grep
(9, 109, '2.21-10.el8.x86_64', '99', '9', 9, true), -- which
(10, 110, '0.80-2.el8.x86_64', '1010', '10', 9, true), -- passwd
(11, 101, '5.6.13-201.fc31.x86_64', '11', '1', 7, true), -- kernel
(12, 102, '76.0.1-2.fc31.x86_64', '22', '2', null, true), -- firefox
(13, 102, '77.0.1-1.fc31.x86_64', '22', '2', null, true); -- firefox

INSERT INTO system_package2 (rh_account_id, system_id, name_id, package_id, installable_id, applicable_id) VALUES
(1, 2, 101, 1, 11, null),
(1, 2, 102, 2, 12, null),
(1, 3, 101, 1, 11, null),
(1, 3, 102, 2, 12, null),
(3, 12, 101, 1, 11, null),
(3, 12, 102, 2, 12, null),
(3, 13, 101, 1, null, null),
(3, 13, 102, 2, 12, 13),
(3, 13, 103, 3, null, null),
(3, 13, 104, 4, null, null),
(3, 16, 101, 1, 11, 11),
(1, 17, 101, 1, 11, null),
(1, 17, 102, 2, 12, null);

INSERT INTO timestamp_kv (name, value) VALUES
('last_eval_repo_based', '2018-04-05T01:23:45+02:00');

INSERT INTO inventory.hosts_v1_0 (id, insights_id, account, display_name, tags, updated, created, stale_timestamp, system_profile, reporter, per_reporter_staleness, org_id, groups) VALUES
('00000000000000000000000000000001', '00000000-0000-0000-0001-000000000001', '1', '00000000-0000-0000-0000-000000000001', '[{"key": "k1", "value": "val1", "namespace": "ns1"},{"key": "k2", "value": "val2", "namespace": "ns1"}]',
'2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true, "sids": ["ABC", "DEF", "GHI"]}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 10}, "rhsm": {"version": "8.10"}}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-1", "name": "group1"}]'),
('00000000000000000000000000000002', '00000000-0000-0000-0002-000000000001', '1', '00000000-0000-0000-0000-000000000002', '[{"key": "k1", "value": "val1", "namespace": "ns1"},{"key": "k2", "value": "val2", "namespace": "ns1"},{"key": "k3", "value": "val3", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true, "sids": ["ABC"]}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.1"}}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-1", "name": "group1"}]'),
('00000000000000000000000000000003', '00000000-0000-0000-0003-000000000001', '1', '00000000-0000-0000-0000-000000000003', '[{"key": "k1", "value": "val1", "namespace": "ns1"}, {"key": "k3", "value": "val4", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.0"}}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-1", "name": "group1"}]'),
('00000000000000000000000000000004', '00000000-0000-0000-0004-000000000001', '1', '00000000-0000-0000-0000-000000000004', '[{"key": "k3", "value": "val4", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.3"}, "owner_id": "cccccccc-0000-0000-0001-000000000004"}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-1", "name": "group1"}]'),
('00000000000000000000000000000005', '00000000-0000-0000-0005-000000000001', '1', '00000000-0000-0000-0000-000000000005', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 3}, "rhsm": {"version": "8.3"}, "owner_id": "cccccccc-0000-0000-0001-000000000005"}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-1", "name": "group1"}]'),
('00000000000000000000000000000006', '00000000-0000-0000-0006-000000000001', '1', '00000000-0000-0000-0000-000000000006', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}, "mssql": { "version": "15.3.0"}}, "operating_system": {"name": "RHEL", "major": 7, "minor": 3}, "rhsm": {"version": "7.3"}}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-1", "name": "group1"}]'),
('00000000000000000000000000000007', '00000000-0000-0000-0007-000000000001', '1', '00000000-0000-0000-0000-000000000007','[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}, "ansible": {"controller_version": "1.0"}}, "operating_system": {"name": "RHEL", "major": 8, "minor": "x"}, "rhsm": {"version": "8.x"}, "owner_id": "cccccccc-0000-0000-0001-000000000007"}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-2", "name": "group2"}]'),
('00000000000000000000000000000008', '00000000-0000-0000-0008-000000000001', '1', '00000000-0000-0000-0000-000000000008', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 3}, "rhsm": {"version": "8.3"}, "owner_id": "cccccccc-0000-0000-0001-000000000008"}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-2", "name": "group2"}]'),
('00000000000000000000000000000009', '00000000-0000-0000-0009-000000000001', '2', '00000000-0000-0000-0000-000000000009', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.1"}}',
 'puptoo', '{}', 'org_2', NULL),
('00000000000000000000000000000010', '00000000-0000-0000-0010-000000000001', '2', '00000000-0000-0000-0000-000000000010', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.2"}}',
 'puptoo', '{}', 'org_2', NULL),
('00000000000000000000000000000011', '00000000-0000-0000-0011-000000000001', '2', '00000000-0000-0000-0000-000000000011', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 3}, "rhsm": {"version": "8.3"}}',
 'puptoo', '{}', 'org_2', '[]'),
('00000000000000000000000000000012', '00000000-0000-0000-0012-000000000001', '3', '00000000-0000-0000-0000-000000000012', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.1"}}',
 'puptoo', '{}', 'org_3', '[]'),
('00000000000000000000000000000013', '00000000-0000-0000-0013-000000000001', '3', '00000000-0000-0000-0000-000000000013', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.2"}}',
 'puptoo', '{}', 'org_3', '[]'),
('00000000000000000000000000000014', '00000000-0000-0000-0014-000000000001', '3', '00000000-0000-0000-0000-000000000014', '[{"key": "k1", "value": "val1", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 3}}',
 'puptoo', '{}', 'org_3', '[]'),
('00000000000000000000000000000015', '00000000-0000-0000-0015-000000000001', '3', '00000000-0000-0000-0000-000000000015', '[{"key": "k3", "value": "val4", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": false}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "rhsm": {"version": "8.1"}}',
 'puptoo', '{}', 'org_3', '[]'),
('00000000000000000000000000000016', '00000000-0000-0000-0016-000000000001', '3', '00000000-0000-0000-0000-000000000016', '[]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": false}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.2"}}',
 'puptoo', '{}', 'org_3', '[]'),
('00000000000000000000000000000017', '00000000-0000-0000-0017-000000000001', '3', '00000000-0000-0000-0000-000000000017', '[]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04',
 '{"rhsm": {"version": "8.1"}, "operating_system": {"name": "RHEL", "major": 8, "minor": 1}, "workloads": {"ansible": {"controller_version": "1.0", "hub_version": "3.4.1", "catalog_worker_version": "100.387.9846.12", "sso_version": "1.28.3.52641.10000513168495123"}, "mssql": { "version": "15.3.0"}}}',
 'puptoo', '{}', 'org_3', '[]'),
 ('00000000000000000000000000000018', '00000000-0000-0000-0018-000000000001', '1', '00000000-0000-0000-0000-000000000018', '[{"key": "k3", "value": "val4", "namespace": "ns1"}]',
 '2018-09-22 12:00:00-04', '2018-08-26 12:00:00-04', '2018-08-26 12:00:00-04', '{"workloads": {"sap": {"sap_system": true}}, "operating_system": {"name": "RHEL", "major": 8, "minor": 2}, "rhsm": {"version": "8.3"}, "owner_id": "99999999-9999-9999-9999-999999999404"}',
 'puptoo', '{}', 'org_1', '[{"id": "inventory-group-1", "name": "group1"}]');

SELECT refresh_all_cached_counts();

ALTER TABLE advisory_metadata ALTER COLUMN id RESTART WITH 100;
ALTER TABLE system_inventory ALTER COLUMN id RESTART WITH 100;
ALTER TABLE rh_account ALTER COLUMN id RESTART WITH 100;
ALTER TABLE repo ALTER COLUMN id RESTART WITH 100;
ALTER TABLE package ALTER COLUMN id RESTART WITH 100;
ALTER TABLE package_name ALTER COLUMN id RESTART WITH 150;
ALTER TABLE template ALTER COLUMN id RESTART WITH 100;
