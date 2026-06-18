# AGENTS.md

## Quick Start for AI Agents

**patchman-engine** is the backend for Red Hat Insights patch management: a REST API and background workers that evaluate which advisories and package updates apply to registered systems. See [README.md](README.md) for user-facing setup.

### Authoritative Documentation

| Topic | Location |
|--------|----------|
| Architecture and components | [docs/md/architecture.md](docs/md/architecture.md) |
| Database layout and migrations | [docs/md/database.md](docs/md/database.md) |
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
| Migration flow and session flags | `database_admin/update.go`, [docs/md/database.md#migrations](docs/md/database.md#migrations) |
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

## Database migrations: `terminate_db_sessions`

When advising on migrations or deploy config, use [docs/md/database.md#migrations](docs/md/database.md#migrations). Summary for agents:

**Default:** do **not** set `terminate_db_sessions`. It defaults to `false`; normal deploys must stay unchanged.

**What it does:** After `NOLOGIN` on app DB users, database-admin optionally runs `pg_terminate_backend` on open `listener` / `evaluator` / `manager` / `vmaas_sync` sessions, then waits until `pg_stat_activity` shows none, then runs DDL. Code: `prepareForMigration()` in `database_admin/update.go`.

**Recommend `terminate_db_sessions=true` only when:**

- The migration is a **major DDL** change likely to need exclusive locks or long runtimes (large `ALTER TABLE`, partition restructuring, similar)
- Migration logs show blocking after user lock with app sessions still present
- Ops are doing a **one-off major migration deploy** via `DATABASE_ADMIN_CONFIG` on the **db-migration Job**

**Do not recommend it when:**

- The change is a routine migration or standard release deploy
- The user is working locally or in CI
- There is no session-lock symptom — it forcibly drops client connections and is not a safe default

**How to set (production):** `DATABASE_ADMIN_CONFIG=terminate_db_sessions=true` on the db-migration Job for that deploy only; remove afterward. Do not enable on manager/listener/evaluator pods.

**Related:** Session wait logic and `pg_stat_activity` queries are in `database_admin/update.go`. Deploy layout (single migration Job, `check-for-db` init) is in `deploy/clowdapp.yaml`. Expected migration log sequence (advisory lock → sessions cleared → DDL start) is in [docs/md/database.md#migration-log-sequence](docs/md/database.md#migration-log-sequence).
