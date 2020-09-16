CREATE TABLE reporter
(
    id   INT  NOT NULL,
    name TEXT NOT NULL UNIQUE CHECK ( not empty(name) ),
    PRIMARY KEY (id)
);

INSERT INTO reporter (id, name)
VALUES (1, 'puptoo'),
       (2, 'rhsm-conduit'),
       (3, 'yupana')
ON CONFLICT DO NOTHING;

ALTER TABLE system_platform
    ADD COLUMN reporter_id INT;

ALTER TABLE system_platform
    ADD CONSTRAINT reporter_id
        FOREIGN KEY (reporter_id) REFERENCES reporter (id);

GRANT SELECT ON TABLE reporter TO evaluator;
GRANT SELECT ON TABLE reporter TO listener;
GRANT SELECT ON TABLE reporter TO manager;
GRANT SELECT ON TABLE reporter TO vmaas_sync;
