#!/bin/bash

set -e -o pipefail

# Wait untill database is ready
./scripts/wait-for-services.sh

# Run database test, destroys and recreates database
go test -v app/database

# Fill database with testing data
./scripts/feed_db.sh

# Normal test run - everything except database schema test
TEST_DIRS=$(go list ./... | grep -v "app/database")
./scripts/go_test.sh "${TEST_DIRS}"
