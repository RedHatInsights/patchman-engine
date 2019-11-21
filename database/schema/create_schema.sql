create table if not exists hosts
(
    id       integer primary key,
    request  varchar                     not null,
    checksum varchar                     not null,
    updated  TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO hosts
VALUES (1, '{"req":"pkg1"}', 'abcd');
INSERT INTO hosts
VALUES (2, '{"req":"pkg2"}', 'efgh');

-- set_last_updated
CREATE OR REPLACE FUNCTION set_last_updated()
    RETURNS TRIGGER AS
$set_last_updated$
BEGIN
    IF (TG_OP = 'UPDATE' OR TG_OP = 'INSERT') OR
       NEW.updated IS NULL THEN
        NEW.updated := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$set_last_updated$
    LANGUAGE 'plpgsql';


CREATE TRIGGER hosts_last_updated
    BEFORE INSERT OR UPDATE
    ON hosts
    FOR EACH ROW
EXECUTE PROCEDURE set_last_updated();