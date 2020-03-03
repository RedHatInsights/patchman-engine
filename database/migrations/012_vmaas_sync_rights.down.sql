-- vmaas_sync needs to update stale mark, which creates and deletes advisory_account_data
REVOKE SELECT, INSERT, UPDATE, DELETE ON advisory_account_data FROM vmaas_sync;
