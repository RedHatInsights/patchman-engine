-- One-off repair for corrupt system_advisories_0 (pg_xact / unreadable heap).
-- Gated by database_admin flag repair_system_advisories_0=true (default false).
-- Irreversible: truncates partition data for hash remainder 0 and clears related caches.
-- Safe to re-run only when intentionally wiping bucket 0 again (e.g. failed cutover).
-- TRUNCATE drops table segments without scanning the heap (validated on restore DB).

BEGIN;

TRUNCATE TABLE system_advisories_0;

-- Clear denormalized counts for accounts that hash into remainder 0 (do not read old _0).
DELETE FROM advisory_account_data aad
 WHERE satisfies_hash_partition(
         'system_advisories'::regclass, 32, 0, aad.rh_account_id);

DELETE FROM account_advisory aa
 WHERE satisfies_hash_partition(
         'system_advisories'::regclass, 32, 0, aa.rh_account_id);

COMMIT;
