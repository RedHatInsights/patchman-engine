CREATE OR REPLACE FUNCTION create_table_partitions(tbl regclass, parts INTEGER)
    RETURNS VOID AS
$$
DECLARE
    I INTEGER;
BEGIN
    I := 0;
    WHILE I < parts
        LOOP
            EXECUTE 'CREATE TABLE ' || text(tbl) || '_' || text(I) || ' PARTITION OF ' || text(tbl) ||
                    ' FOR VALUES WITH ' || ' ( MODULUS ' || text(parts) || ', REMAINDER ' || text(I) || ');';
            I = I + 1;
        END LOOP;
END;
$$ LANGUAGE plpgsql;

ALTER TABLE system_advisories
    RENAME TO system_advisories_old;

CREATE TABLE system_advisories
(
    system_id      INT                      NOT NULL,
    advisory_id    INT                      NOT NULL,
    first_reported TIMESTAMP WITH TIME ZONE NOT NULL,
    when_patched   TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    status_id      INT                      DEFAULT 0,
    PRIMARY KEY (system_id, advisory_id),
    CONSTRAINT system_platform_id
        FOREIGN KEY (system_id)
            REFERENCES system_platform (id),
    CONSTRAINT advisory_metadata_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id),
    CONSTRAINT status_id
        FOREIGN KEY (status_id)
            REFERENCES status (id)
) PARTITION BY HASH (system_id);

SELECT create_table_partitions('system_advisories', 32);

INSERT INTO system_advisories
SELECT system_id, advisory_id, first_reported, when_patched, status_id
FROM system_advisories_old;

DROP TABLE system_advisories_old;
