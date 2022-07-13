#!/bin/bash

set -e -o pipefail # stop on error

MIGRATION_FILES=file://./database_admin/migrations

echo "Running in $(pwd) as $(id)"
${GORUN:+go run} ./main${GORUN:+.go} migrate $MIGRATION_FILES
