CREATE TABLE IF NOT EXISTS hosts
(
    id       INTEGER PRIMARY KEY,
    request  VARCHAR                     NOT NULL,
    checksum VARCHAR                     NOT NULL,
    updated  TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

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
