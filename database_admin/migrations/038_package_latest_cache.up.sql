CREATE MATERIALIZED VIEW IF NOT EXISTS package_latest_cache
AS
SELECT DISTINCT ON (p.name_id) p.name_id, p.id as package_id, sum.value as summary
FROM package p
         INNER JOIN strings sum on p.summary_hash = sum.id
         LEFT JOIN advisory_metadata am on p.advisory_id = am.id
ORDER BY p.name_id, am.public_date;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE package_latest_cache TO vmaas_sync;

GRANT SELECT ON TABLE public.package_latest_cache TO evaluator;
GRANT SELECT ON TABLE public.package_latest_cache TO listener;
GRANT SELECT ON TABLE public.package_latest_cache TO manager;

CREATE UNIQUE INDEX IF NOT EXISTS package_latest_cache_pkey ON package_latest_cache (name_id);

REFRESH MATERIALIZED VIEW package_latest_cache;


CREATE OR REPLACE FUNCTION refresh_latest_packages_view()
    RETURNS void
    SECURITY DEFINER
AS $$
BEGIN
    REFRESH MATERIALIZED VIEW package_latest_cache WITH DATA;
    RETURN;
END;
$$ LANGUAGE plpgsql;
