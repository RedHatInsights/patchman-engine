ALTER TABLE system_platform ADD COLUMN IF NOT EXISTS yum_checksum TEXT CHECK (NOT empty(yum_checksum));

CREATE OR REPLACE FUNCTION check_unchanged()
    RETURNS TRIGGER AS
$check_unchanged$
BEGIN
    IF (TG_OP = 'INSERT') AND
       NEW.unchanged_since IS NULL THEN
        NEW.unchanged_since := CURRENT_TIMESTAMP;
    END IF;
    IF (TG_OP = 'UPDATE') AND
       (NEW.json_checksum <> OLD.json_checksum OR NEW.yum_checksum <> OLD.yum_checksum) THEN
        NEW.unchanged_since := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$check_unchanged$
    LANGUAGE 'plpgsql';
