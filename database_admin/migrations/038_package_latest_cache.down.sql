DROP FUNCTION IF EXISTS refresh_latest_packages_view;
DROP INDEX IF EXISTS package_latest_cache_pkey;
DROP MATERIALIZED VIEW IF EXISTS package_latest_cache;