-- Create "inventory.hosts" for testing purposes. In deployment it's created by remote Cyndi service.
CREATE SCHEMA IF NOT EXISTS inventory;

DO
$$
    BEGIN
        -- The admin ROLE that allows the inventory schema to be managed
        CREATE ROLE cyndi_admin;
        -- The reader ROLE that provides SELECT access to the inventory.hosts view
        CREATE ROLE cyndi_reader;
    EXCEPTION
        WHEN DUPLICATE_OBJECT THEN NULL;
    END
$$;

CREATE TABLE IF NOT EXISTS inventory.hosts_v1_0 (
    id uuid NOT NULL,
    insights_id uuid,
    account character varying(10) NOT NULL,
    display_name character varying(200) NOT NULL,
    tags jsonb NOT NULL,
    updated timestamp with time zone NOT NULL,
    created timestamp with time zone NOT NULL,
    stale_timestamp timestamp with time zone NOT NULL,
    system_profile jsonb NOT NULL,
    PRIMARY KEY (id)
);

DELETE FROM inventory.hosts_v1_0;

CREATE INDEX IF NOT EXISTS hosts_v1_0_account_index ON inventory.hosts_v1_0 USING btree (account);
CREATE INDEX IF NOT EXISTS hosts_v1_0_display_name_index ON inventory.hosts_v1_0 USING btree (display_name);
CREATE INDEX IF NOT EXISTS hosts_v1_0_stale_timestamp_index ON inventory.hosts_v1_0 USING btree (stale_timestamp);
CREATE INDEX IF NOT EXISTS hosts_v1_0_system_profile_index ON inventory.hosts_v1_0 USING gin (system_profile jsonb_path_ops);
CREATE INDEX IF NOT EXISTS hosts_v1_0_tags_index ON inventory.hosts_v1_0 USING gin (tags jsonb_path_ops);

CREATE OR REPLACE VIEW inventory.hosts AS
 SELECT hosts_v1_0.id,
    hosts_v1_0.insights_id,
    hosts_v1_0.account,
    hosts_v1_0.display_name,
    hosts_v1_0.created,
    hosts_v1_0.updated,
    hosts_v1_0.stale_timestamp,
    (hosts_v1_0.stale_timestamp + ('1 day'::interval day * '7'::double precision)) AS stale_warning_timestamp,
    (hosts_v1_0.stale_timestamp + ('1 day'::interval day * '14'::double precision)) AS culled_timestamp,
    hosts_v1_0.tags,
    hosts_v1_0.system_profile
 FROM inventory.hosts_v1_0;

GRANT SELECT ON TABLE inventory.hosts TO cyndi_reader;
