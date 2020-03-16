-- vmaas_sync needs to update stale mark, which creates and deletes advisory_account_data
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO vmaas_sync;

-- vmaas_sync needs to delete culled systems, which cascades to system_advisories
GRANT SELECT, DELETE ON system_advisories TO vmaas_sync;

-- vmaas_sync needs to delete culled systems, which cascades to system_repo
GRANT SELECT, DELETE on system_repo to vmaas_sync;