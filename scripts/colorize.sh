#!/bin/bash

# Colorize some keywords (e.g. PASS - green, FAIL - red) in stdin.
# Example: echo This PASS, this FAIL | ./scripts/colorize.sh

sed "s/PASS/\x1b[32mPASS\x1b[0m/
     s/FAIL/\x1b[31mFAIL\x1b[0m/"
