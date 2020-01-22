#!/bin/bash

set -o pipefail
set -u

# wait untill database is ready
./scripts/wait-for-services.sh

# Normal test run - everything except database schema test
TEST_DIRS=$(go list ./... | grep -v "app/database")

# Run database test, destroys and recreates database
go test -v app/database

# fill database with testing data
echo $DB_PASSWD | psql -h $DB_HOST -d $DB_NAME -U $DB_USER -p $DB_PORT \
                       -v ON_ERROR_STOP=1 \
                       -a -q -f ./scripts/test_data.sql

# run tests
./scripts/go_test.sh "${TEST_DIRS}"
