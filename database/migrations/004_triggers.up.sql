CREATE TRIGGER system_platform_set_first_reported
    BEFORE INSERT
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE set_first_reported();

CREATE TRIGGER system_platform_set_last_updated
    BEFORE INSERT OR UPDATE
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE set_last_updated();

CREATE TRIGGER system_platform_check_unchanged
    BEFORE INSERT OR UPDATE
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE check_unchanged();

CREATE TRIGGER system_advisories_set_first_reported
    BEFORE INSERT
    ON system_advisories
    FOR EACH ROW
EXECUTE PROCEDURE set_first_reported();
