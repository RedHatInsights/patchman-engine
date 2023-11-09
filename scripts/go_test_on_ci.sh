#!/bin/bash

set -e -o pipefail

# Analyse generated docs/v2/openapi.json
# FIXME: temporary disable openapi check, swagger converter returns 500
# ./scripts/check-openapi-docs.sh

# Check dockerfiles and docker-composes consistency
./scripts/check-dockercomposes.sh

# Check if all env variables have defined value
./scripts/check-deploy-envs.sh

# Analyse code using lint
golangci-lint run --timeout 5m
echo "Go code analysed successfully."

# Run project tests
./scripts/go_test_db.sh | ./scripts/colorize.sh
