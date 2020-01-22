#!/bin/bash

set -e

# Analyse code using lint
/go/bin/golangci-lint run
echo "Go code analysed successfully."

# Run project tests
./scripts/go_test_db.sh

if [ -n "$TRAVIS" ]; then
  bash <(curl -s https://codecov.io/bash)
fi
