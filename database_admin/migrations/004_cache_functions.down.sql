DROP TRIGGER system_platform_opt_out_cache ON system_platform;

DROP FUNCTION opt_out_system_update_cache();

DROP FUNCTION delete_system(inventory_id_in varchar);

DROP FUNCTION refresh_system_cached_counts(inventory_id_in varchar);

DROP FUNCTION refresh_advisory_account_cached_counts(advisory_name varchar, rh_account_name varchar);

DROP FUNCTION refresh_advisory_cached_counts(advisory_name varchar);

DROP FUNCTION refresh_account_cached_counts(rh_account_in varchar);

DROP FUNCTION refresh_all_cached_counts();

DROP FUNCTION system_advisories_count(system_id_in INT, advisory_type_id_in INT);

DROP FUNCTION update_system_caches(system_id_in INT);
