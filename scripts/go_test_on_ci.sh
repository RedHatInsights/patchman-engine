#!/bin/bash

set -e

./scripts/go_test_db.sh

if [ -n "$TRAVIS" ]; then
  bash <(curl -s https://codecov.io/bash)
fi
