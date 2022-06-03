#!/bin/bash

set -e -o pipefail # stop on error

MIGRATION_FILES=file://./database_admin/migrations

echo "Running in $(pwd) as $(id)"
if [[ -n $GORUN ]]; then
  go run main.go migrate $MIGRATION_FILES
else
  ./main migrate $MIGRATION_FILES
fi

exec sleep infinity
