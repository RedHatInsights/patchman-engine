#!/bin/bash

set -e -o pipefail

MIGRATION_FILES=file://./database_admin/migrations
MAX_RETRIES=${MIGRATION_MAX_RETRIES:-1}

run_migration() {
  ${GORUN:+go run} ./main${GORUN:+.go} migrate $MIGRATION_FILES
}

echo "Running in $(pwd) as $(id)"
for attempt in $(seq 1 "$MAX_RETRIES"); do
  echo "Migration attempt ${attempt}/${MAX_RETRIES}"
  if run_migration; then
    exit 0
  fi
  if [ "$attempt" -lt "$MAX_RETRIES" ]; then
    echo "Migration failed, retrying..."
    sleep 5
  fi
done
echo "Migration failed after ${MAX_RETRIES} attempts"
exit 1
