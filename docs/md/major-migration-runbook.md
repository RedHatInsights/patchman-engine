# Major database migration runbook

Operational guide for deploying schema migrations that run heavy DDL (for example large `ALTER TABLE` on partitioned tables). 

See also [database.md — Migrations](database.md#migrations) for config reference.

---

## How deploy works

```
New deploy triggered
    ↓
db-migration Job starts (completions: 1, parallelism: 1)
    ‖  (in parallel)
New app pods start → check-for-db init polls schema every 5s (up to ~5 min)
    ↓
Job: advisory lock → block users → [terminate sessions] → MigrateUp (DDL)
    ↓
Job succeeds → check-for-db init passes → rollout continues
Job fails   → new pods fail init → old pods keep serving
```

- **One migrator per deploy** — only the Job runs `migrate`;
- **Job limits** — `MIGRATION_TIMEOUT` (default 7200s / 2h), `MIGRATION_MAX_RETRIES=3` with 5s sleep between attempts (`database_admin/entrypoint.sh`).
- **Advisory lock** — `pg_advisory_lock(123)` ensures a single migration process even if something else triggers database-admin.

---

## When to use this runbook

Use for migrations that need exclusive locks or long DDL runtime. Routine migrations (new tables, additive columns, typical indexes) follow the normal deploy; do **not** set `terminate_db_sessions` by default.

---

## Before deploy

1. **Review the migration** — identify tables that need `ACCESS EXCLUSIVE` locks and expected runtime.
2. **Set target schema** (if not migrating to latest):
   ```
   DATABASE_ADMIN_CONFIG=schema_migration=161
   ```
   on the **db-migration Job** only (via app-interface / ClowdApp `DATABASE_ADMIN_CONFIG`).
3. **Major DDL only** — enable session termination:
   ```
   DATABASE_ADMIN_CONFIG=terminate_db_sessions=true
   ```
   Can be combined: `terminate_db_sessions=true;schema_migration=161`
4. **Communicate** — brief app errors are expected while sessions are terminated and during DDL; clients reconnect after `LOGIN` is restored.
5. **Optional** — scale down listener/evaluator if a previous deploy showed DDL blocked by lingering connections even with the flag.

---

## `DATABASE_ADMIN_CONFIG` flags

Set on the **db-migration Job** via `DATABASE_ADMIN_CONFIG` (passed as `POD_CONFIG`). Multiple keys are semicolon-separated, e.g. `terminate_db_sessions=true;schema_migration=161`.

Config keys are defined in `database_admin/config.go`. ClowdApp comments in `deploy/clowdapp.yaml` may use older names (`schema_version`, `force_schema_version`) — the code keys are `schema_migration` and `force_migration_version`.

### `schema_migration`

| | |
|---|---|
| **Config key** | `schema_migration` (integer, default `-1`) |
| **Where** | `DATABASE_ADMIN_CONFIG` on the db-migration Job |
| **Effect** | Target schema version to migrate to. `-1` means latest available migration file. Values `>= 0` migrate only up to that version. Also used by `check-for-db` / `migrateAction` to decide whether deployment should proceed. |

**Set when:** you need to pin or cap the migration version (stage validation, staged rollout, or blocking auto-upgrade past a known-good version).

**Leave at `-1` when:** normal production deploy should apply all pending migrations.

**Note:** If current DB version equals `schema_migration` but newer migration files exist, deploy is **blocked** until `schema_migration` is raised — intentional safety gate.

### `force_migration_version`

| | |
|---|---|
| **Config key** | `force_migration_version` (integer, default `-1`, inactive when `<= 0`) |
| **Where** | `DATABASE_ADMIN_CONFIG` on the db-migration Job |
| **Effect** | Before `MigrateUp`, calls `migrate.Force(version)` — sets `schema_migrations.version` and clears `dirty`. Used to recover from a failed migration left in dirty state. Migration then continues per `schema_migration`. |

**Set when:** `schema_migrations.dirty = true` after a failed migration and engineering/DBA has confirmed it is safe to reset the version marker (and any partial DDL has been handled).

**Leave unset when:** schema is clean (`dirty = false`). Misuse can mark a broken schema as valid.

### `reset_schema`

| | |
|---|---|
| **Config key** | `reset_schema` (boolean, default `false`) |
| **Where** | `DATABASE_ADMIN_CONFIG` on the db-migration Job |
| **Effect** | `DROP SCHEMA public CASCADE` and recreate empty `public` schema before migration logic runs. **Destructive** — wipes all application data. |

**Set when:** local/dev database rebuild only, or explicit empty-environment bootstrap under controlled conditions.

**Never set in production** unless performing a deliberate full data reset with sign-off.

### `update_users`

| | |
|---|---|
| **Config key** | `update_users` (boolean, default `false`) |
| **Where** | `DATABASE_ADMIN_CONFIG` (db-migration Job; also common in local `conf/database_admin.env`) |
| **Effect** | Runs `create_users.sql`, then after migration sets passwords for `listener`, `evaluator`, `manager`, `vmaas_sync` from environment variables. |

**Set when:** initial environment setup or refreshing DB role definitions/passwords (typical in local docker and first-time deploy).

**Leave off when:** users already exist and passwords are managed separately — normal prod Job runs usually rely on this being set only where needed in app-interface.

### `unlock_users`

| | |
|---|---|
| **Config key** | `unlock_users` (boolean, default `false`) |
| **Where** | `DATABASE_ADMIN_CONFIG` on the db-migration Job |
| **Effect** | `ALTER USER … LOGIN` for app users **before** migration, without running DDL. Recovery helper if a previous migration left users at `NOLOGIN`. |

**Set when:** app users are stuck at `NOLOGIN` after an aborted migration and you need to restore login without running a full migrate.

**Leave off for normal deploys** — migration flow blocks and unblocks users automatically.

### `update_db_config`

| | |
|---|---|
| **Config key** | `update_db_config` (boolean, default `false`) |
| **Where** | `DATABASE_ADMIN_CONFIG` (db-migration Job; also in local `conf/database_admin.env`) |
| **Effect** | Re-runs `database_admin/config.sql` (PostgreSQL settings such as `work_mem` for the application). |

**Set when:** applying or refreshing DB-level settings from `config.sql` after deploy.

**Leave off when:** only schema migration is needed.

### `terminate_db_sessions`

| | |
|---|---|
| **Config key** | `terminate_db_sessions` (boolean, default `false`) |
| **Where** | `DATABASE_ADMIN_CONFIG` on the **db-migration Job** only |
| **Effect** | After `NOLOGIN` on app users, runs `pg_terminate_backend` on open `listener` / `evaluator` / `manager` / `vmaas_sync` sessions, then waits until `pg_stat_activity` is clear |

**Enable when:** heavy DDL, prior stuck migration after “Blocking writing users”, or planned maintenance window.

**Leave off when:** routine release, local/CI, no session-blocking symptoms.

**Remove after** the major migration deploy completes.

`NOLOGIN` alone does not close existing connections — that is why this flag exists.

---

## During deploy

### Where to watch logs

Kibana — filter by log stream and message text (field names vary by environment; adjust `@log_stream` as needed):

```kql
@log_stream: patchman-* and message: *advisory lock*
```

Migration progress:

```kql
@log_stream: patchman-* and (message: "Migrating the database" or message: "Starting schema migration" or message: "App database sessions cleared")
```

Init containers polling for schema (may appear on manager/listener/evaluator streams):

```kql
@log_stream: patchman-* and message: *DB migration in progress*
```

### Expected log sequence (db-migration Job)

| Step | Log line | Notes |
|------|----------|--------|
| 1 | `Getting advisory lock` | |
| 2 | `Advisory lock acquired` | **Missing** → another holder of advisory lock 123 |
| 3 | `Migrating the database` | |
| 4 | `Blocking writing users during the migration` | `NOLOGIN` on app DB users |
| 5 | `Terminating active app database sessions` | Only if `terminate_db_sessions=true` |
| 6 | `Terminated session pid=... user=...` | Per terminated backend |
| 7 | `Waiting for N sessions: ...` | Repeats each second until drain |
| 8 | `App database sessions cleared` | |
| 9 | `Starting schema migration to version X` | DDL begins |
| 10 | *(silence)* | Normal during long DDL |
| 11 | `Reverting components privileges` | `LOGIN` restored |
| 12 | `Releasing advisory lock` | |

### If stuck

| Last log seen | Likely cause | Action |
|---------------|--------------|--------|
| Only `Getting advisory lock` | Another process holds advisory lock 123 | See [Advisory lock diagnostics](#advisory-lock-diagnostics); check for duplicate migration Job or stale pod |
| `Waiting for N sessions` (repeating) | App connections still open | Enable or verify `terminate_db_sessions=true`; scale down listener/evaluator; inspect `pg_stat_activity` |
| Past `Starting schema migration`, long silence | DDL waiting on table lock | Find blockers on target table; scale down apps; see [DDL lock diagnostics](#ddl-lock-diagnostics) |
| `failed to check app database sessions after 5 attempts` | DB connectivity or permissions on `pg_stat_activity` | Fix admin DB access; do not ignore — migration aborted intentionally |
| Job failed, new pods `CrashLoopBackOff` on init | Migration failed or timed out | Old pods still serve; fix migration state before retrying |

---

## After deploy

1. Verify schema: `SELECT version, dirty FROM schema_migrations;` — `dirty` must be `false`.
2. Remove `terminate_db_sessions` from `DATABASE_ADMIN_CONFIG` (or set `false`).
3. Confirm app pods passed `check-for-db` and are ready.
4. Smoke-test manager API and a sample evaluation path if the migration touched core tables.

---

## Rollback

- **Application rollback** — deploy previous image tag; if schema already migrated forward, old code may be incompatible with new schema. Coordinate with engineering before rolling back app only.
- **Failed migration (`dirty = true`)** — do not re-deploy blindly. Inspect `schema_migrations`, Job logs, and whether DDL partially applied. May require `force_migration_version` (see `database_admin/config.go`) under DBA/engineering guidance.
- **Stuck advisory lock** — identify holder PID; terminate only after confirming it is a stale migration pod, not an active legitimate migration.

---

## Postgres diagnostics

### Advisory lock diagnostics

Advisory lock id **123** is hardcoded in `database_admin/update.go`.

```sql
-- Who holds advisory lock 123?
SELECT l.pid, a.usename, a.state, a.application_name, left(a.query, 120) AS query
FROM pg_locks l
JOIN pg_stat_activity a ON a.pid = l.pid
WHERE l.locktype = 'advisory'
  AND l.classid = 0
  AND l.objid = 123;
```

### App session diagnostics

```sql
-- Open sessions for patchman app users
SELECT pid, usename, state, wait_event_type, wait_event, left(query, 80) AS query
FROM pg_stat_activity
WHERE usename IN ('listener', 'evaluator', 'manager', 'vmaas_sync')
ORDER BY usename, pid;
```

### DDL lock diagnostics

Replace `system_inventory` with the table your migration touches. `blocked_locks` is the waiting lock (typically the db-migration DDL); `blocking_locks` is the granted lock on the same resource from another session. The JOIN already matches them on the same `relation`, so filter on `blocked_locks.relation`:

```sql
SELECT blocked.pid     AS blocked_pid,
       blocked.usename AS blocked_user,
       left(blocked.query, 80) AS blocked_query,
       blocking.pid    AS blocking_pid,
       blocking.usename AS blocking_user,
       left(blocking.query, 80) AS blocking_query
FROM pg_stat_activity blocked
JOIN pg_locks blocked_locks ON blocked_locks.pid = blocked.pid AND NOT blocked_locks.granted
JOIN pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
  AND blocking_locks.database IS NOT DISTINCT FROM blocked_locks.database
  AND blocking_locks.relation IS NOT DISTINCT FROM blocked_locks.relation
  AND blocking_locks.page IS NOT DISTINCT FROM blocked_locks.page
  AND blocking_locks.tuple IS NOT DISTINCT FROM blocked_locks.tuple
  AND blocking_locks.virtualxid IS NOT DISTINCT FROM blocked_locks.virtualxid
  AND blocking_locks.transactionid IS NOT DISTINCT FROM blocked_locks.transactionid
  AND blocking_locks.classid IS NOT DISTINCT FROM blocked_locks.classid
  AND blocking_locks.objid IS NOT DISTINCT FROM blocked_locks.objid
  AND blocking_locks.objsubid IS NOT DISTINCT FROM blocked_locks.objsubid
  AND blocking_locks.pid != blocked_locks.pid
JOIN pg_stat_activity blocking ON blocking.pid = blocking_locks.pid
WHERE blocking_locks.granted
  AND blocked_locks.relation = 'system_inventory'::regclass;
```

### Migration state

```sql
SELECT version, dirty FROM schema_migrations;
```

---

## Job parameters

| Parameter | Default | Where | Purpose |
|-----------|---------|-------|---------|
| `MIGRATION_TIMEOUT` | `7200` | ClowdApp Job `activeDeadlineSeconds` | Max Job runtime (seconds) |
| `MIGRATION_MAX_RETRIES` | `3` | db-migration Job env | Migrate command retries on failure (`entrypoint.sh`, 5s between attempts) |

---

## Related code and deploy files

| Topic | Location |
|-------|----------|
| Migration flow, session wait/terminate | `database_admin/update.go` |
| Migrate retries | `database_admin/entrypoint.sh` |
| Init schema poll | `database_admin/check-upgraded.sh` |
| ClowdApp Job and `check-for-db` init | `deploy/clowdapp.yaml` |
