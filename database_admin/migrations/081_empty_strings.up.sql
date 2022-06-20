ALTER TABLE advisory_metadata ALTER COLUMN solution DROP NOT NULL;

UPDATE advisory_metadata SET solution = NULL WHERE empty(solution);
UPDATE advisory_metadata SET url = NULL WHERE empty(url);
UPDATE baseline SET description = NULL WHERE empty(description);
UPDATE strings SET value = '(empty)' WHERE empty(value);
UPDATE system_platform SET json_checksum = NULL WHERE empty(json_checksum);
UPDATE system_platform SET vmaas_json = NULL WHERE empty(vmaas_json);

ALTER TABLE advisory_metadata ADD CHECK (NOT empty(solution));
ALTER TABLE advisory_metadata ADD CHECK (NOT empty(url));
ALTER TABLE baseline ADD CHECK (NOT empty(description));
ALTER TABLE strings ADD CHECK (NOT empty(value));
ALTER TABLE system_package ADD CHECK (NOT empty(latest_evra));
ALTER TABLE system_platform ADD CHECK (NOT empty(json_checksum));
ALTER TABLE system_platform ADD CHECK (NOT empty(vmaas_json));
