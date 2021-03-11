#!/bin/bash

set -e -o pipefail

# Analyse generated docs/openapi.json
./scripts/check-openapi-docs.sh

# Analyse dockerfiles
./scripts/check-dockerfiles.sh

# Analyse code using lint
golangci-lint run --timeout 5m
echo "Go code analysed successfully."

# Run project tests
./scripts/go_test_db.sh | ./scripts/colorize.sh

# if [ -n "$TRAVIS" ]; then
bash <(curl -s https://codecov.io/bash)
# fi
