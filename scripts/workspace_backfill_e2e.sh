#!/usr/bin/env bash
# End-to-end local run: migrate DB, load system_inventory test data, clear workspace
# columns, run workspace_backfill job once, verify.
set -e -o pipefail

cd "$(dirname "$0")/.."

psql_admin() {
  PGPASSWORD="${DB_ADMIN_PASSWD:-passwd}" psql \
    -h "${DB_HOST:-db}" \
    -p "${DB_PORT:-5432}" \
    -U "${DB_ADMIN_USER:-admin}" \
    -d "${DB_NAME:-patchman}" \
    "$@"
}

echo "==> Waiting for PostgreSQL"
export WAIT_FOR_EMPTY_DB=1
./dev/scripts/wait-for-services.sh true

echo "==> Running migrations"
go run main.go migrate file://./database_admin/migrations

echo "==> Waiting for migrations to finish"
unset WAIT_FOR_EMPTY_DB
export WAIT_FOR_FULL_DB=1
./dev/scripts/wait-for-services.sh true
unset WAIT_FOR_FULL_DB

echo "==> Loading system_inventory test data"
psql_admin -f dev/workspace_backfill/test_generate_system_inventory.sql

echo "==> Clearing workspace_id / workspace_name (triggers disabled per transaction)"
psql_admin -f dev/workspace_backfill/prepare_workspace_backfill_test.sql

echo "==> Running workspace_backfill job"
./scripts/entrypoint.sh job workspace_backfill

echo "==> Verifying backfill"
mapfile -t _wb_counts < <(psql_admin -t -A -f dev/workspace_backfill/verify_workspace_backfill.sql)
pending="${_wb_counts[0]}"
mismatched="${_wb_counts[1]}"

echo "pending=${pending} mismatched=${mismatched}"

if [[ "${pending}" != "0" || "${mismatched}" != "0" ]]; then
  echo "Workspace backfill e2e verification failed" >&2
  exit 1
fi

echo "Workspace backfill e2e finished successfully"
