# AGENTS.md

## Quick Start for AI Agents

**patchman-engine** is the backend for Red Hat Insights patch management: a REST API and background workers that evaluate which advisories and package updates apply to registered systems. See [README.md](README.md) for user-facing setup.

### Authoritative Documentation

| Topic | Location |
|--------|----------|
| Architecture and components | [docs/md/architecture.md](docs/md/architecture.md) |
| Database layout and migrations | [docs/md/database.md](docs/md/database.md) |
| Major migration operations runbook | [docs/md/major-migration-runbook.md](docs/md/major-migration-runbook.md) |
| Local dev, tests, OpenAPI | [README.md](README.md) |
| Commits, PRs, contribution style | [CONTRIBUTING.md](CONTRIBUTING.md) |

Prefer these sources over guessing when behavior or schema matters.

### Code Discipline

- Match existing **Go style**, package layout, and patterns in the touched area.
- Keep changes **minimal** and scoped to the task; avoid drive-by refactors.
- When touching the REST surface or request/response types, consider **OpenAPI** regeneration and any contract consumers.

### Where to Look in the Tree

| Area | Typical paths |
|------|----------------|
| HTTP REST API | `manager/` |
| External messages | `listener/`, topic names in code and `conf/` |
| Evaluation | `evaluator/`, topic names in code and `conf/` |
| Advisory sync | `tasks/vmaas_sync/` |
| Migrations | `database_admin/migrations/` (verify naming against existing migrations) |
| Migration flow, session flags, ops runbook | `database_admin/update.go`, [docs/md/database.md#migrations](docs/md/database.md#migrations), [docs/md/major-migration-runbook.md](docs/md/major-migration-runbook.md) |
| Database schema and SQL | `database_admin/schema/` |
| Containers and local orchestration | `docker-compose.yml`, `docker-compose.test.yml`, `Dockerfile*` |
| Scheduled jobs | `tasks/` |
| Admin REST API | `turnpike/` |
| Platform services mock for development and testing | `platform/` |

---

## Communication Flows

### System Upload Flow
```
Inventory Upload
    ↓
[platform.inventory.events] Kafka Topic
    ↓
Listener Component
    ↓ (updates system_inventory, system_patch, system_repo)
    ↓
[patchman.evaluator.upload] Kafka Topic
    ↓
Evaluator-Upload Component
    ↓ (calls VMaaS /updates)
    ↓ (updates system_advisories, advisory_account_data)
    ↓
[platform.notifications.ingress] (optional)
[platform.remediation-updates.patch] (optional)
[platform.inventory.host-apps] (optional)
```

### Advisory Sync Flow
```
Scheduled Job Trigger
    ↓
VMaaS Sync Component
    ↓ (calls VMaaS /errata, /pkglist, /repos)
    ↓ (updates advisory_metadata, package, repo)
    ↓
[patchman.evaluator.recalc] Kafka Topic
    ↓
Evaluator-Recalc Component
    ↓ (bulk re-evaluation of affected systems)
    ↓
Database Updates
```

### User Query Flow
```
User Request
    ↓
Manager Component (REST API)
    ↓ (queries PostgreSQL)
    ↓
Response to User
```

---

## Database migrations (major DDL)

When advising on migrations or deploy config, use [docs/md/database.md#migrations](docs/md/database.md#migrations) for overview and [docs/md/major-migration-runbook.md](docs/md/major-migration-runbook.md) for ops procedure, troubleshooting, and flag reference.

**Deploy model:** One **db-migration** Job per deploy runs migrations; app pods only **check-for-db** init (poll schema). Failed migration → new pods fail init, old pods keep serving.

**Session handling:** Before DDL, database-admin sets app users (`listener`, `evaluator`, `manager`, `vmaas_sync`) to `NOLOGIN`, optionally terminates lingering backends (`terminate_db_sessions`), polls `pg_stat_activity` until clear (`waitForSessionClosed` in `database_admin/update.go`; fails after 5 consecutive query errors — does not proceed silently), then runs DDL and restores `LOGIN`. `NOLOGIN` stops new connections but does not close existing ones.

**`terminate_db_sessions`:** Default **off** (`false`); normal deploys must stay unchanged. When enabled on the **db-migration Job only** via `DATABASE_ADMIN_CONFIG=terminate_db_sessions=true`, runs `pg_terminate_backend` on open app-user sessions, then waits again until `pg_stat_activity` is clear. Remove after deploy. Do not enable on manager/listener/evaluator pods. Other flags (`schema_migration`, `force_migration_version`, etc.) are documented in the runbook.

**Recommend `terminate_db_sessions=true` only when:**

- The migration is a **major DDL** change likely to need exclusive locks or long runtimes (large `ALTER TABLE`, partition restructuring, similar)
- Migration logs show blocking after user lock with app sessions still present
- Ops are doing a **one-off major migration deploy** via `DATABASE_ADMIN_CONFIG` on the **db-migration Job**

**Do not recommend it when:**

- The change is a routine migration or standard release deploy
- The user is working locally or in CI
- There is no session-lock symptom — it forcibly drops client connections and is not a safe default

**Logging:** Key lines — `Advisory lock acquired`, `Waiting for N sessions`, `App database sessions cleared`, `Starting schema migration to version X`. Stuck at only `Getting advisory lock` → advisory lock 123 held elsewhere. Use `message:` filters in Kibana, not `kubernetes.container_name`. Full log sequence in the runbook.

**When advising users:** Point to the runbook for before/during/after steps, Kibana queries, and Postgres diagnostics. Deploy layout (single migration Job, `check-for-db` init) is in `deploy/clowdapp.yaml`.
