-- Replace SHA-256 strings.id values with SHA-512 of strings.value.
--
-- Run only after every vmaas_sync writer stores SHA-512. A writer still hashing
-- with SHA-256 can point package rows back at ids this script removes.
-- Idempotent: a second run changes nothing once every id is 64 bytes.
-- Rewrites every package row that still points at a 32-byte id. The SQL job
-- holds those locks until the transaction commits.
--
-- Verify afterward. Both counts must be 0:
--   SELECT count(*) FROM strings WHERE octet_length(id) = 32;
--   SELECT count(*) FROM package
--   WHERE octet_length(description_hash) = 32
--      OR octet_length(summary_hash) = 32;

CREATE INDEX IF NOT EXISTS package_description_hash_idx ON package (description_hash);
CREATE INDEX IF NOT EXISTS package_summary_hash_idx ON package (summary_hash);

INSERT INTO strings (id, value)
SELECT sha512(convert_to(s.value, 'UTF8')), s.value
FROM strings s
WHERE octet_length(s.id) = 32
  AND EXISTS (
      SELECT 1
      FROM package p
      WHERE p.description_hash = s.id
         OR p.summary_hash = s.id
  )
ON CONFLICT (id) DO NOTHING;

UPDATE package p
SET description_hash = sha512(convert_to(s.value, 'UTF8'))
FROM strings s
WHERE p.description_hash = s.id
  AND octet_length(description_hash) = 32;

UPDATE package p
SET summary_hash = sha512(convert_to(s.value, 'UTF8'))
FROM strings s
WHERE p.summary_hash = s.id
  AND octet_length(summary_hash) = 32;

DELETE FROM strings
WHERE octet_length(id) = 32;

DROP INDEX IF EXISTS package_description_hash_idx;
DROP INDEX IF EXISTS package_summary_hash_idx;