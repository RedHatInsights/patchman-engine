#!/bin/bash

set -o pipefail

export TEST_WD=`pwd`

TEST_DIRS=${1:-./...}

# Run go test and colorize output (PASS - green, FAIL - red).
# Set "-p 1" to run test sequentially to avoid parallel changes in testing database.
go test $BUILD_TAGS_ENV -v -p 1 -coverprofile=coverage.txt -covermode=atomic $TEST_DIRS
