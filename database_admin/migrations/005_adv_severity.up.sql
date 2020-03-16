CREATE TABLE advisory_severity
(
    id   INT  NOT NULL,
    name TEXT NOT NULL UNIQUE CHECK ( not empty(name) ),
    PRIMARY KEY (id)
);

INSERT INTO advisory_severity (id, name)
VALUES (1, 'Low'),
       (2, 'Moderate'),
       (3, 'Important'),
       (4, 'Critical')
ON CONFLICT DO NOTHING;

ALTER TABLE advisory_metadata
    ADD COLUMN severity_id INT;

ALTER TABLE advisory_metadata
    ADD CONSTRAINT advisory_severity_id
        FOREIGN KEY (severity_id) REFERENCES advisory_severity (id);

GRANT SELECT ON TABLE advisory_severity TO evaluator;
GRANT SELECT ON TABLE advisory_severity TO listener;
GRANT SELECT ON TABLE advisory_severity TO manager;
GRANT SELECT ON TABLE advisory_severity TO vmaas_sync;
