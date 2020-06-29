CREATE TABLE IF NOT EXISTS package
(
    id           INT  NOT NULL,
    name         TEXT NOT NULL CHECK ( NOT empty(name)),
    last_version TEXT NOT NULL CHECK ( NOT empty(last_version)),
    description  TEXT NOT NULL CHECK ( NOT empty(description)),
    summary      TEXT NOT NULL CHECK ( NOT empty(summary))
);

CREATE TABLE IF NOT EXISTS system_package
(
    system_id         INT  NOT NULL REFERENCES system_platform,
    package_id        INT  NOT NULL REFERENCES package,
    version_installed TEXT NOT NULL CHECK ( NOT empty(version_installed) ),
    -- Use null to represent up-to-date packages
    update_data       JSONB DEFAULT NULL
)