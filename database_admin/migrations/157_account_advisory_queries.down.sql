DROP FUNCTION IF EXISTS backfill_account_advisory(INTEGER);

DROP FUNCTION IF EXISTS refresh_account_advisory_caches(INTEGER, INTEGER);

DROP FUNCTION IF EXISTS refresh_account_advisory_caches_multi(INTEGER[], INTEGER);

DROP INDEX IF EXISTS account_advisory_systems_applicable_idx;
DROP INDEX IF EXISTS account_advisory_systems_installable_idx;

TRUNCATE account_advisory;
