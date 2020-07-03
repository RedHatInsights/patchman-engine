CREATE TABLE system_tags
(
    tag       TEXT NOT NULL CHECK ( NOT EMPTY(tag)),
    system_id INT  NOT NULL REFERENCES system_platform ON DELETE CASCADE,
    PRIMARY KEY (system_id, tag)
);

CREATE INDEX system_tags_idx on system_tags (tag);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_tags to listener;
GRANT SELECT ON system_tags to evaluator;
GRANT SELECT ON system_tags to manager;
GRANT SELECT ON system_tags to vmaas_sync;

