ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS package_data JSONB DEFAULT NULL;


CREATE INDEX IF NOT EXISTS
    system_platform_pkgdata_idx ON system_platform
    USING GIN ((system_platform.package_data));
-- The gin index should speed up WHERE package_data ? 'pkgname' queries
-- WHERE package_data ?& array ['kernel', 'firefox'] are not yet sped up
-- TODO: Investigate


ALTER TABLE advisory_metadata
    ADD COLUMN IF NOT EXISTS package_data JSONB DEFAULT NULL;


CREATE INDEX IF NOT EXISTS
    advisory_metadata_pkgdata_idx ON advisory_metadata
    USING GIN((advisory_metadata.package_data));