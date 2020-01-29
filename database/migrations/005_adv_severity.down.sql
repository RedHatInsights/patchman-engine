ALTER TABLE advisory_metadata
    DROP CONSTRAINT advisory_severity_id;

ALTER TABLE advisory_metadata
    DROP COLUMN severity_id;

DROP TABLE advisory_severity;