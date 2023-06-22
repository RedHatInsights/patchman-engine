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
	id uuid PRIMARY KEY,
	account character varying(10),
	display_name character varying(200) NOT NULL,
	tags jsonb NOT NULL,
	updated timestamp with time zone NOT NULL,
	created timestamp with time zone NOT NULL,
	stale_timestamp timestamp with time zone NOT NULL,
	system_profile jsonb NOT NULL,
	insights_id uuid,
	reporter character varying(255) NOT NULL,
	per_reporter_staleness jsonb NOT NULL,
	org_id character varying(36),
	groups jsonb
);

DELETE FROM inventory.hosts_v1_0;

CREATE INDEX IF NOT EXISTS hosts_v1_0_tags_index ON inventory.hosts_v1_0 USING GIN (tags JSONB_PATH_OPS);
CREATE INDEX IF NOT EXISTS hosts_v1_0_insights_reporter_index ON inventory.hosts_v1_0 (reporter);
CREATE INDEX IF NOT EXISTS hosts_v1_0_stale_timestamp_index ON inventory.hosts_v1_0 USING btree (stale_timestamp);
CREATE INDEX IF NOT EXISTS hosts_v1_0_groups_index ON inventory.hosts_v1_0 USING GIN (groups JSONB_PATH_OPS);

CREATE OR REPLACE VIEW inventory.hosts AS SELECT
	id,
	account,
	display_name,
	created,
	updated,
	stale_timestamp,
	stale_timestamp + INTERVAL '1' DAY * '7'::double precision AS stale_warning_timestamp,
	stale_timestamp + INTERVAL '1' DAY * '14'::double precision AS culled_timestamp,
	tags,
	system_profile,
	insights_id,
	reporter,
	per_reporter_staleness,
	org_id,
	groups
FROM inventory.hosts_v1_0;

GRANT SELECT ON TABLE inventory.hosts TO cyndi_reader;
