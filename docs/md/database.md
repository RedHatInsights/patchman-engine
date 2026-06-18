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

## Schema
The ERD image below may lag `database_admin/schema/create_schema.sql`; for systems it may not reflect the split between **system_inventory** (host profile / upload payload) and **system_patch** (evaluation caches and aggregates).

![](graphics/db_diagram.png)

## Migrations

Schema changes live in `database_admin/migrations/` and are applied by the **database-admin** component (`database_admin/update.go`). In production, a single **db-migration** ClowdApp Job runs migrations; other pods wait in a `check-for-db` init container until the schema matches.

### Pre-migration session handling

Before running DDL, database-admin blocks app database users from new logins and waits for existing sessions to drain:

1. `ALTER USER … NOLOGIN` for `listener`, `evaluator`, `manager`, `vmaas_sync`
2. Optionally (see below) `pg_terminate_backend` on remaining app sessions
3. Poll `pg_stat_activity` until no app-user sessions remain
4. Run `MigrateUp`
5. `ALTER USER … LOGIN` to restore access

`NOLOGIN` stops **new** connections but does **not** close existing ones. Lingering sessions can hold locks and block DDL on large or sensitive migrations.

### `terminate_db_sessions` flag

| | |
|---|---|
| **Config key** | `terminate_db_sessions` (boolean, default `false`) |
| **Where to set** | `DATABASE_ADMIN_CONFIG` / `POD_CONFIG` on the db-migration Job only |
| **Example** | `terminate_db_sessions=true` |

When enabled, database-admin calls `pg_terminate_backend` on all open sessions for the four app users above (excluding its own connection), then waits again until `pg_stat_activity` is clear.

**Set `terminate_db_sessions=true` when:**

- The migration runs heavy or long-held DDL (e.g. `ALTER TABLE` on large partitioned tables, structural changes that need exclusive locks)
- A previous migration appeared stuck after “Blocking writing users” with app sessions still in `pg_stat_activity`
- Operations explicitly plan a major migration deploy and want to force-close stale app connections

**Leave unset (default `false`) when:**

- Routine deploys and normal migrations (additive columns, new tables, typical index changes)
- Local development, CI, and test runs
- There is no evidence of session-related blocking — the flag forcibly disconnects clients and should not be the default

Remove the flag after the major migration deploy completes; subsequent deploys should not need it.

### Migration log sequence

When the db-migration Job runs, expect these log lines in order (Kibana: `@log_stream: patchman-*` and `kubernetes.container_name: db-migration`):

1. `Getting advisory lock`
2. `Advisory lock acquired` — if missing, another process holds advisory lock 123
3. `Migrating the database`
4. `Blocking writing users during the migration`
5. `Terminating active app database sessions` / `Terminated session pid=...` — only when `terminate_db_sessions=true`
6. `Waiting for N sessions: ...` — repeats until sessions drain
7. `App database sessions cleared`
8. `Starting schema migration to version X`
9. Silence during DDL (normal)
10. `Reverting components privileges`
11. `Releasing advisory lock`

### Other `DATABASE_ADMIN_CONFIG` options

See `deploy/clowdapp.yaml` parameters and `database_admin/config.go`: `schema_migration`, `force_migration_version`, `reset_schema`, `update_users`, `unlock_users`, `update_db_config`.
