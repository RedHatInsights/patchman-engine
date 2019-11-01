#!/bin/bash

# ./go_test.sh

set -o pipefail

# Run go test and colorize output (PASS - green, FAIL - red).
go test -v ./app/... | sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/''
