#!/bin/bash

set -o pipefail

MIGRATION_FILES=file://./database_admin/migrations
DATABASE_URL="postgres://$DB_HOST/$DB_NAME?sslmode=$DB_SSLMODE${DB_SSLROOTCERT:+&sslrootcert=$DB_SSLROOTCERT}"

if [[ -n $GORUN ]]; then
  go run main.go migrate $MIGRATION_FILES $DATABASE_URL
else
  ./main migrate $MIGRATION_FILES $DATABASE_URL
fi
