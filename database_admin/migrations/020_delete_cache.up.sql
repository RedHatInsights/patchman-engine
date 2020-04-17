CREATE TABLE IF NOT EXISTS deleted_system
(
    inventory_id TEXT                     NOT NULL,
    CHECK (NOT empty(inventory_id)),
    when_deleted TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE (inventory_id)
) TABLESPACE pg_default;

CREATE INDEX ON deleted_system (when_deleted);

GRANT SELECT, INSERT, UPDATE, DELETE ON deleted_system TO listener;
GRANT SELECT ON TABLE deleted_system TO evaluator;
GRANT SELECT ON TABLE deleted_system TO manager;
GRANT SELECT ON TABLE deleted_system TO vmaas_sync;
