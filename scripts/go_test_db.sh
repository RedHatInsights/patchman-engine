#!/bin/bash

set -e

# wait untill database is ready
./scripts/wait-for-services.sh

# Normal test run - everything except database schema test
TEST_DIRS=$(go list ./... | grep -v "app/database")

# Run database test, destroys and recreates database
go test -v app/database

# fill database with testing data
./scripts/feed_db.sh

# run tests
./scripts/go_test.sh "${TEST_DIRS}"
