# Workspace backfill — local testing

Backfill copies `workspaces` JSON into `workspace_id` and `workspace_name` on `system_inventory` via the `workspace_backfill` job.

**Do not run** `docker-compose.test.yml` and `docker-compose.workspace-backfill.yml` at the same time (both use host port **5433**).

## Files

| File | Purpose |
|------|---------|
| [`docker-compose.workspace-backfill.yml`](../../docker-compose.workspace-backfill.yml) | `db` + optional one-shot e2e runner |
| [`conf/workspace_backfill.env`](../../conf/workspace_backfill.env) | Local Docker `POD_CONFIG` (**1000** rows per job run) |
| [`test_generate_system_inventory.sql`](test_generate_system_inventory.sql) | Fast load: accounts + inventory + patch only |
| [`prepare_workspace_backfill_test.sql`](prepare_workspace_backfill_test.sql) | Clear `workspace_id` / `workspace_name` without triggers |
| [`verify_workspace_backfill.sql`](verify_workspace_backfill.sql) | Check pending / mismatched rows |
| [`scripts/workspace_backfill_e2e.sh`](../../scripts/workspace_backfill_e2e.sh) | Automated pipeline: setup + **one** job run + verify |

## Configuration (`POD_CONFIG`)

| Key | Local Docker ([`conf/workspace_backfill.env`](../../conf/workspace_backfill.env)) | Code / production default |
|-----|----------------------------------------------------------------------------------|---------------------------|
| `workspace_backfill_batch_size` | `1000` | `1000` |
| `workspace_backfill_max_rows_per_run` | **`1000`** | **`50000`** ([`tasks/config.go`](../../tasks/config.go), [`deploy/clowdapp.yaml`](../../deploy/clowdapp.yaml)) |
| `workspace_backfill_batch_sleep_ms` | `0` | `0` |

Local compose loads `workspace_backfill.env` so each job invocation updates at most **1000** rows. Re-run the job manually until logs say `Workspace backfill complete`.

Override per run: `docker compose run --rm -e 'POD_CONFIG=...' workspace-backfill ...`

## Manual workflow (recommended for batched local runs)

### 1. Run DB

```bash
docker compose -f docker-compose.workspace-backfill.yml up -d db
```

Wait for Postgres:

```bash
until PGPASSWORD=passwd psql -h localhost -p 5433 -U admin -d patchman -c 'SELECT 1' >/dev/null 2>&1; do sleep 1; done
```

### 2. Setup once

Migrates, loads test data, clears workspace columns. **Destructive** (`TRUNCATE rh_account CASCADE` in the generator).

```bash
docker compose -f docker-compose.workspace-backfill.yml run --rm workspace-backfill bash -c '
set -e -o pipefail
cd /go/src/app
export WAIT_FOR_EMPTY_DB=1
./dev/scripts/wait-for-services.sh true
go run main.go migrate file://./database_admin/migrations
unset WAIT_FOR_EMPTY_DB
export WAIT_FOR_FULL_DB=1
./dev/scripts/wait-for-services.sh true
unset WAIT_FOR_FULL_DB
PGPASSWORD=passwd psql -h db -p 5432 -U admin -d patchman -f dev/workspace_backfill/test_generate_system_inventory.sql
PGPASSWORD=passwd psql -h db -p 5432 -U admin -d patchman -f dev/workspace_backfill/prepare_workspace_backfill_test.sql
'
```

Default generator creates **7500** systems. Lower `_const` in `test_generate_system_inventory.sql` first if you want a smaller dataset (e.g. `500` systems).

### 3. Run job (repeat until complete)

Each run processes up to **1000** rows (local env):

```bash
docker compose -f docker-compose.workspace-backfill.yml run --rm workspace-backfill \
  ./scripts/entrypoint.sh job workspace_backfill
```

Logs:

- `Workspace backfill paused (per-run limit); more rows remain` — run step 3 again
- `Workspace backfill complete` — done

Check pending count:

```bash
PGPASSWORD=passwd psql -h localhost -p 5433 -U admin -d patchman -c \
  "SELECT count(*) AS pending FROM system_inventory WHERE workspace_id IS NULL AND workspaces IS NOT NULL;"
```

Verify when `pending` is 0:

```bash
PGPASSWORD=passwd psql -h localhost -p 5433 -U admin -d patchman -f dev/workspace_backfill/verify_workspace_backfill.sql
```

Expect `pending = 0` and `mismatched = 0`.

### Teardown

```bash
docker compose -f docker-compose.workspace-backfill.yml down
```

## One-shot e2e (automated script)

Runs setup, **one** job invocation (local limit 1000 rows), then verification:

```bash
docker compose -f docker-compose.workspace-backfill.yml up --build --abort-on-container-exit
```

With the default generator (**7500** systems), verification fails after a single job run because only 1000 rows are backfilled. Use either:

- **≤1000** systems in `_const` before e2e, or  
- The **manual workflow** above (setup once, job repeated), or  
- A one-off higher limit for a full single-run test only:

```bash
docker compose -f docker-compose.workspace-backfill.yml run --rm \
  -e 'POD_CONFIG=update_users;update_db_config;wait_for_db=empty;use_testing_db;workspace_backfill_batch_size=1000;workspace_backfill_max_rows_per_run=10000000' \
  workspace-backfill ./scripts/workspace_backfill_e2e.sh
```

## Database user

The job connects as the **admin** user (`core.ConfigureAdminApp()`), same credentials as migrations (`database.adminUsername` / `adminPassword` from Clowder). That is required for `SET LOCAL session_replication_role = replica` and matches local testing via `cdappconfig.json`.

## Production

CronJob `workspace-backfill` in [`deploy/clowdapp.yaml`](../../deploy/clowdapp.yaml) uses `WORKSPACE_BACKFILL_CONFIG` with `workspace_backfill_max_rows_per_run=50000` (suspended by default). Run on a schedule until pending rows are gone. Admin DB credentials are provided by the platform (Clowder), not component secrets.
