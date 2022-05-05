#!/bin/bash

set -e -o pipefail

# Create database
./database_admin/update.sh

# Run database test, destroys and recreates database
go test $BUILD_TAGS_ENV -v app/database_admin

# Fill database with testing data
./scripts/feed_db.sh

# Normal test run - everything except database schema test
TEST_DIRS=$(go list ./... | grep -v "app/database_admin")
./scripts/go_test.sh "${TEST_DIRS}"
