
-- VMaaS sync needs to be able to perform system culling tasks
GRANT SELECT, UPDATE, DELETE ON system_platform to vmaas_sync;