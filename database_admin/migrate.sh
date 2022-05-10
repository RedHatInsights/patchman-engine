#!/bin/bash

set -o pipefail

MIGRATION_FILES=file://./database_admin/migrations
DATABASE_URL="postgres://$DB_HOST/$DB_NAME?sslmode=$DB_SSLMODE${DB_SSLROOTCERT:+&sslrootcert=$DB_SSLROOTCERT}"

echo "Blocking writing users during the migration"
psql -c "ALTER USER listener NOLOGIN"
psql -c "ALTER USER evaluator NOLOGIN"
psql -c "ALTER USER vmaas_sync NOLOGIN"
./scripts/wait-for-sessions-closed.sh

if [[ -n $GORUN ]]; then
  go run main.go migrate $MIGRATION_FILES $DATABASE_URL
else
  ./main migrate $MIGRATION_FILES $DATABASE_URL
fi

echo "Reverting components privileges"
psql -c "ALTER USER listener LOGIN"
psql -c "ALTER USER evaluator LOGIN"
psql -c "ALTER USER vmaas_sync LOGIN"
