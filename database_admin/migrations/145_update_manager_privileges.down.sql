REVOKE UPDATE ON system_inventory FROM manager;
GRANT UPDATE ON system_inventory (stale) TO manager;

REVOKE UPDATE ON system_patch FROM manager;
GRANT UPDATE ON system_patch (
    installable_advisory_count_cache,
    installable_advisory_enh_count_cache,
    installable_advisory_bug_count_cache,
    installable_advisory_sec_count_cache,
    applicable_advisory_count_cache,
    applicable_advisory_enh_count_cache,
    applicable_advisory_bug_count_cache,
    applicable_advisory_sec_count_cache,
    template_id) TO manager;
