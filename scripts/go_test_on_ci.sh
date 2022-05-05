#!/bin/bash

set -e -o pipefail

# Analyse generated docs/openapi.json
./scripts/check-openapi-docs.sh

# Check dockerfiles and docker-composes consistency
./scripts/check-dockercomposes.sh

# Analyse code using lint
build_tags=""
if [[ -n $BUILD_TAGS_ENV ]]; then
    build_tags="--build-tags dynamic"
fi
golangci-lint run $build_tags --timeout 5m
echo "Go code analysed successfully."

# Run project tests
./scripts/go_test_db.sh | ./scripts/colorize.sh
