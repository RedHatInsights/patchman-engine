#!/bin/bash

set -o pipefail

TEST_DIRS=${1:-./...}

# Run go test and colorize output (PASS - green, FAIL - red).
# Set "-p 1" to run test sequentially to avoid parallel changes in testing database.
go test -v -p 1 -coverprofile=coverage.txt -covermode=atomic $TEST_DIRS \
  | sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/''
