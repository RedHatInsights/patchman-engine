# Database

## Tables
Main database tables description:
- **system_inventory** — Partitioned table for the registered host / inventory profile: internal `id`, Insights `inventory_id`, `rh_account_id`, `vmaas_json` (packages, repos, modules for VMaaS), `yum_updates` and related checksums, staleness and culling timestamps, `display_name`, OS fields, tags, workspace fields, and workload flags. **`system_repo`** (and similar link tables) use this internal `id` as the system key. The **listener** upserts rows here and relies on **system_inventory** for upload locks and unchanged detection; the **evaluator** reads it via a join to **system_patch**.
- **system_patch** — Partitioned evaluation output for each system, keyed by `rh_account_id` and `system_id` where `system_id` equals **system_inventory.id** on the same account. Holds advisory and package count caches, `last_evaluation`, `third_party`, `template_id`, and related aggregates. Rows are created or updated by the **listener** together with **system_inventory**; the **evaluator** persists evaluation results here (not into a single legacy table).
- **advisory_metadata** - stores info about advisories (`description`, `summary`, `solution` etc.). It's synced and stored on trigger by `vmaas_sync` component. It allows to display detail information about the advisory.
- **system_advisories** - stores info about advisories evaluated for particular systems (system - advisory M-N mapping table). `system_id` references **system_inventory.id**. Contains info when system advisory was firstly reported and patched (if so). Records are created and updated by `evaluator` component. It allows to display list of advisories related to a system.
- **advisory_account_data** - stores info about all advisories detected within at least one system that belongs to a given account. So it provides overall statistics about system advisories displayed by the application.
- **account_advisory** - workspace-scoped version of `advisory_account_data`. Stores per-advisory aggregate counts (`systems_applicable`, `systems_installable`) and notification state for each workspace within an account. Keyed by `(rh_account_id, workspace_id, advisory_id)`, partitioned by `rh_account_id` (32 partitions).
- **package_name** - names of the packages installed on systems
- **package** - list of all packages versions, precisely all EVRAs (epoch-version-release-arch)
- **system_package2** - list of packages installed on a system
- **template** - Content Sources template metadata per account (`rh_account_id`, `id`, `uuid`, `name`, `environment_id`,
  `arch`, `version`, etc.). Upserted by **listener** from `platform.content-sources.template` events.
- **template_advisory** - junction table linking a **template** to **advisory_metadata** rows
  (`rh_account_id`, `template_id`, `advisory_id`). Populated by **listener** on `template-updated` from Content Sources.
  Read by **evaluator** (when `template_advisory_eval=true`) to determine which advisories are installable for systems
  assigned via **`system_patch.template_id`**. The same flag on **listener** triggers re-evaluation when rows change.

## Schema
The ERD image below may lag `database_admin/schema/create_schema.sql`; for systems it may not reflect the split between **system_inventory** (host profile / upload payload) and **system_patch** (evaluation caches and aggregates).

![](graphics/db_diagram.png)

## Migrations

Schema changes live in `database_admin/migrations/` and are applied by **database-admin** (`database_admin/update.go`).

In production:

- A single **db-migration** ClowdApp Job runs `migrate` once per deploy (`completions: 1`, `parallelism: 1`).
- Manager, listener, evaluator, and other components use a **check-for-db** init container that polls until the schema matches (`database_admin/check-upgraded.sh`).

Before DDL, database-admin blocks app database users from new logins and waits for existing sessions to drain:

1. `ALTER USER … NOLOGIN` for `listener`, `evaluator`, `manager`, `vmaas_sync`
2. Optionally `pg_terminate_backend` on remaining app sessions when `terminate_db_sessions=true`
3. Poll `pg_stat_activity` until no app-user sessions remain
4. Run `MigrateUp`
5. `ALTER USER … LOGIN` to restore access

`NOLOGIN` stops **new** connections but does **not** close existing ones. Lingering sessions can hold locks and block DDL on large or sensitive migrations.

| Topic | Document |
|-------|----------|
| Major DDL deploy procedure, troubleshooting, SQL diagnostics | [major-migration-runbook.md](major-migration-runbook.md) |
| `DATABASE_ADMIN_CONFIG` flags (including `terminate_db_sessions`) and log sequence | [major-migration-runbook.md](major-migration-runbook.md) |
| ClowdApp parameters | `deploy/clowdapp.yaml`, `database_admin/config.go` |
