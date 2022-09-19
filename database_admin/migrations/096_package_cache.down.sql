REVOKE UPDATE ON rh_account FROM vmaas_sync;
ALTER TABLE rh_account DROP valid_package_cache;
DROP FUNCTION refresh_packages_caches;

CREATE OR REPLACE FUNCTION refresh_all_cached_counts()
    RETURNS void AS
$refresh_all_cached_counts$
BEGIN
    PERFORM refresh_system_caches(NULL, NULL);
    PERFORM refresh_advisory_caches(NULL, NULL);
END;
$refresh_all_cached_counts$
    LANGUAGE 'plpgsql';
