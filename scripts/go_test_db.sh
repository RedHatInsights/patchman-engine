#!/bin/bash

set -e -o pipefail
MIGRATION_FILES=file://./database_admin/migrations

go run ./scripts/feed_db.go inventory_hosts

# Create database
go run main.go migrate $MIGRATION_FILES

# Run database test, destroys and recreates database
gotestsum --format=standard-verbose -- -v app/database_admin

# Fill database with testing data
go run ./scripts/feed_db.go feed

# Normal test run - everything except database schema test
TEST_DIRS=$(go list -buildvcs=false ./... | grep -v "app/database_admin")
./scripts/go_test.sh "${TEST_DIRS}"
