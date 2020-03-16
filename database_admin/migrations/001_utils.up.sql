-- ---------------------------------------------------------------------------
-- Utility functions
-- ---------------------------------------------------------------------------
-- empty
CREATE OR REPLACE FUNCTION empty(t TEXT)
    RETURNS BOOLEAN as
$empty$
BEGIN
    RETURN t ~ '^[[:space:]]*$';
END;
$empty$
    LANGUAGE 'plpgsql';

-- set_first_reported
CREATE OR REPLACE FUNCTION set_first_reported()
    RETURNS TRIGGER AS
$set_first_reported$
BEGIN
    IF NEW.first_reported IS NULL THEN
        NEW.first_reported := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$set_first_reported$
    LANGUAGE 'plpgsql';

-- set_last_updated
CREATE OR REPLACE FUNCTION set_last_updated()
    RETURNS TRIGGER AS
$set_last_updated$
BEGIN
    IF (TG_OP = 'UPDATE') OR
       NEW.last_updated IS NULL THEN
        NEW.last_updated := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$set_last_updated$
    LANGUAGE 'plpgsql';

-- check_unchanged
CREATE OR REPLACE FUNCTION check_unchanged()
    RETURNS TRIGGER AS
$check_unchanged$
BEGIN
    IF (TG_OP = 'INSERT') AND
       NEW.unchanged_since IS NULL THEN
        NEW.unchanged_since := CURRENT_TIMESTAMP;
    END IF;
    IF (TG_OP = 'UPDATE') AND
       NEW.json_checksum <> OLD.json_checksum THEN
        NEW.unchanged_since := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$check_unchanged$
    LANGUAGE 'plpgsql';