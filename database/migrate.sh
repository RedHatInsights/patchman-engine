#!/bin/bash

# Install the migrate tool
#go get -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate

# Allow running migrate either from system installation or local download
PATH=.:$PATH

set -u
# Require environment variables to be present
DB_URL="postgres://${DB_USER}:${DB_PASSWD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

args=$@
ls -allh .

# Run the migrations up to latest
migrate \
  -source file://database/migrations \
  -database $DB_URL ${args:-up}
