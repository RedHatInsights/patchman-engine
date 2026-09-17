DROP FUNCTION IF EXISTS backfill_account_advisory(rh_account_id_in INTEGER);

ALTER TABLE rh_account DROP COLUMN IF EXISTS valid_advisory_cache;

DROP FUNCTION IF EXISTS refresh_advisory_caches(advisory_id_in INTEGER, rh_account_id_in INTEGER);

DROP FUNCTION IF EXISTS refresh_advisory_cached_counts(advisory_name varchar);

DROP FUNCTION IF EXISTS refresh_advisory_account_cached_counts(advisory_name varchar, rh_account_name varchar);

DROP FUNCTION IF EXISTS refresh_account_cached_counts(rh_account_in varchar);

DROP FUNCTION IF EXISTS refresh_all_cached_counts();

DROP FUNCTION IF EXISTS refresh_advisory_caches_multi(advisory_ids_in INTEGER[], rh_account_id_in INTEGER);
