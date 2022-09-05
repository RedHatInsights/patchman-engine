-- estimation: Time: 19301.326 ms (00:19.301)
ALTER TABLE advisory_metadata ALTER COLUMN id TYPE BIGINT;

-- estimation: Time: 308144.886 ms (05:08.145)
ALTER TABLE system_advisories ALTER COLUMN system_id TYPE BIGINT,
                              ALTER COLUMN advisory_id TYPE BIGINT;

-- estimation: Time: 16696.425 ms (00:16.696)
ALTER TABLE advisory_account_data ALTER COLUMN advisory_id TYPE BIGINT;

CREATE OR REPLACE FUNCTION refresh_advisory_cached_counts(advisory_name varchar)
    RETURNS void AS
$refresh_advisory_cached_counts$
DECLARE
    advisory_id_id BIGINT;
BEGIN
    -- update system count for advisory
    SELECT id FROM advisory_metadata WHERE name = advisory_name INTO advisory_id_id;

    PERFORM refresh_advisory_caches(advisory_id_id, NULL);
END;
$refresh_advisory_cached_counts$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION refresh_advisory_account_cached_counts(advisory_name varchar, rh_account_name varchar)
    RETURNS void AS
$refresh_advisory_account_cached_counts$
DECLARE
    advisory_md_id   BIGINT;
    rh_account_id_in INT;
BEGIN
    -- update system count for ordered advisories
    SELECT id FROM advisory_metadata WHERE name = advisory_name INTO advisory_md_id;
    SELECT id FROM rh_account WHERE name = rh_account_name INTO rh_account_id_in;

    PERFORM refresh_advisory_caches(advisory_md_id, rh_account_id_in);
END;
$refresh_advisory_account_cached_counts$
    LANGUAGE 'plpgsql';
