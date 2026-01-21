-- INSTEAD OF triggers for system_platform view to handle INSERT and UPDATE

CREATE OR REPLACE FUNCTION system_platform_insert()
    RETURNS TRIGGER AS
$system_platform_insert$
DECLARE
    new_system_id BIGINT;
    created TIMESTAMPTZ := CURRENT_TIMESTAMP;
BEGIN
    INSERT INTO system_inventory (
        inventory_id,
        rh_account_id,
        vmaas_json,
        json_checksum,
        last_updated,
        unchanged_since,
        last_upload,
        stale_timestamp,
        stale_warning_timestamp,
        culled_timestamp,
        stale,
        display_name,
        reporter_id,
        yum_updates,
        satellite_managed,
        built_pkgcache,
        yum_checksum,
        arch,
        bootc,
        tags,
        created
    ) VALUES (
        NEW.inventory_id,
        NEW.rh_account_id,
        NEW.vmaas_json,
        NEW.json_checksum,
        NEW.last_updated,
        NEW.unchanged_since,
        NEW.last_upload,
        NEW.stale_timestamp,
        NEW.stale_warning_timestamp,
        NEW.culled_timestamp,
        COALESCE(NEW.stale, false),
        NEW.display_name,
        NEW.reporter_id,
        NEW.yum_updates,
        COALESCE(NEW.satellite_managed, false),
        COALESCE(NEW.built_pkgcache, false),
        NEW.yum_checksum,
        NEW.arch,
        COALESCE(NEW.bootc, false),
        '[]'::JSONB,
        created
    )
    RETURNING id INTO new_system_id;

    INSERT INTO system_patch (
        system_id,
        rh_account_id,
        last_evaluation,
        installable_advisory_count_cache,
        installable_advisory_enh_count_cache,
        installable_advisory_bug_count_cache,
        installable_advisory_sec_count_cache,
        packages_installed,
        packages_installable,
        packages_applicable,
        third_party,
        applicable_advisory_count_cache,
        applicable_advisory_enh_count_cache,
        applicable_advisory_bug_count_cache,
        applicable_advisory_sec_count_cache,
        template_id
    ) VALUES (
        new_system_id,
        NEW.rh_account_id,
        NEW.last_evaluation,
        COALESCE(NEW.installable_advisory_count_cache, 0),
        COALESCE(NEW.installable_advisory_enh_count_cache, 0),
        COALESCE(NEW.installable_advisory_bug_count_cache, 0),
        COALESCE(NEW.installable_advisory_sec_count_cache, 0),
        COALESCE(NEW.packages_installed, 0),
        COALESCE(NEW.packages_installable, 0),
        COALESCE(NEW.packages_applicable, 0),
        COALESCE(NEW.third_party, false),
        COALESCE(NEW.applicable_advisory_count_cache, 0),
        COALESCE(NEW.applicable_advisory_enh_count_cache, 0),
        COALESCE(NEW.applicable_advisory_bug_count_cache, 0),
        COALESCE(NEW.applicable_advisory_sec_count_cache, 0),
        NEW.template_id
    );

    NEW.id := new_system_id;
    RETURN NEW;
END;
$system_platform_insert$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION system_platform_update()
    RETURNS TRIGGER AS
$system_platform_update$
BEGIN
    UPDATE system_inventory SET
        inventory_id = NEW.inventory_id,
        vmaas_json = NEW.vmaas_json,
        json_checksum = NEW.json_checksum,
        last_updated = NEW.last_updated,
        unchanged_since = NEW.unchanged_since,
        last_upload = NEW.last_upload,
        stale_timestamp = NEW.stale_timestamp,
        stale_warning_timestamp = NEW.stale_warning_timestamp,
        culled_timestamp = NEW.culled_timestamp,
        stale = NEW.stale,
        display_name = NEW.display_name,
        reporter_id = NEW.reporter_id,
        yum_updates = NEW.yum_updates,
        satellite_managed = NEW.satellite_managed,
        built_pkgcache = NEW.built_pkgcache,
        yum_checksum = NEW.yum_checksum,
        arch = NEW.arch,
        bootc = NEW.bootc
    WHERE id = OLD.id AND rh_account_id = OLD.rh_account_id;

    UPDATE system_patch SET
        last_evaluation = NEW.last_evaluation,
        installable_advisory_count_cache = NEW.installable_advisory_count_cache,
        installable_advisory_enh_count_cache = NEW.installable_advisory_enh_count_cache,
        installable_advisory_bug_count_cache = NEW.installable_advisory_bug_count_cache,
        installable_advisory_sec_count_cache = NEW.installable_advisory_sec_count_cache,
        packages_installed = NEW.packages_installed,
        packages_installable = NEW.packages_installable,
        packages_applicable = NEW.packages_applicable,
        third_party = NEW.third_party,
        applicable_advisory_count_cache = NEW.applicable_advisory_count_cache,
        applicable_advisory_enh_count_cache = NEW.applicable_advisory_enh_count_cache,
        applicable_advisory_bug_count_cache = NEW.applicable_advisory_bug_count_cache,
        applicable_advisory_sec_count_cache = NEW.applicable_advisory_sec_count_cache,
        template_id = NEW.template_id
    WHERE system_id = OLD.id AND rh_account_id = OLD.rh_account_id;

    RETURN NEW;
END;
$system_platform_update$
    LANGUAGE 'plpgsql';

CREATE TRIGGER system_platform_insert_trigger
    INSTEAD OF INSERT ON system_platform
    FOR EACH ROW
    EXECUTE FUNCTION system_platform_insert();

CREATE TRIGGER system_platform_update_trigger
    INSTEAD OF UPDATE ON system_platform
    FOR EACH ROW
    EXECUTE FUNCTION system_platform_update();
