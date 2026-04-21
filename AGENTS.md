# AGENTS.md

## Quick Start for AI Agents

**patchman-engine** is the backend for Red Hat Insights patch management: a REST API and background workers that evaluate which advisories and package updates apply to registered systems. See [README.md](README.md) for user-facing setup.

### Authoritative Documentation

| Topic | Location |
|--------|----------|
| Architecture and components | [docs/md/architecture.md](docs/md/architecture.md) |
| Database layout | [docs/md/database.md](docs/md/database.md) |
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
