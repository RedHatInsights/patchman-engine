CREATE OR REPLACE FUNCTION refresh_latest_packages_view()
    RETURNS void
    SECURITY DEFINER
AS
$$
BEGIN
    REFRESH MATERIALIZED VIEW package_latest_cache WITH DATA;
    RETURN;
END;
$$ LANGUAGE plpgsql;

CREATE MATERIALIZED VIEW IF NOT EXISTS package_latest_cache
AS
SELECT DISTINCT ON (p.name_id) p.name_id, p.id as package_id, sum.value as summary
FROM package p
         LEFT JOIN strings sum on p.summary_hash = sum.id
         LEFT JOIN advisory_metadata am on p.advisory_id = am.id
ORDER BY p.name_id, am.public_date DESC NULLS LAST, sum.value DESC NULLS LAST;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE package_latest_cache TO vmaas_sync;

CREATE UNIQUE INDEX IF NOT EXISTS package_latest_cache_pkey ON package_latest_cache (name_id);
