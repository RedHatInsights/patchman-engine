#!/bin/bash

set -e -o pipefail

# Analyse generated docs/openapi.json
./scripts/check-openapi-docs.sh

# Check dockerfiles and docker-composes consistency
./scripts/check-dockercomposes.sh

# Analyse code using lint
golangci-lint run --timeout 5m
echo "Go code analysed successfully."

# Run project tests
./scripts/go_test_db.sh | ./scripts/colorize.sh
