#!/bin/bash

set -e -o pipefail # stop on error

MIGRATION_DIR=file://./database_admin/migrations

echo "Running in $(pwd) as $(id)"
${GORUN:+go run} ./main${GORUN:+.go} check_upgraded $MIGRATION_DIR
