#!/bin/bash

set -e -o pipefail
MIGRATION_FILES=file://./database_admin/migrations

# Create database
go run main.go migrate $MIGRATION_FILES

# Run database test, destroys and recreates database
go test -v app/database_admin

# Fill database with testing data
WAIT_FOR_DB=full go run ./scripts/feed_db.go

# Normal test run - everything except database schema test
TEST_DIRS=$(go list ./... | grep -v "app/database_admin")
./scripts/go_test.sh "${TEST_DIRS}"
