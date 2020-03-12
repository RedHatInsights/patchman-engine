#!/bin/bash

# Colorize some keywords (e.g. PASS - green, FAIL - red) in stdin.
# Example: echo This PASS, this FAIL | ./scripts/colorize.sh

cat /dev/stdin | \
sed ''/PASS/s//$(printf "\033[32mPASS\033[0m")/'' | \
sed ''/FAIL/s//$(printf "\033[31mFAIL\033[0m")/''
